package api

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

func smokeFixture() (*Analyzer, *SmokeCloud) {
	analyzer, _ := utilityKillFixture()
	cloud := &SmokeCloud{ID: "smoke", RoundNumber: 1, StartTick: -64, ReadyTick: 0, EndTick: 1000, EndReason: "expired"}
	analyzer.smokeState().clouds["smoke"] = cloud
	analyzer.match.SmokeClouds = []*SmokeCloud{cloud}
	return analyzer, cloud
}

func smokePath(analyzer *Analyzer, from, to r3.Vector, first, last int) {
	for tick := first; tick <= last; tick++ {
		position := smokeLerp(from, to, float64(tick-first)/float64(last-first))
		analyzer.updateSmokePlayer(smokePlayerSample{tick: tick, frame: tick, round: 1, id: 1, name: "walker", side: common.TeamCounterTerrorists, eye: position}, 64)
	}
}

func TestSmokeResidenceUnionAndFullPass(t *testing.T) {
	for _, overlap := range []bool{false, true} {
		analyzer, _ := smokeFixture()
		if overlap {
			analyzer.smokeState().clouds["second"] = &SmokeCloud{ID: "second", RoundNumber: 1, ReadyTick: 0, EndTick: 1000, EndReason: "expired"}
		}
		smokePath(analyzer, r3.Vector{X: -200}, r3.Vector{X: 200}, 0, 128)
		analyzer.markSmokePasses()
		analyzer.markSmokePasses()
		stats := (&Player{SteamID64: 1, match: analyzer.match}).SmokeStats()
		if len(analyzer.match.SmokeVisits) != 1 || math.Abs(stats.SmokeStayDuration-1.44) > 1e-10 || stats.DrySmokePassCount != 1 || stats.SmokeObservedSeconds != 2 || !stats.HasSmokeTrackingData {
			t.Fatalf("overlap=%v: %+v visits=%+v", overlap, stats, analyzer.match.SmokeVisits)
		}
		visit := analyzer.match.SmokeVisits[0]
		if !visit.IsFullPass || !visit.HasAssistanceData || visit.Reason != "dry_pass" || math.Abs(visit.EntryPosition.X+144) > 1e-9 || math.Abs(visit.ExitPosition.X-144) > 1e-9 {
			t.Fatalf("bad crossing evidence: %+v", visit)
		}
	}
}

func TestSmokePartialAndEdgeCases(t *testing.T) {
	for _, tt := range []struct {
		name     string
		run      func(*Analyzer, *SmokeCloud)
		duration float64
	}{
		{"turn back", func(a *Analyzer, _ *SmokeCloud) {
			smokePath(a, r3.Vector{X: -200}, r3.Vector{}, 0, 64)
			smokePath(a, r3.Vector{}, r3.Vector{X: -200}, 64, 128)
		}, 1.44},
		{"death inside", func(a *Analyzer, _ *SmokeCloud) {
			smokePath(a, r3.Vector{X: -200}, r3.Vector{}, 0, 64)
			a.breakSmokePlayer(1, "death")
		}, 0.72},
		{"smoke expires", func(a *Analyzer, c *SmokeCloud) {
			c.EndTick = 64
			smokePath(a, r3.Vector{X: -200}, r3.Vector{X: 200}, 0, 128)
		}, 0.72},
		{"smoke forms around player", func(a *Analyzer, c *SmokeCloud) {
			c.ReadyTick = 64
			smokePath(a, r3.Vector{X: -200}, r3.Vector{X: 200}, 0, 128)
		}, 0.72},
		{"initially inside", func(a *Analyzer, _ *SmokeCloud) { smokePath(a, r3.Vector{}, r3.Vector{X: 200}, 0, 64) }, 0.72},
		{"tangent", func(a *Analyzer, _ *SmokeCloud) {
			smokePath(a, r3.Vector{X: -200, Y: 144}, r3.Vector{X: 200, Y: 144}, 0, 128)
		}, 0},
		{"different floor", func(a *Analyzer, _ *SmokeCloud) {
			smokePath(a, r3.Vector{X: -200, Z: 200}, r3.Vector{X: 200, Z: 200}, 0, 128)
		}, 0},
		{"shallow edge", func(a *Analyzer, _ *SmokeCloud) {
			smokePath(a, r3.Vector{X: -200, Y: 120}, r3.Vector{X: 200, Y: 120}, 0, 128)
		}, 2 * math.Sqrt(144*144-120*120) / 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, c := smokeFixture()
			tt.run(a, c)
			a.markSmokePasses()
			stats := (&Player{SteamID64: 1, match: a.match}).SmokeStats()
			if stats.DrySmokePassCount != 0 || math.Abs(stats.SmokeStayDuration-tt.duration) > 1e-9 {
				t.Fatalf("%+v want duration %v", stats, tt.duration)
			}
		})
	}
}

func TestSmokeConservativeAssistance(t *testing.T) {
	for _, tt := range []struct {
		name       string
		support    smokeSupport
		dry, known bool
	}{
		{"friendly flash", smokeSupport{round: 1, SmokeAssistance: SmokeAssistance{Kind: "flash", Tick: 100, EndTick: 100, Side: common.TeamCounterTerrorists, Position: r3.Vector{X: 144}}}, false, true},
		{"friendly HE", smokeSupport{round: 1, SmokeAssistance: SmokeAssistance{Kind: "he", Tick: 100, EndTick: 100, Side: common.TeamCounterTerrorists}}, false, true},
		{"live fire", smokeSupport{round: 1, SmokeAssistance: SmokeAssistance{Kind: "inferno", Tick: -100, EndTick: 80, Side: common.TeamCounterTerrorists}}, false, true},
		{"fire still active", smokeSupport{round: 1, SmokeAssistance: SmokeAssistance{Kind: "inferno", Tick: -100, Side: common.TeamCounterTerrorists}}, false, true},
		{"extinguished fire", smokeSupport{round: 1, SmokeAssistance: SmokeAssistance{Kind: "inferno", Tick: -100, EndTick: -64, Side: common.TeamCounterTerrorists}}, true, true},
		{"enemy flash", smokeSupport{round: 1, SmokeAssistance: SmokeAssistance{Kind: "flash", Tick: 100, EndTick: 100, Side: common.TeamTerrorists}}, true, true},
		{"unknown owner", smokeSupport{round: 1, SmokeAssistance: SmokeAssistance{Kind: "flash", Tick: 100, EndTick: 100}}, false, false},
		{"too late", smokeSupport{round: 1, SmokeAssistance: SmokeAssistance{Kind: "flash", Tick: 143, EndTick: 143, Side: common.TeamCounterTerrorists}}, true, true},
		{"old event", smokeSupport{round: 1, SmokeAssistance: SmokeAssistance{Kind: "flash", Tick: -47, EndTick: -47, Side: common.TeamCounterTerrorists}}, true, true},
		{"another round", smokeSupport{round: 2, SmokeAssistance: SmokeAssistance{Kind: "flash", Tick: 100, EndTick: 100, Side: common.TeamCounterTerrorists}}, true, true},
		{"600 unit boundary", smokeSupport{round: 1, SmokeAssistance: SmokeAssistance{Kind: "he", Tick: 100, EndTick: 100, Side: common.TeamCounterTerrorists, Position: r3.Vector{X: 744}}}, false, true},
		{"beyond boundary", smokeSupport{round: 1, SmokeAssistance: SmokeAssistance{Kind: "he", Tick: 100, EndTick: 100, Side: common.TeamCounterTerrorists, Position: r3.Vector{X: 744.01}}}, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := smokeFixture()
			a.smokeState().supports = append(a.smokeState().supports, &tt.support)
			smokePath(a, r3.Vector{X: -200}, r3.Vector{X: 200}, 0, 128)
			a.markSmokePasses()
			visit := a.match.SmokeVisits[0]
			if visit.IsDryPass != tt.dry || visit.HasAssistanceData != tt.known || !visit.IsFullPass {
				t.Fatalf("%+v", visit)
			}
		})
	}
}

func TestSmokeGapsLivesAndRoundRemoval(t *testing.T) {
	a, _ := smokeFixture()
	smokePath(a, r3.Vector{}, r3.Vector{}, 0, 8)
	a.updateSmokePlayer(smokePlayerSample{tick: 24, frame: 24, round: 1, id: 1, side: common.TeamCounterTerrorists}, 64)
	a.breakSmokePlayer(1, "spawn")
	a.markSmokePasses()
	stats := (&Player{SteamID64: 1, match: a.match}).SmokeStats()
	if stats.DrySmokePassCount != 0 || stats.SmokeStayDuration != .125 || stats.SmokeObservedSeconds != .125 {
		t.Fatalf("gap extrapolated time: %+v", stats)
	}
	a.smokeState().supports = []*smokeSupport{{round: 1}, {round: 2}}
	a.discardSmokeRound(1)
	if len(a.smokeState().players) != 0 || len(a.smokeState().clouds) != 0 || len(a.smokeState().supports) != 1 || a.smokeState().supports[0].round != 2 {
		t.Fatal("round restart retained live history or assistance")
	}
	a.match.resetRound(1)
	stats = (&Player{SteamID64: 1, match: a.match}).SmokeStats()
	if stats.HasSmokeTrackingData || stats.SmokeStayDuration != 0 || len(a.match.SmokeClouds) != 0 {
		t.Fatal("removed round leaked into stats")
	}
	// An unobserved teleport must not fabricate passage or residence.
	a, _ = smokeFixture()
	smokePath(a, r3.Vector{X: -200}, r3.Vector{X: 200}, 0, 1)
	a.markSmokePasses()
	if len(a.match.SmokeVisits) != 0 {
		t.Fatal("teleport became a passage")
	}
}

func TestSmokeJSONAndCSV(t *testing.T) {
	a, _ := smokeFixture()
	smokePath(a, r3.Vector{X: -200}, r3.Vector{X: 200}, 0, 128)
	a.markSmokePasses()
	player := &Player{SteamID64: 1, match: a.match, Team: a.match.TeamA}
	a.match.PlayersBySteamID = map[uint64]*Player{1: player}
	encoded, err := json.Marshal(player)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err = json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["drySmokePassCount"] != float64(1) || fields["smokeStayDuration"] != 1.44 || fields["hasSmokeTrackingData"] != true || fields["smokeModel"] != smokeStatsModel {
		t.Fatalf("%s", encoded)
	}
	dir := t.TempDir()
	if err := exportMatchToCSV(a.match, dir); err != nil {
		t.Fatal(err)
	}
	read := func(suffix string) []map[string]string {
		file, err := os.Open(filepath.Join(dir, "utility-kills_"+suffix+".csv"))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		rows, err := csv.NewReader(file).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		var values []map[string]string
		for _, row := range rows[1:] {
			record := map[string]string{}
			for i, column := range rows[0] {
				record[column] = row[i]
			}
			values = append(values, record)
		}
		return values
	}
	if row := read("players")[0]; row["dry smoke pass count"] != "1" || row["smoke stay duration"] != "1.440000" || row["has smoke tracking data"] != "1" {
		t.Fatal("player CSV")
	}
	if row := read("smoke_visits")[0]; row["is dry pass"] != "1" || row["entry x"] != "-144.000000" || row["duration"] != "1.440000" {
		t.Fatal("visit CSV")
	}
	if row := read("smoke_clouds")[0]; row["id"] != "smoke" || row["model"] != smokeStatsModel {
		t.Fatal("cloud CSV")
	}
}
