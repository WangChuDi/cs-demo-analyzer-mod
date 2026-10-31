package api

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

func TestFlashEstimatePhasesAndFullWhiteBoundary(t *testing.T) {
	for _, tc := range []struct {
		name                string
		duration, maximum   float64
		tick                int
		phase               string
		overlay, screenshot float64
		full                bool
	}{
		{"initial buildup", 4, 255, 0, "buildup", 0, 0, false},
		{"still building despite three seconds remaining", 4, 255, 6, "buildup", (6.0 / 64) / flashBuildUpSeconds, (6.0 / 64) / flashBuildUpSeconds, false},
		{"first decay tick", 4, 255, 7, "decay", 1, (4 - 7.0/64) / 4, true},
		{"exactly three seconds", 4, 255, 64, "decay", 1, .75, true},
		{"one tick below three seconds", 4, 255, 65, "decay", math.Pow((4-65.0/64)/3, 2), (4 - 65.0/64) / 4, false},
		{"two seconds is four ninths", 4, 255, 128, "decay", 4.0 / 9, .5, false},
		{"reduced cap", 4, 128, 64, "decay", 128.0 / 255, 128.0 / 255 * .75, false},
		{"observed zero cap", 4, 0, 64, "decay", 0, 0, false},
		{"expired", 1, 255, 128, "decay", 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			estimate := estimateFlashAtTick(tc.duration, 0, tc.tick, 64, tc.maximum)
			if estimate == nil || estimate.Model != flashEstimateModel || estimate.Phase != tc.phase || estimate.IsFullWhite != tc.full {
				t.Fatalf("incorrect classification: %+v", estimate)
			}
			if math.Abs(estimate.OverlayAlpha-tc.overlay) > 1e-12 || math.Abs(estimate.ScreenshotAlpha-tc.screenshot) > 1e-12 {
				t.Fatalf("incorrect alpha: %+v", estimate)
			}
		})
	}
}

func TestFlashEstimateRejectsUnavailableInputs(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		duration, rate, alpha float64
		flashTick, tick       int
	}{
		{"no duration", 0, 64, 255, 0, 1},
		{"negative duration", -1, 64, 255, 0, 1},
		{"duration NaN", math.NaN(), 64, 255, 0, 1},
		{"duration infinite", math.Inf(1), 64, 255, 0, 1},
		{"unknown tickrate", 4, 0, 255, 0, 1},
		{"negative tickrate", 4, -64, 255, 0, 1},
		{"tickrate NaN", 4, math.NaN(), 255, 0, 1},
		{"tickrate infinite", 4, math.Inf(1), 255, 0, 1},
		{"elapsed overflows", 4, math.SmallestNonzeroFloat64, 255, 0, 1},
		{"negative cap", 4, 64, -1, 0, 1},
		{"cap above maximum", 4, 64, 256, 0, 1},
		{"cap NaN", 4, 64, math.NaN(), 0, 1},
		{"cap infinite", 4, 64, math.Inf(1), 0, 1},
		{"flash from the future", 4, 64, 255, 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := estimateFlashAtTick(tc.duration, tc.flashTick, tc.tick, tc.rate, tc.alpha); got != nil {
				t.Fatalf("invalid input became an estimate: %+v", got)
			}
		})
	}
	if (&Analyzer{}).estimateKillerFlash(&common.Player{}) != nil ||
		(&Analyzer{isSource2: true}).estimateKillerFlash(nil) != nil ||
		(&Analyzer{isSource2: true}).estimateKillerFlash(&common.Player{}) != nil {
		t.Fatal("unsupported game or missing pawn became an estimate")
	}
}

func TestFullWhiteKillCountOnlyCreditsEnemyKills(t *testing.T) {
	ct, tt := common.TeamCounterTerrorists, common.TeamTerrorists
	full := estimateFlashAtTick(4, 0, 64, 64, 255)
	partial := estimateFlashAtTick(4, 0, 128, 64, 255)
	match := &Match{Kills: []*Kill{
		{KillerSteamID64: 1, VictimSteamID64: 2, KillerSide: ct, VictimSide: tt, KillerFlashEstimate: full},
		{KillerSteamID64: 1, VictimSteamID64: 3, KillerSide: ct, VictimSide: tt, KillerFlashEstimate: partial},
		{KillerSteamID64: 1, VictimSteamID64: 4, KillerSide: ct, VictimSide: tt},
		{KillerSteamID64: 1, VictimSteamID64: 5, KillerSide: ct, VictimSide: ct, KillerFlashEstimate: full},
		{KillerSteamID64: 1, VictimSteamID64: 1, KillerSide: ct, VictimSide: tt, KillerFlashEstimate: full},
		{KillerSteamID64: 2, VictimSteamID64: 1, KillerSide: tt, VictimSide: ct, KillerFlashEstimate: full},
		{KillerSteamID64: 1, VictimSteamID64: 0, KillerSide: ct, VictimSide: tt, KillerFlashEstimate: full},
		{KillerSteamID64: 1, VictimSteamID64: 6, KillerSide: ct, VictimSide: common.TeamSpectators, KillerFlashEstimate: full},
	}}
	if got := (&Player{SteamID64: 1, match: match}).FullWhiteKillCount(); got != 1 {
		t.Fatalf("enemy kill count = %d, want 1", got)
	}
}

func TestFlashEstimateExportsAvailabilityAndCoefficients(t *testing.T) {
	ct, tt := common.TeamCounterTerrorists, common.TeamTerrorists
	match := &Match{
		DemoFileName: "flash-estimate", TeamA: &Team{CurrentSide: &ct}, TeamB: &Team{CurrentSide: &tt},
		Kills: []*Kill{
			{},
			{KillerFlashEstimate: estimateFlashAtTick(4, 0, 64, 64, 0)},
			{KillerFlashEstimate: estimateFlashAtTick(4, 0, 64, 64, 255)},
		},
	}
	for i, kill := range match.Kills {
		encoded, err := json.Marshal(kill)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err = json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatal(err)
		}
		value, exists := decoded["killerFlashEstimate"]
		if exists != (i > 0) {
			t.Fatalf("incorrect JSON availability: %s", encoded)
		}
		if exists && value.(map[string]any)["isFullWhite"] != (i == 2) {
			t.Fatalf("incorrect JSON classification: %s", encoded)
		}
	}
	dir := t.TempDir()
	if err := exportMatchToCSV(match, dir); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "flash-estimate_kills.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("exported %d rows, want header plus three kills", len(rows))
	}
	want := map[string][3]string{
		"has killer flash estimate":      {"0", "1", "1"},
		"is killer full white estimated": {"0", "0", "1"},
		"killer flash duration":          {"0.000000", "4.000000", "4.000000"},
		"killer flash elapsed":           {"0.000000", "1.000000", "1.000000"},
		"killer flash remaining":         {"0.000000", "3.000000", "3.000000"},
		"killer flash max alpha":         {"0.000000", "0.000000", "255.000000"},
		"killer flash overlay alpha":     {"0.000000", "0.000000", "1.000000"},
		"killer flash screenshot alpha":  {"0.000000", "0.000000", "0.750000"},
		"killer flash phase":             {"", "decay", "decay"},
		"killer flash model":             {"", flashEstimateModel, flashEstimateModel},
	}
	for column, name := range rows[0] {
		if values, exists := want[name]; exists {
			for i, value := range values {
				if rows[i+1][column] != value {
					t.Errorf("%s row %d = %q, want %q", name, i, rows[i+1][column], value)
				}
			}
			delete(want, name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing flash CSV columns: %v", want)
	}
}
