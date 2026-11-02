package api

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/akiver/cs-demo-analyzer/internal/smokevoxel"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

func voxelStatsFixture() (*Analyzer, *SmokeCloud, []float32) {
	a, c := smokeFixture()
	a.match.smokeResidenceModel = smokeVoxelModel
	a.smokeVoxels = &voxelSmokeTracker{entities: map[*st.Entity]*voxelSmokeVolume{}, players: map[uint64]*voxelSmokePlayer{}, protocol: 14191, classAvailable: true}
	grid := make([]float32, smokevoxel.Cells)
	for x := 12; x < 20; x++ {
		grid[smokevoxel.Morton(x, 16, 16)] = 10
	}
	c.voxel = &voxelSmokeVolume{cloud: c, history: []voxelObservation{{tick: 0, density: grid}}}
	return a, c, grid
}

func TestVoxelResidenceUnionAndProvenance(t *testing.T) {
	a, c, _ := voxelStatsFixture()
	second := *c
	second.ID = "second"
	a.smokeState().clouds["second"] = &second
	smokePath(a, r3.Vector{X: -200, Y: 10, Z: 10}, r3.Vector{X: 200, Y: 10, Z: 10}, 0, 128)
	a.markSmokePasses()
	stats := (&Player{SteamID64: 1, match: a.match}).SmokeStats()
	if math.Abs(stats.SmokeStayDuration-.8) > 1e-10 || stats.SmokeVoxelObservedSeconds != 2 || !stats.HasSmokeVoxelData || stats.SmokeModel != smokeVoxelModel || stats.DrySmokePassModel != smokeVoxelModel || stats.SmokeVisualEffectsAvailable {
		t.Fatalf("%+v", stats)
	}
	if stats.SmokeSphereStayDuration <= stats.SmokeStayDuration || stats.DrySmokePassCount != 1 {
		t.Fatal("overlapping voxel pass counted incorrectly", stats)
	}
	if len(a.match.SmokeVoxelVisits) != 1 || len(a.match.SmokeVoxelVisits[0].SmokeIDs) != 2 {
		t.Fatal("overlap double counted or lost provenance", a.match.SmokeVoxelVisits)
	}
	visit := a.match.SmokeVoxelVisits[0]
	if math.Abs(visit.EntryPosition.X+80) > 1e-9 || math.Abs(visit.ExitPosition.X-80) > 1e-9 || math.Abs((visit.ExitTick-visit.EntryTick)/64-visit.Duration) > 1e-10 {
		t.Fatal(visit)
	}
	if !visit.IsFullPass || !visit.IsDryPass || !visit.HasAssistanceData || visit.Reason != "dry_pass" {
		t.Fatal("missing voxel pass evidence", visit)
	}
}

func TestVoxelPassMovementAndModelIsolation(t *testing.T) {
	for _, tt := range []struct {
		name  string
		run   func(*Analyzer, *SmokeCloud, []float32)
		count int
	}{
		{"exact forward boundaries", func(a *Analyzer, _ *SmokeCloud, _ []float32) {
			smokePath(a, r3.Vector{X: -200, Y: 10, Z: 10}, r3.Vector{X: 200, Y: 10, Z: 10}, 0, 80)
		}, 1},
		{"exact reverse boundaries", func(a *Analyzer, _ *SmokeCloud, _ []float32) {
			smokePath(a, r3.Vector{X: 200, Y: 10, Z: 10}, r3.Vector{X: -200, Y: 10, Z: 10}, 0, 80)
		}, 1},
		{"turn back", func(a *Analyzer, _ *SmokeCloud, _ []float32) {
			smokePath(a, r3.Vector{X: -200, Y: 10, Z: 10}, r3.Vector{X: 50, Y: 10, Z: 10}, 0, 50)
			smokePath(a, r3.Vector{X: 50, Y: 10, Z: 10}, r3.Vector{X: -200, Y: 10, Z: 10}, 50, 100)
		}, 0},
		{"initially inside", func(a *Analyzer, _ *SmokeCloud, _ []float32) {
			smokePath(a, r3.Vector{X: -70, Y: 10, Z: 10}, r3.Vector{X: 200, Y: 10, Z: 10}, 0, 80)
		}, 0},
		{"forms around player", func(a *Analyzer, c *SmokeCloud, grid []float32) {
			c.voxel.history = []voxelObservation{{tick: 0, density: make([]float32, smokevoxel.Cells)}, {tick: 32, density: grid}}
			smokePath(a, r3.Vector{X: -200, Y: 10, Z: 10}, r3.Vector{X: 200, Y: 10, Z: 10}, 0, 80)
		}, 0},
		{"expires inside", func(a *Analyzer, c *SmokeCloud, _ []float32) {
			c.EndTick = 40
			smokePath(a, r3.Vector{X: -200, Y: 10, Z: 10}, r3.Vector{X: 200, Y: 10, Z: 10}, 0, 80)
		}, 0},
		{"density disappears", func(a *Analyzer, c *SmokeCloud, _ []float32) {
			c.voxel.history = append(c.voxel.history, voxelObservation{tick: 40, density: make([]float32, smokevoxel.Cells)})
			smokePath(a, r3.Vector{X: -200, Y: 10, Z: 10}, r3.Vector{X: 200, Y: 10, Z: 10}, 0, 80)
		}, 0},
		{"missing grid inside", func(a *Analyzer, c *SmokeCloud, grid []float32) {
			c.voxel.history = append(c.voxel.history, voxelObservation{tick: 40}, voxelObservation{tick: 48, density: grid})
			smokePath(a, r3.Vector{X: -200, Y: 10, Z: 10}, r3.Vector{X: 200, Y: 10, Z: 10}, 0, 80)
		}, 0},
		{"death inside", func(a *Analyzer, _ *SmokeCloud, _ []float32) {
			smokePath(a, r3.Vector{X: -200, Y: 10, Z: 10}, r3.Vector{Y: 10, Z: 10}, 0, 40)
			a.breakSmokePlayer(1, "death")
		}, 0},
		{"shallow touch", func(a *Analyzer, _ *SmokeCloud, grid []float32) {
			for x := 12; x < 20; x++ {
				if x < 15 || x >= 17 {
					grid[smokevoxel.Morton(x, 16, 16)] = 0
				}
			}
			smokePath(a, r3.Vector{X: -200, Y: 10, Z: 10}, r3.Vector{X: 200, Y: 10, Z: 10}, 0, 80)
		}, 0},
		{"sphere pass is not voxel pass", func(a *Analyzer, _ *SmokeCloud, _ []float32) {
			smokePath(a, r3.Vector{X: -200, Y: 30, Z: 10}, r3.Vector{X: 200, Y: 30, Z: 10}, 0, 80)
		}, 0},
		{"voxel pass outside sphere", func(a *Analyzer, c *SmokeCloud, _ []float32) {
			c.voxel.history[0].origin[0] = 500
			smokePath(a, r3.Vector{X: 300, Y: 10, Z: 10}, r3.Vector{X: 700, Y: 10, Z: 10}, 0, 80)
		}, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, c, grid := voxelStatsFixture()
			tt.run(a, c, grid)
			a.markSmokePasses()
			a.markSmokePasses() // Postprocessing is idempotent.
			stats := (&Player{SteamID64: 1, match: a.match}).SmokeStats()
			if stats.DrySmokePassCount != tt.count || stats.DrySmokePassModel != smokeVoxelModel {
				t.Fatalf("stats=%+v visits=%+v", stats, a.match.SmokeVoxelVisits)
			}
		})
	}
}

func TestVoxelPassAssistance(t *testing.T) {
	for _, tt := range []struct {
		name       string
		side       common.Team
		dry, known bool
	}{
		{"friendly", common.TeamCounterTerrorists, false, true},
		{"enemy", common.TeamTerrorists, true, true},
		{"unknown", common.TeamUnassigned, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, _, _ := voxelStatsFixture()
			a.smokeState().supports = []*smokeSupport{{round: 1, SmokeAssistance: SmokeAssistance{Kind: "flash", Tick: 50, EndTick: 50, Side: tt.side, Position: r3.Vector{X: 80, Y: 10, Z: 10}}}}
			smokePath(a, r3.Vector{X: -200, Y: 10, Z: 10}, r3.Vector{X: 200, Y: 10, Z: 10}, 0, 80)
			a.markSmokePasses()
			visit := a.match.SmokeVoxelVisits[0]
			if !visit.IsFullPass || visit.IsDryPass != tt.dry || visit.HasAssistanceData != tt.known {
				t.Fatalf("%+v", visit)
			}
		})
	}
}

func TestVoxelObservationTimesAndUnknownCoverage(t *testing.T) {
	for _, tt := range []struct {
		name               string
		first              int
		second             *voxelObservation
		duration, coverage float64
	}{
		{"no retroactive first grid", 16, nil, .75, .75},
		{"density clears", 0, &voxelObservation{tick: 32, density: make([]float32, smokevoxel.Cells)}, .5, 1},
		{"missing later state", 0, &voxelObservation{tick: 32}, .5, .5},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, c, _ := voxelStatsFixture()
			c.voxel.history[0].tick = tt.first
			if tt.second != nil {
				c.voxel.history = append(c.voxel.history, *tt.second)
			}
			smokePath(a, r3.Vector{Y: 10, Z: 10}, r3.Vector{Y: 10, Z: 10}, 0, 64)
			a.breakSmokePlayer(1, "death")
			stats := (&Player{SteamID64: 1, match: a.match}).SmokeStats()
			if math.Abs(stats.SmokeStayDuration-tt.duration) > 1e-10 || math.Abs(stats.SmokeVoxelObservedSeconds-tt.coverage) > 1e-10 {
				t.Fatalf("%+v", stats)
			}
		})
	}
	a, c, grid := voxelStatsFixture()
	c.voxel.history = append(c.voxel.history, voxelObservation{tick: 32}, voxelObservation{tick: 48, density: grid})
	smokePath(a, r3.Vector{Y: 10, Z: 10}, r3.Vector{Y: 10, Z: 10}, 0, 64)
	a.breakSmokePlayer(1, "death")
	stats := (&Player{SteamID64: 1, match: a.match}).SmokeStats()
	if stats.SmokeStayDuration != .75 || stats.SmokeVoxelObservedSeconds != .75 || len(a.match.SmokeVoxelVisits) != 2 {
		t.Fatal("recovery bridged an unknown interval", stats)
	}
}

func TestVoxelExpiryDeathGapsAndRoundReset(t *testing.T) {
	a, c, _ := voxelStatsFixture()
	c.EndTick = 32
	smokePath(a, r3.Vector{Y: 10, Z: 10}, r3.Vector{Y: 10, Z: 10}, 0, 64)
	a.breakSmokePlayer(1, "death")
	stats := (&Player{SteamID64: 1, match: a.match}).SmokeStats()
	if stats.SmokeStayDuration != .5 || stats.SmokeVoxelObservedSeconds != 1 {
		t.Fatal("expiry not clipped", stats)
	}
	a, c, _ = voxelStatsFixture()
	smokePath(a, r3.Vector{Y: 10, Z: 10}, r3.Vector{Y: 10, Z: 10}, 0, 8)
	a.updateSmokePlayer(smokePlayerSample{tick: 24, frame: 24, round: 1, id: 1, side: a.smokeState().players[1].last.side, eye: r3.Vector{Y: 10, Z: 10}}, 64)
	a.breakSmokePlayer(1, "death")
	stats = (&Player{SteamID64: 1, match: a.match}).SmokeStats()
	if stats.SmokeStayDuration != .125 || stats.SmokeVoxelObservedSeconds != .125 {
		t.Fatal("gap fabricated residence", stats)
	}
	a.discardSmokeRound(1)
	a.match.resetRound(1)
	stats = (&Player{SteamID64: 1, match: a.match}).SmokeStats()
	if stats.HasSmokeVoxelData || stats.SmokeStayDuration != 0 || len(a.match.SmokeVoxelVisits) != 0 || c.voxel != nil {
		t.Fatal("discarded round retained voxel state", stats)
	}
	a, _, _ = voxelStatsFixture()
	a.smokeVoxels.classAvailable = false
	smokePath(a, r3.Vector{Y: 10, Z: 10}, r3.Vector{Y: 10, Z: 10}, 0, 8)
	stats = (&Player{SteamID64: 1, match: a.match}).SmokeStats()
	if stats.HasSmokeVoxelData || stats.SmokeStayDuration != 0 {
		t.Fatal("unavailable class fell back to sphere", stats)
	}
}

func TestVoxelResidenceJSONAndCSV(t *testing.T) {
	a, _, _ := voxelStatsFixture()
	smokePath(a, r3.Vector{X: -200, Y: 10, Z: 10}, r3.Vector{X: 200, Y: 10, Z: 10}, 0, 128)
	a.markSmokePasses()
	player := &Player{SteamID64: 1, match: a.match, Team: a.match.TeamA}
	a.match.PlayersBySteamID = map[uint64]*Player{1: player}
	data, err := json.Marshal(player)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err = json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	if math.Abs(fields["smokeStayDuration"].(float64)-.8) > 1e-10 || fields["smokeModel"] != smokeVoxelModel || fields["hasSmokeVoxelData"] != true || fields["smokeVisualEffectsAvailable"] != false || fields["drySmokePassCount"] != float64(1) || fields["drySmokePassModel"] != smokeVoxelModel {
		t.Fatal(string(data))
	}
	dir := t.TempDir()
	if err = exportMatchToCSV(a.match, dir); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(filepath.Join(dir, "utility-kills_smoke_voxel_visits.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[1][0] != smokeVoxelModel || rows[1][15] != "0.800000" || rows[1][16] != "5.000000" || rows[1][18] != "1" || rows[1][19] != "1" || rows[1][20] != "1" || rows[1][21] != "dry_pass" {
		t.Fatal(rows)
	}
}
