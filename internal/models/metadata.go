package models

type InclusionRange struct {
	MinDays int `yaml:"min_days" json:"min_days"`
	MaxDays int `yaml:"max_days" json:"max_days"`
}

type BossMetadata struct {
	Name           string          `yaml:"name" json:"name"`
	InclusionRange *InclusionRange `yaml:"inclusion_range,omitempty" json:"inclusion_range,omitempty"`
}

type BossMetadataFile struct {
	Comment string                  `yaml:"_comment" json:"_comment"`
	Bosses  map[string]BossMetadata `yaml:"bosses" json:"bosses"`
}
