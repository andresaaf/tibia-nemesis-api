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
	rowRE          = regexp.MustCompile(`<tr[^>]*id="boss-[^"]+"[^>]*>(.*?)</tr>`)
	nameRE         = regexp.MustCompile(`class="boss-name-link"[^>]*>\s*(.*?)\s*</a>`)
	lastSeenDaysRE = regexp.MustCompile(`(?i)(?:Last\s*Seen|Last\s*kill)[^<:]*:\s*(\d{1,4})\s*day`)
	daysTextRE     = regexp.MustCompile(`(?i)class\s*=\s*"days-text"[^>]*>\s*(\d{1,4})\s*day(?:s)?(?:\s+ago)?`)
	htmlTagRE      = regexp.MustCompile(`<[^>]+>`)
	whitespaceRE   = regexp.MustCompile(`\s+`)
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

	return w.parseDaysSinceLastKill(world, html)
}

func (w *WebScraper) parseDaysSinceLastKill(world, html string) ([]models.SpawnChance, error) {
	out := make(map[string]int)

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

		// Try to extract days
		var days int
		var found bool

		// Try lastSeenDaysRE first
		daysMatch := lastSeenDaysRE.FindStringSubmatch(row)
		if len(daysMatch) >= 2 {
			if d, err := strconv.Atoi(daysMatch[1]); err == nil {
				days = d
				found = true
			}
		}

		// Fallback to daysTextRE
		if !found {
			daysMatch = daysTextRE.FindStringSubmatch(row)
			if len(daysMatch) >= 2 {
				if d, err := strconv.Atoi(daysMatch[1]); err == nil {
					days = d
					found = true
				}
			}
		}

		if found {
			out[name] = days
		}
	}

	// Convert map to SpawnChance slice
	// For now, we're only collecting days; percent will be computed later
	var result []models.SpawnChance
	now := time.Now().UTC()
	for name, days := range out {
		result = append(result, models.SpawnChance{
			World:     world,
			Name:      name,
			Percent:   nil, // Will be computed from days in service layer
			UpdatedAt: now,
		})
		// Store days temporarily - we'll need to add a field or use metadata
		// For now, log it
		log.Printf("scraper: %s - %s: %d days since last kill", world, name, days)
	}

	return result, nil
}

func cleanHTMLText(text string) string {
	text = htmlTagRE.ReplaceAllString(text, "")
	text = whitespaceRE.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
