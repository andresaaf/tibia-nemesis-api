package scraper

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"tibia-nemesis-api/internal/config"
	"tibia-nemesis-api/internal/models"
)

// Scraper sources, selected with the SCRAPER setting.
const (
	SourceGuildStats     = "guildstats"
	SourceTibiaStatistic = "tibia-statistic"
)

type Scraper interface {
	Fetch(world string) ([]models.SpawnChance, error)
	// Source names the site the data comes from, so clients can credit it
	Source() models.DataSource
}

// New returns the scraper selected by cfg.Scraper. knownBosses are the boss
// names from the metadata file; scrapers use them to decide which bosses
// without a prediction are still worth storing.
func New(cfg config.Config, knownBosses []string) (Scraper, error) {
	switch strings.ToLower(cfg.Scraper) {
	case SourceGuildStats:
		return NewGuildStats(cfg, knownBosses), nil
	case SourceTibiaStatistic:
		return NewTibiaStatistic(cfg), nil
	default:
		return nil, fmt.Errorf("unknown scraper %q (use %q or %q)", cfg.Scraper, SourceGuildStats, SourceTibiaStatistic)
	}
}

// fetchPage GETs url and returns the response body.
func fetchPage(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "TibiaNemesisAPI/1.0")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("scraper: fetch failed for %s: %v", url, err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.Header.Get("cf-mitigated") == "challenge" {
			log.Printf("scraper: HTTP %d for %s (blocked by Cloudflare challenge)", resp.StatusCode, url)
			return "", fmt.Errorf("HTTP %d: blocked by Cloudflare challenge", resp.StatusCode)
		}
		log.Printf("scraper: HTTP %d for %s", resp.StatusCode, url)
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
