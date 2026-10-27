package scraper

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"tibia-nemesis-api/internal/config"
	"tibia-nemesis-api/internal/models"

	"github.com/PuerkitoBio/goquery"
)

// Special data-sort-values in the guildstats.eu "Possibility" column.
// Positive values are the spawn chance in percent.
const (
	guildStatsLow          = -1 // "Low"
	guildStatsNoChance     = -2 // "No"
	guildStatsNoPrediction = -3 // empty cell
)

// guildStatsAliases maps guildstats.eu boss names (lower-case) to the names
// used in bosses_metadata.yaml. Bosses that guildstats tracks as one entry but
// the metadata splits by location get the same data copied to every variant.
var guildStatsAliases = map[string][]string{
	"albino dragons":        {"Albino Dragon"},
	"crustaceae giganticae": {"Crustacea Gigantica"},
	"draptors":              {"Draptor"},
	"midnight panthers":     {"Midnight Panther"},
	"undead cavebears":      {"Undead Cavebear"},
	"yetis":                 {"Yeti"},
	"battlemaster zunzu":    {"Battlemaster Zunzu", "Battlemaster Zunzu (West)", "Battlemaster Zunzu (Middle)", "Battlemaster Zunzu (East)"},
	"dreadmaw":              {"Dreadmaw", "Dreadmaw (West)", "Dreadmaw (East)"},
	"flamecaller zazrak":    {"Flamecaller Zazrak", "Flamecaller Zazrak (Mountain)", "Flamecaller Zazrak (Dojo)"},
	"fleabringer":           {"Fleabringer", "Fleabringer (North)", "Fleabringer (South)", "Fleabringer (Surface)"},
	"hirintror":             {"Hirintror", "Hirintror (Nibelor)", "Hirintror (Mines)"},
	"the voice of ruin":     {"The Voice of Ruin", "Voice of Ruin (Ghastly)", "The Voice of Ruin (Middle)"},
}

// GuildStatsScraper scrapes the boss list on guildstats.eu.
type GuildStatsScraper struct {
	cfg   config.Config
	known map[string]bool // lower-case metadata boss names
	now   func() time.Time
}

func NewGuildStats(cfg config.Config, knownBosses []string) Scraper {
	known := make(map[string]bool, len(knownBosses))
	for _, name := range knownBosses {
		known[strings.ToLower(name)] = true
	}
	return &GuildStatsScraper{cfg: cfg, known: known, now: time.Now}
}

func (g *GuildStatsScraper) Source() models.DataSource {
	return models.DataSource{Name: "GuildStats", URL: "https://guildstats.eu"}
}

func (g *GuildStatsScraper) Fetch(world string) ([]models.SpawnChance, error) {
	pageURL := "https://guildstats.eu/bosses?world=" + url.QueryEscape(world)

	body, err := fetchPage(pageURL)
	if err != nil {
		return nil, err
	}
	return g.parseSpawnChances(world, body)
}

// parseSpawnChances reads the boss table. It keeps bosses that have a
// prediction, plus bosses without one that are listed in the metadata.
func (g *GuildStatsScraper) parseSpawnChances(world, html string) ([]models.SpawnChance, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	table := doc.Find("table.sortable-table").First()
	if table.Length() == 0 {
		return nil, errors.New("guildstats: boss table not found")
	}

	// Locate columns by header text so added or reordered columns don't break parsing
	cols := map[string]int{}
	table.Find("thead th").Each(func(i int, th *goquery.Selection) {
		cols[strings.ToLower(strings.TrimSpace(th.Text()))] = i
	})
	nameCol, okName := cols["boss name"]
	lastSeenCol, okLastSeen := cols["last seen"]
	possCol, okPoss := cols["possibility"]
	if !okName || !okLastSeen || !okPoss {
		return nil, fmt.Errorf("guildstats: missing expected columns (found %v)", cols)
	}

	tz, err := time.LoadLocation(g.cfg.TZ)
	if err != nil {
		tz = time.Local
	}
	today := g.now().In(tz)

	var result []models.SpawnChance
	now := time.Now().UTC()

	table.Find("tbody tr").Each(func(i int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() <= possCol {
			return
		}

		rawName := cleanHTMLText(tds.Eq(nameCol).Text())
		if rawName == "" {
			return
		}

		possValue, _ := tds.Eq(possCol).Attr("data-sort-value")
		poss, err := strconv.ParseFloat(strings.TrimSpace(possValue), 64)
		if err != nil {
			poss = guildStatsNoPrediction
		}

		names := guildStatsAliases[strings.ToLower(rawName)]
		if names == nil {
			names = []string{rawName}
		}

		if poss == guildStatsNoPrediction && !g.anyKnown(names) {
			return
		}

		var percent *int
		isNoChance := false
		switch {
		case poss == guildStatsNoChance:
			isNoChance = true
		case poss > 0:
			p := int(math.Round(poss))
			if p == 0 {
				p = 1 // keep small chances distinguishable from "No"
			}
			percent = &p
		}
		// guildStatsLow and guildStatsNoPrediction leave percent nil

		days := daysSince(strings.TrimSpace(tds.Eq(lastSeenCol).Text()), today)

		for _, name := range names {
			result = append(result, models.SpawnChance{
				World:         world,
				Name:          name,
				Percent:       percent,
				DaysSinceKill: days,
				IsNoChance:    isNoChance,
				UpdatedAt:     now,
			})

			logMsg := fmt.Sprintf("scraper: %s - %s:", world, name)
			switch {
			case percent != nil:
				logMsg += fmt.Sprintf(" %d%%", *percent)
			case isNoChance:
				logMsg += " no chance"
			case poss == guildStatsLow:
				logMsg += " low chance"
			}
			if days != nil {
				logMsg += fmt.Sprintf(" (%d days)", *days)
			}
			log.Print(logMsg)
		}
	})

	if len(result) == 0 {
		return nil, errors.New("guildstats: no bosses parsed")
	}

	log.Printf("scraper: parsed %d bosses for %s", len(result), world)
	return result, nil
}

func (g *GuildStatsScraper) anyKnown(names []string) bool {
	for _, name := range names {
		if g.known[strings.ToLower(name)] {
			return true
		}
	}
	return false
}

// daysSince returns the number of calendar days between a "2006-01-02" date
// and today, or nil if the date is missing ("-") or unparseable.
func daysSince(date string, today time.Time) *int {
	seen, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil
	}
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	d := int(todayDate.Sub(seen).Hours() / 24)
	if d < 0 {
		d = 0
	}
	return &d
}
