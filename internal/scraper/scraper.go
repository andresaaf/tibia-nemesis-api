package scraper

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tibia-nemesis-api/internal/config"
	"tibia-nemesis-api/internal/models"

	"github.com/PuerkitoBio/goquery"
)

var (
	rowRE           = regexp.MustCompile(`(?s)<tr[^>]*id="boss-[^"]+"[^>]*>(.*?)</tr>`)
	nameRE          = regexp.MustCompile(`(?s)class="boss-name-link"[^>]*>\s*(.*?)\s*</a>`)
	daysTextRE      = regexp.MustCompile(`(?s)class\s*=\s*"days-text"[^>]*>\s*(\d{1,4})\s*day(?:s)?(?:\s+ago)?`)
	chancePercentRE = regexp.MustCompile(`(?s)class\s*=\s*"chance-percentage[^"]*"[^>]*>\s*\((\d{1,3})%\)`)
	noChanceRE      = regexp.MustCompile(`(?s)class\s*=\s*"chance-text[^"]*"[^>]*>\s*No\s+Chance`)
	htmlTagRE       = regexp.MustCompile(`<[^>]+>`)
	whitespaceRE    = regexp.MustCompile(`\s+`)
)

type Scraper interface {
	Fetch(world string) ([]models.SpawnChance, error)
}

type WebScraper struct {
	cfg config.Config
}

func New(cfg config.Config) Scraper {
	return &WebScraper{cfg: cfg}
}

func (w *WebScraper) Fetch(world string) ([]models.SpawnChance, error) {
	return w.fetchHTML(world)
}

func (w *WebScraper) fetchHTML(world string) ([]models.SpawnChance, error) {
	url := fmt.Sprintf("https://www.tibia-statistic.com/bosshunter/details/%s", strings.ToLower(world))

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "TibiaNemesisAPI/1.0")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("scraper: fetch failed for %s: %v", url, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("scraper: HTTP %d for %s", resp.StatusCode, url)
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	html, err := doc.Html()
	if err != nil {
		return nil, err
	}

	return w.parseSpawnChances(world, html)
}

func (w *WebScraper) parseSpawnChances(world, html string) ([]models.SpawnChance, error) {
	var result []models.SpawnChance
	now := time.Now().UTC()

	rows := rowRE.FindAllStringSubmatch(html, -1)
	for _, match := range rows {
		if len(match) < 2 {
			continue
		}
		row := match[1]

		// Extract boss name
		nameMatch := nameRE.FindStringSubmatch(row)
		if len(nameMatch) < 2 {
			continue
		}
		name := cleanHTMLText(nameMatch[1])

		// Check if this is a "No Chance" boss - skip these
		if noChanceRE.MatchString(row) {
			continue
		}

		// Extract days since last kill (optional)
		var days *int
		if daysMatch := daysTextRE.FindStringSubmatch(row); len(daysMatch) >= 2 {
			if d, err := strconv.Atoi(daysMatch[1]); err == nil {
				days = &d
			}
		}

		// Extract percentage (optional - some bosses in "without prediction" section won't have this)
		var percent *int
		if percentMatch := chancePercentRE.FindStringSubmatch(row); len(percentMatch) >= 2 {
			if p, err := strconv.Atoi(percentMatch[1]); err == nil {
				percent = &p
			}
		}

		// Only include bosses that have at least one piece of data
		if days != nil || percent != nil {
			result = append(result, models.SpawnChance{
				World:         world,
				Name:          name,
				Percent:       percent,
				DaysSinceKill: days,
				UpdatedAt:     now,
			})
			
			logMsg := fmt.Sprintf("scraper: %s - %s:", world, name)
			if percent != nil {
				logMsg += fmt.Sprintf(" %d%%", *percent)
			}
			if days != nil {
				logMsg += fmt.Sprintf(" (%d days)", *days)
			}
			log.Print(logMsg)
		}
	}

	log.Printf("scraper: parsed %d bosses for %s", len(result), world)
	return result, nil
}

func cleanHTMLText(text string) string {
	text = htmlTagRE.ReplaceAllString(text, "")
	text = whitespaceRE.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
