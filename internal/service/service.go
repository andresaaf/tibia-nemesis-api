package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"tibia-nemesis-api/internal/config"
	"tibia-nemesis-api/internal/models"
	"tibia-nemesis-api/internal/scraper"
	"tibia-nemesis-api/internal/store"
)

type Service struct {
	store    *store.SQLite
	scraper  scraper.Scraper
	cfg      config.Config
	metadata map[string]models.BossMetadata
}

func New(st *store.SQLite, sc scraper.Scraper, cfg config.Config) *Service {
	svc := &Service{store: st, scraper: sc, cfg: cfg}

	// Load boss metadata for inclusion_range filtering
	if meta, err := LoadBossMetadata("bosses_metadata.yaml"); err != nil {
		log.Printf("Warning: Failed to load boss metadata: %v (filtering disabled)", err)
	} else {
		svc.metadata = meta
	}

	return svc
} // StartScheduler performs a daily refresh at configured time.
func (s *Service) StartScheduler() {
	log.Printf("Scheduler started. Next refresh at: %v", s.nextRun())
	for {
		next := s.nextRun()
		d := time.Until(next)
		log.Printf("Scheduler: sleeping until %v (in %v)", next, d)
		if d > 0 {
			time.Sleep(d)
		}

		log.Printf("Scheduler: starting automatic refresh")
		// In absence of configured worlds list, refresh the worlds we already know
		worlds, err := s.store.GetWorlds()
		if err != nil {
			log.Printf("scheduled refresh: failed to get worlds: %v", err)
			continue
		}

		if len(worlds) == 0 {
			log.Printf("scheduled refresh: no worlds in database to refresh")
			continue
		}

		log.Printf("Scheduler: refreshing %d worlds: %v", len(worlds), worlds)
		for _, w := range worlds {
			log.Printf("Scheduler: refreshing world %s", w)
			if err := s.RefreshWorld(context.Background(), w); err != nil {
				log.Printf("scheduled refresh %s: %v", w, err)
			} else {
				log.Printf("Scheduler: successfully refreshed %s", w)
			}
		}
		log.Printf("Scheduler: refresh complete")
	}
}

func (s *Service) nextRun() time.Time {
	tz, err := time.LoadLocation(s.cfg.TZ)
	if err != nil {
		tz = time.Local
	}
	now := time.Now().In(tz)
	parts := strings.SplitN(s.cfg.RefreshAt, ":", 2)
	hour, min := 9, 0
	if len(parts) == 2 {
		if v, err := time.Parse("15:04", s.cfg.RefreshAt); err == nil {
			hour, min = v.Hour(), v.Minute()
		}
	}
	run := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, tz)
	if !run.After(now) {
		run = run.Add(24 * time.Hour)
	}
	return run
}

func (s *Service) RefreshWorld(ctx context.Context, world string) error {
	if world == "" {
		return errors.New("world required")
	}
	list, err := s.scraper.Fetch(world)
	if err != nil {
		return err
	}
	// Additional logic hook: clamp percent to [0, 100], round, etc.
	for i := range list {
		if list[i].Percent != nil {
			v := *list[i].Percent
			if v < 0 {
				v = 0
			}
			if v > 100 {
				v = 100
			}
			list[i].Percent = &v
		}
		list[i].UpdatedAt = time.Now().UTC()
	}
	return s.store.UpsertSpawnChances(world, list)
}

// Bosses returns all bosses with their spawnable status
func (s *Service) Bosses(ctx context.Context, world string) (*models.BossesResponse, error) {
	allChances, err := s.store.GetSpawnChances(world)
	if err != nil {
		return nil, err
	}

	// Create map of existing bosses from database
	// Use lower-case for all keys for case-insensitive matching
	existingBosses := make(map[string]models.SpawnChance)
	missingFromDB := make(map[string]bool) // Track bosses added from metadata
	nameMap := make(map[string]string)     // lower-case name -> metadata name
	latestUpdate := time.Now().UTC()

	// Build nameMap from metadata
	for metaName := range s.metadata {
		nameMap[strings.ToLower(metaName)] = metaName
	}

	for _, chance := range allChances {
		lc := strings.ToLower(chance.Name)
		// If metadata has this boss, use metadata casing
		if metaName, ok := nameMap[lc]; ok {
			chance.Name = metaName
		}
		existingBosses[lc] = chance
		if chance.UpdatedAt.After(latestUpdate) {
			latestUpdate = chance.UpdatedAt
		}
	}

	// Add metadata bosses that aren't in the database yet
	for metaName := range s.metadata {
		lc := strings.ToLower(metaName)
		if _, exists := existingBosses[lc]; !exists {
			existingBosses[lc] = models.SpawnChance{
				World:         world,
				Name:          metaName,
				Percent:       nil,
				DaysSinceKill: nil,
				IsNoChance:    false,
				UpdatedAt:     time.Now().UTC(),
			}
			missingFromDB[lc] = true // Mark as missing from DB (spawnable by default)
		}
	}

	// Convert map back to slice
	allBosses := make([]models.SpawnChance, 0, len(existingBosses))
	for _, boss := range existingBosses {
		allBosses = append(allBosses, boss)
	}

	// Get spawnables (filtered list)
	spawnableChances := allBosses
	if len(s.metadata) > 0 {
		spawnableChances = ApplyInclusionRange(allBosses, s.metadata)
	}

	// Create a map for quick lookup
	spawnableMap := make(map[string]bool)
	for _, sc := range spawnableChances {
		spawnableMap[strings.ToLower(sc.Name)] = true
	}

	// Build response with all bosses
	bosses := make([]models.BossInfo, 0, len(allBosses))

	for _, chance := range allBosses {
		lc := strings.ToLower(chance.Name)
		// Bosses not in DB (missing from tibia-statistic) are spawnable by default
		spawnable := spawnableMap[lc] || missingFromDB[lc]

		bosses = append(bosses, models.BossInfo{
			Name:          chance.Name,
			Percent:       chance.Percent,
			DaysSinceKill: chance.DaysSinceKill,
			Spawnable:     spawnable,
		})
	}

	return &models.BossesResponse{
		World:     world,
		UpdatedAt: latestUpdate,
		Bosses:    bosses,
	}, nil
}

func (s *Service) Worlds(ctx context.Context) ([]string, error) {
	return s.store.GetWorlds()
}

func (s *Service) BossHistory(ctx context.Context, world, name string, limit int) ([]models.SpawnChance, error) {
	return s.store.GetBossHistory(world, name, limit)
}
