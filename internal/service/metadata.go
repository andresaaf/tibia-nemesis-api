package service

import (
	"log"
	"os"

	"tibia-nemesis-api/internal/models"

	"gopkg.in/yaml.v3"
)

// LoadBossMetadata loads boss metadata from YAML file
func LoadBossMetadata(path string) (map[string]models.BossMetadata, error) {
	if path == "" {
		path = "bosses_metadata.yaml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var file models.BossMetadataFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, err
	}

	log.Printf("Loaded metadata for %d bosses (%d with inclusion_range filters)",
		len(file.Bosses), countWithFilters(file.Bosses))

	return file.Bosses, nil
}

func countWithFilters(bosses map[string]models.BossMetadata) int {
	count := 0
	for _, b := range bosses {
		if b.InclusionRange != nil {
			count++
		}
	}
	return count
}

// ApplyInclusionRange filters spawn chances based on boss metadata inclusion_range rules
// Rules:
// - If days < min_days: exclude boss (not ready to spawn yet)
// - If days >= max_days: always include boss (definitely ready)
// - If no inclusion_range defined: always include boss
func ApplyInclusionRange(chances []models.SpawnChance, metadata map[string]models.BossMetadata) []models.SpawnChance {
	if len(metadata) == 0 {
		return chances
	}

	var filtered []models.SpawnChance
	for _, chance := range chances {
		meta, exists := metadata[chance.Name]

		// If no metadata or no inclusion_range, include it
		if !exists || meta.InclusionRange == nil {
			filtered = append(filtered, chance)
			continue
		}

		// If no days data, include it (can't filter without knowing days)
		if chance.DaysSinceKill == nil {
			filtered = append(filtered, chance)
			continue
		}

		days := *chance.DaysSinceKill
		minDays := meta.InclusionRange.MinDays
		maxDays := meta.InclusionRange.MaxDays

		// Exclude if below minimum
		if days < minDays {
			continue
		}

		// Always include if at or above maximum
		if days >= maxDays {
			filtered = append(filtered, chance)
			continue
		}

		// In between min and max: include it
		filtered = append(filtered, chance)
	}

	return filtered
}
