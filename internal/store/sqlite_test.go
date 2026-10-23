package store

import (
	"path/filepath"
	"testing"
	"time"

	"tibia-nemesis-api/internal/models"
)

func TestUpsertRemovesStaleBosses(t *testing.T) {
	st, err := NewSQLite(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Now().UTC()
	entry := func(world, name string) models.SpawnChance {
		return models.SpawnChance{World: world, Name: name, UpdatedAt: now}
	}

	if err := st.UpsertSpawnChances("Kalanta", []models.SpawnChance{entry("Kalanta", "Wild Horse"), entry("Kalanta", "Munster")}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertSpawnChances("Antica", []models.SpawnChance{entry("Antica", "Wild Horse")}); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertSpawnChances("Kalanta", []models.SpawnChance{entry("Kalanta", "Munster"), entry("Kalanta", "Yeti")}); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetSpawnChances("Kalanta")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Munster" || got[1].Name != "Yeti" {
		t.Errorf("Kalanta = %+v, want Munster and Yeti only", got)
	}

	other, err := st.GetSpawnChances("Antica")
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 1 {
		t.Errorf("Antica rows were touched: %+v", other)
	}
}
