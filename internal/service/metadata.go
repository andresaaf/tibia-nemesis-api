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
// - "No Chance" bosses: Only shown if days >= max_days (when inclusion_range is set)
// - "Without prediction" bosses: Always shown (no percentage, not marked as "No Chance")
// - Bosses with percentage: Shown based on inclusion_range rules
// - If days < min_days: exclude boss (not ready to spawn yet)
// - If min_days <= days < max_days: include if has percentage > 0
// - If days >= max_days: always include (even "No Chance")
// - If no inclusion_range defined: include if has percentage OR is "without prediction"
func ApplyInclusionRange(chances []models.SpawnChance, metadata map[string]models.BossMetadata) []models.SpawnChance {
	if len(metadata) == 0 {
		return chances
	}

	var filtered []models.SpawnChance
	for _, chance := range chances {
		meta, exists := metadata[chance.Name]

		// If no metadata or no inclusion_range
		if !exists || meta.InclusionRange == nil {
			// "Without prediction" (no percent, not "No Chance"): always include
			if chance.Percent == nil && !chance.IsNoChance {
				filtered = append(filtered, chance)
				continue
			}
			// Has a percentage > 0: include
			if chance.Percent != nil && *chance.Percent > 0 {
				filtered = append(filtered, chance)
				continue
			}
			// "No Chance" with no inclusion_range: exclude
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

		// Always include if at or above maximum (even "No Chance" bosses)
		if days >= maxDays {
			filtered = append(filtered, chance)
			continue
		}

		// Between min and max:
		// - Include "without prediction" bosses (no percent, not "No Chance")
		// - Include bosses with percentage > 0
		// - Exclude "No Chance" bosses
		if chance.Percent == nil && !chance.IsNoChance {
			// "Without prediction": include
			filtered = append(filtered, chance)
		} else if chance.Percent != nil && *chance.Percent > 0 {
			// Has percentage: include
			filtered = append(filtered, chance)
		}
		// "No Chance" in between range: excluded
	}

	return filtered
}
