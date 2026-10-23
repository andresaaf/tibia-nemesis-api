package scraper

import (
	"strings"
	"testing"
	"time"

	"tibia-nemesis-api/internal/config"
	"tibia-nemesis-api/internal/models"
)

// guildStatsRow builds a table row shaped like guildstats.eu's boss list.
func guildStatsRow(name, lastSeen, possibility, possText string) string {
	return `<tr>
		<td class="py-3 text-gray-400 hidden md:table-cell">1 </td>
		<td class="py-3"><a href="bosses/Kalanta/x" class="text-white">
			` + name + `                                </a></td>
		<td class="py-3 text-center"></td>
		<td class="py-3 text-center"><img src="x.gif"></td>
		<td class="py-3 text-center text-gray-300">0</td>
		<td class="py-3 text-center text-gray-300">0</td>
		<td class="py-3 text-center text-gray-300">10</td>
		<td class="py-3 text-center text-gray-300">0</td>
		<td class="py-3 text-center text-gray-300" data-sort-value="0">` + lastSeen + `</td>
		<td class="py-3 text-center text-gray-300" data-sort-value="0">-</td>
		<td class="py-3 text-center" data-sort-value="3"></td>
		<td class="py-3 text-center" data-sort-value="` + possibility + `">` + possText + `</td>
		<td class="py-3 text-center text-gray-300"></td>
	</tr>`
}

func guildStatsPage(rows ...string) string {
	return `<html><body><table class="w-full text-sm sortable-table">
		<thead class="sticky top-16 z-20"><tr>
			<th>#</th><th>Boss name</th><th data-no-sort></th><th>Image</th>
			<th>Yesterday killed</th><th>Yesterday players</th><th>Overall killed</th><th>Overall players</th>
			<th>Last seen</th><th>Introduced</th><th>Type</th>
			<th class="py-3 text-center">Possibility</th>
			<th>Expected in</th>
		</tr></thead>
		<tbody>` + strings.Join(rows, "\n") + `</tbody></table></body></html>`
}

func TestGuildStatsParse(t *testing.T) {
	g := NewGuildStats(config.Config{TZ: "Europe/Berlin"}, []string{
		"Arthom the Hunter", "Draptor", "Fleabringer", "Fleabringer (North)",
	}).(*GuildStatsScraper)
	g.now = func() time.Time { return time.Date(2026, 10, 23, 7, 30, 0, 0, time.UTC) }

	html := guildStatsPage(
		guildStatsRow("Munster", "2026-10-17", "53.22", `<span class="text-green-400 font-bold">53.22%</span>`),
		guildStatsRow("Tiny Chance", "2026-10-20", "0.3", `<span>0.3%</span>`),
		guildStatsRow("Countess Sorrow", "2026-09-23", "-1", `<span class="text-gray-400">Low</span>`),
		guildStatsRow("Ferumbras", "2026-08-10", "-2", `<span class="text-red-400">No</span>`),
		guildStatsRow("Arthom The Hunter", "2026-10-13", "-3", ""),
		guildStatsRow("Abyssador", "2026-10-22", "-3", ""),
		guildStatsRow("draptors", "2026-10-20", "15.11", `<span>15.11%</span>`),
		guildStatsRow("Fleabringer", "-", "13.85", `<span>13.85%</span>`),
	)

	got, err := g.parseSpawnChances("Kalanta", html)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	byName := map[string]models.SpawnChance{}
	for _, c := range got {
		byName[c.Name] = c
	}

	if _, ok := byName["Abyssador"]; ok {
		t.Errorf("Abyssador has no prediction and no metadata, should be skipped")
	}

	tests := []struct {
		name       string
		percent    *int
		days       *int
		isNoChance bool
	}{
		{"Munster", intp(53), intp(6), false},
		{"Tiny Chance", intp(1), intp(3), false},
		{"Countess Sorrow", nil, intp(30), false},
		{"Ferumbras", nil, intp(74), true},
		{"Arthom The Hunter", nil, intp(10), false},
		{"Draptor", intp(15), intp(3), false},
		{"Fleabringer", intp(14), nil, false},
		{"Fleabringer (North)", intp(14), nil, false},
		{"Fleabringer (South)", intp(14), nil, false},
		{"Fleabringer (Surface)", intp(14), nil, false},
	}
	for _, tt := range tests {
		c, ok := byName[tt.name]
		if !ok {
			t.Errorf("%s: missing from result", tt.name)
			continue
		}
		if !eqIntp(c.Percent, tt.percent) {
			t.Errorf("%s: percent = %v, want %v", tt.name, fmtIntp(c.Percent), fmtIntp(tt.percent))
		}
		if !eqIntp(c.DaysSinceKill, tt.days) {
			t.Errorf("%s: days = %v, want %v", tt.name, fmtIntp(c.DaysSinceKill), fmtIntp(tt.days))
		}
		if c.IsNoChance != tt.isNoChance {
			t.Errorf("%s: isNoChance = %v, want %v", tt.name, c.IsNoChance, tt.isNoChance)
		}
		if c.World != "Kalanta" {
			t.Errorf("%s: world = %q", tt.name, c.World)
		}
	}
	if len(got) != len(tests) {
		t.Errorf("got %d entries, want %d", len(got), len(tests))
	}
}

func TestGuildStatsParseMissingTable(t *testing.T) {
	g := NewGuildStats(config.Config{TZ: "CET"}, nil).(*GuildStatsScraper)
	if _, err := g.parseSpawnChances("Kalanta", "<html><body>maintenance</body></html>"); err == nil {
		t.Fatal("expected error for page without boss table")
	}
}

func TestNewSelectsScraper(t *testing.T) {
	if s, err := New(config.Config{Scraper: "guildstats"}, nil); err != nil {
		t.Fatal(err)
	} else if _, ok := s.(*GuildStatsScraper); !ok {
		t.Errorf("guildstats: got %T", s)
	}
	if s, err := New(config.Config{Scraper: "Tibia-Statistic"}, nil); err != nil {
		t.Fatal(err)
	} else if _, ok := s.(*TibiaStatisticScraper); !ok {
		t.Errorf("tibia-statistic: got %T", s)
	}
	if _, err := New(config.Config{Scraper: "nope"}, nil); err == nil {
		t.Error("expected error for unknown scraper")
	}
}

func intp(v int) *int { return &v }

func eqIntp(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func fmtIntp(p *int) any {
	if p == nil {
		return "nil"
	}
	return *p
}
