package config

import (
	"os"
)

type Config struct {
	Port      string
	DBPath    string
	RefreshAt string // HH:MM (24h)
	TZ        string // IANA TZ, e.g. Europe/Berlin
	Scraper   string // data source: guildstats or tibia-statistic
}

func Load() Config {
	cfg := Config{
		Port:      getenv("PORT", "8080"),
		DBPath:    getenv("DB_PATH", "tibia-nemesis-api.db"),
		RefreshAt: getenv("REFRESH_AT", "9:30"),
		TZ:        getenv("TZ", "CET"),
		Scraper:   getenv("SCRAPER", "guildstats"),
	}
	return cfg
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
