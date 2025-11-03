package models

import "time"

type SpawnChance struct {
	World         string    `json:"world"`
	Name          string    `json:"name"`
	Percent       *int      `json:"percent"`
	DaysSinceKill *int      `json:"days_since_kill"`
	IsNoChance    bool      `json:"is_no_chance"` // Explicitly marked as "No Chance" on website
	UpdatedAt     time.Time `json:"updated_at"`
}
