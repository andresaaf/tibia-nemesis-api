package store

import (
	"database/sql"
	"errors"
	"time"

	"tibia-nemesis-api/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

type SQLite struct {
	DB *sql.DB
}

func NewSQLite(path string) (*SQLite, error) {
	if path == "" {
		path = "tibia-nemesis-api.db"
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	s := &SQLite{DB: db}
	if err := s.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) Close() error { return s.DB.Close() }

func (s *SQLite) init() error {
	_, err := s.DB.Exec(`
		CREATE TABLE IF NOT EXISTS spawn_chances (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			world TEXT NOT NULL,
			name TEXT NOT NULL,
			percent INTEGER NULL,
			updated_at TIMESTAMP NOT NULL,
			UNIQUE(world, name)
		);
	`)
	return err
}

func (s *SQLite) UpsertSpawnChances(world string, entries []models.SpawnChance) error {
	if world == "" {
		return errors.New("world required")
	}
	if len(entries) == 0 {
		return nil
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO spawn_chances (world, name, percent, updated_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(world, name) DO UPDATE SET percent=excluded.percent, updated_at=excluded.updated_at`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, e := range entries {
		var p interface{}
		if e.Percent != nil {
			p = *e.Percent
		} else {
			p = nil
		}
		if _, err := stmt.Exec(world, e.Name, p, e.UpdatedAt.UTC()); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLite) GetSpawnChances(world string) ([]models.SpawnChance, error) {
	rows, err := s.DB.Query(`SELECT name, percent, updated_at FROM spawn_chances WHERE world=? ORDER BY name ASC`, world)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SpawnChance
	for rows.Next() {
		var name string
		var percent sql.NullInt64
		var updated time.Time
		if err := rows.Scan(&name, &percent, &updated); err != nil {
			return nil, err
		}
		var p *int
		if percent.Valid {
			v := int(percent.Int64)
			p = &v
		}
		out = append(out, models.SpawnChance{
			World:     world,
			Name:      name,
			Percent:   p,
			UpdatedAt: updated,
		})
	}
	return out, nil
}

func (s *SQLite) GetWorlds() ([]string, error) {
	rows, err := s.DB.Query(`SELECT DISTINCT world FROM spawn_chances ORDER BY world ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var worlds []string
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			return nil, err
		}
		worlds = append(worlds, w)
	}
	return worlds, nil
}

func (s *SQLite) GetBossHistory(world, name string, limit int) ([]models.SpawnChance, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := s.DB.Query(`SELECT percent, updated_at FROM spawn_chances WHERE world=? AND name=? ORDER BY updated_at DESC LIMIT ?`, world, name, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SpawnChance
	for rows.Next() {
		var percent sql.NullInt64
		var updated time.Time
		if err := rows.Scan(&percent, &updated); err != nil {
			return nil, err
		}
		var p *int
		if percent.Valid {
			v := int(percent.Int64)
			p = &v
		}
		out = append(out, models.SpawnChance{
			World:     world,
			Name:      name,
			Percent:   p,
			UpdatedAt: updated,
		})
	}
	return out, nil
}
