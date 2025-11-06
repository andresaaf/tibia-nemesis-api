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

// BossInfo represents a boss in the API response with spawnable status
type BossInfo struct {
	Name          string `json:"name"`
	Percent       *int   `json:"percent"`
	DaysSinceKill *int   `json:"days_since_kill"`
	Spawnable     bool   `json:"spawnable"`
}

// BossesResponse is the wrapper for /api/v1/bosses endpoint
type BossesResponse struct {
	World     string     `json:"world"`
	UpdatedAt time.Time  `json:"updated_at"`
	Bosses    []BossInfo `json:"bosses"`
}
