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
	store   *store.SQLite
	scraper scraper.Scraper
	cfg     config.Config
}

func New(st *store.SQLite, sc scraper.Scraper, cfg config.Config) *Service {
	return &Service{store: st, scraper: sc, cfg: cfg}
}

// StartScheduler performs a daily refresh at configured time.
func (s *Service) StartScheduler() {
	for {
		next := s.nextRun()
		d := time.Until(next)
		if d > 0 {
			time.Sleep(d)
		}
		// In absence of configured worlds list, refresh the worlds we already know
		worlds, _ := s.store.GetWorlds()
		for _, w := range worlds {
			if err := s.RefreshWorld(context.Background(), w); err != nil {
				log.Printf("scheduled refresh %s: %v", w, err)
			}
		}
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

func (s *Service) Spawnables(ctx context.Context, world string) ([]models.SpawnChance, error) {
	return s.store.GetSpawnChances(world)
}

func (s *Service) Worlds(ctx context.Context) ([]string, error) {
	return s.store.GetWorlds()
}

func (s *Service) BossHistory(ctx context.Context, world, name string, limit int) ([]models.SpawnChance, error) {
	return s.store.GetBossHistory(world, name, limit)
}
