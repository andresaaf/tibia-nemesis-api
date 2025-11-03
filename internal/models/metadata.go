package models

type InclusionRange struct {
	MinDays int `json:"min_days"`
	MaxDays int `json:"max_days"`
}

type BossMetadata struct {
	Name           string          `json:"name"`
	InclusionRange *InclusionRange `json:"inclusion_range,omitempty"`
}

type BossMetadataFile struct {
	Comment string                  `json:"_comment"`
	Bosses  map[string]BossMetadata `json:"bosses"`
}
