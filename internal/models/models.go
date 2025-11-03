package models

import "time"

type SpawnChance struct {
	World     string    `json:"world"`
	Name      string    `json:"name"`
	Percent   *int      `json:"percent"`
	UpdatedAt time.Time `json:"updated_at"`
}
