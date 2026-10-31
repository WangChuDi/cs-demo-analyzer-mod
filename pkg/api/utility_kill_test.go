package api

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

func utilityKillFixture() (*Analyzer, *Kill) {
	ct, tt := common.TeamCounterTerrorists, common.TeamTerrorists
	match := &Match{TickRate: 64, DemoFileName: "utility-kills", TeamA: &Team{CurrentSide: &ct}, TeamB: &Team{CurrentSide: &tt}}
	analyzer := &Analyzer{match: match}
	death := &Kill{Tick: 100, Frame: 100, RoundNumber: 1, KillerSteamID64: 2, VictimSteamID64: 1,
		KillerSide: tt, VictimSide: ct, WeaponName: constants.WeaponAK47}
	analyzer.trackUtilityKill(death, true)
	match.Kills = append(match.Kills, death)
	kill := &Kill{Tick: 164, Frame: 164, RoundNumber: 1, KillerSteamID64: 1, VictimSteamID64: 3,
		KillerSide: ct, VictimSide: tt, WeaponName: constants.WeaponHEGrenade,
		VictimActiveWeaponName: constants.WeaponSmoke, HasVictimActiveWeaponData: true}
	return analyzer, kill
}

func TestUtilityInHandDeathUsesObservedActiveWeapon(t *testing.T) {
	for _, weapon := range []constants.WeaponName{
		constants.WeaponDecoy, constants.WeaponFlashbang, constants.WeaponHEGrenade,
		constants.WeaponIncendiary, constants.WeaponMolotov, constants.WeaponSmoke,
	} {
		t.Run(weapon.String(), func(t *testing.T) {
			analyzer, kill := utilityKillFixture()
			kill.VictimActiveWeaponName = weapon
			analyzer.trackUtilityKill(kill, true)
			analyzer.match.Kills = append(analyzer.match.Kills, kill)
			if !kill.IsVictimHoldingUtility || (&Player{SteamID64: 3, match: analyzer.match}).UtilityInHandDeathCount() != 1 {
				t.Fatal("observed active utility was not counted")
			}
		})
	}
	for _, weapon := range []constants.WeaponName{constants.WeaponAK47, constants.WeaponKnife, constants.WeaponBomb, constants.WeaponDefuseKit, constants.WeaponUnknown, ""} {
		analyzer, kill := utilityKillFixture()
		kill.VictimActiveWeaponName = weapon
		analyzer.trackUtilityKill(kill, true)
		if kill.IsVictimHoldingUtility {
			t.Fatalf("%s counted as throwable utility", weapon)
		}
	}
	analyzer, kill := utilityKillFixture()
	kill.HasVictimActiveWeaponData = false
	analyzer.trackUtilityKill(kill, true)
	if kill.IsVictimHoldingUtility {
		t.Fatal("missing active-weapon data counted")
	}
	for _, change := range []func(*Kill){
		func(k *Kill) { k.KillerSide = k.VictimSide },
		func(k *Kill) { k.KillerSteamID64 = k.VictimSteamID64 },
		func(k *Kill) { k.KillerSteamID64 = 0 },
		func(k *Kill) { k.KillerSide = common.TeamSpectators },
	} {
		analyzer, kill := utilityKillFixture()
		change(kill)
		analyzer.trackUtilityKill(kill, true)
		analyzer.match.Kills = append(analyzer.match.Kills, kill)
		if (&Player{SteamID64: kill.VictimSteamID64, match: analyzer.match}).UtilityInHandDeathCount() != 0 {
			t.Fatal("non-enemy death entered player total")
		}
	}
}

func TestPosthumousUtilityKillsKeepOriginalDeathAndDelay(t *testing.T) {
	for _, weapon := range []constants.WeaponName{
		constants.WeaponDecoy, constants.WeaponFlashbang, constants.WeaponHEGrenade,
		constants.WeaponIncendiary, constants.WeaponMolotov, constants.WeaponSmoke,
	} {
		t.Run(weapon.String(), func(t *testing.T) {
			analyzer, kill := utilityKillFixture()
			kill.WeaponName = weapon
			analyzer.trackUtilityKill(kill, false)
			analyzer.match.Kills = append(analyzer.match.Kills, kill)
			if !kill.IsPosthumousKill || kill.PosthumousDeathTick != 100 || kill.PosthumousDelaySeconds != 1 {
				t.Fatalf("missing death provenance: %+v", kill)
			}
			second := *kill
			second.Tick, second.Frame, second.VictimSteamID64 = 228, 228, 4
			analyzer.trackUtilityKill(&second, false)
			analyzer.match.Kills = append(analyzer.match.Kills, &second)
			if !second.IsPosthumousKill || second.PosthumousDeathTick != 100 || second.PosthumousDelaySeconds != 2 ||
				(&Player{SteamID64: 1, match: analyzer.match}).PosthumousKillCount() != 2 {
				t.Fatal("posthumous multikill lost original death")
			}
		})
	}
}

func TestPosthumousKillsExcludeTradesAndOtherLives(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Analyzer, *Kill)
		alive  bool
	}{
		{"same tick same frame", func(_ *Analyzer, k *Kill) { k.Tick, k.Frame = 100, 100 }, false},
		{"same tick later frame", func(_ *Analyzer, k *Kill) { k.Tick, k.Frame = 100, 101 }, false},
		{"before death", func(_ *Analyzer, k *Kill) { k.Tick = 99 }, false},
		{"earlier frame", func(_ *Analyzer, k *Kill) { k.Frame = 99 }, false},
		{"different round", func(_ *Analyzer, k *Kill) { k.RoundNumber = 2 }, false},
		{"no recorded death", func(a *Analyzer, _ *Kill) { clear(a.posthumousDeaths) }, false},
		{"spawn cleared life", func(a *Analyzer, _ *Kill) { a.clearPosthumousLife(1) }, false},
		{"alive again", func(_ *Analyzer, _ *Kill) {}, true},
		{"controlling bot", func(_ *Analyzer, k *Kill) { k.IsKillerControllingBot = true }, false},
		{"rifle kill", func(_ *Analyzer, k *Kill) { k.WeaponName = constants.WeaponAK47 }, false},
		{"knife kill", func(_ *Analyzer, k *Kill) { k.WeaponName = constants.WeaponKnife }, false},
		{"C4 kill", func(_ *Analyzer, k *Kill) { k.WeaponName = constants.WeaponBomb }, false},
		{"teamkill", func(_ *Analyzer, k *Kill) { k.VictimSide = k.KillerSide }, false},
		{"suicide", func(_ *Analyzer, k *Kill) { k.VictimSteamID64 = k.KillerSteamID64 }, false},
		{"non-player victim", func(_ *Analyzer, k *Kill) { k.VictimSteamID64 = 0 }, false},
		{"unknown clock", func(a *Analyzer, _ *Kill) { a.match.TickRate = 0 }, false},
		{"NaN clock", func(a *Analyzer, _ *Kill) { a.match.TickRate = math.NaN() }, false},
		{"infinite delay", func(a *Analyzer, _ *Kill) { a.match.TickRate = math.SmallestNonzeroFloat64 }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			analyzer, kill := utilityKillFixture()
			tc.change(analyzer, kill)
			analyzer.trackUtilityKill(kill, tc.alive)
			if kill.IsPosthumousKill || kill.PosthumousDeathTick != 0 || kill.PosthumousDelaySeconds != 0 {
				t.Fatalf("invalid posthumous classification: %+v", kill)
			}
		})
	}
	analyzer, kill := utilityKillFixture()
	analyzer.trackUtilityKill(kill, true)
	if _, exists := analyzer.posthumousDeaths[1]; exists {
		t.Fatal("observed alive player retained previous death")
	}
}

func TestUtilityKillExports(t *testing.T) {
	analyzer, kill := utilityKillFixture()
	analyzer.trackUtilityKill(kill, false)
	analyzer.match.Kills = append(analyzer.match.Kills, kill)
	encoded, err := json.Marshal(kill)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]any{
		"hasVictimActiveWeaponData": true, "isVictimHoldingUtility": true,
		"isPosthumousKill": true, "posthumousDeathTick": float64(100), "posthumousDelaySeconds": float64(1),
	} {
		if decoded[key] != want {
			t.Fatalf("JSON %s = %v, want %v", key, decoded[key], want)
		}
	}
	dir := t.TempDir()
	if err := exportMatchToCSV(analyzer.match, dir); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(filepath.Join(dir, "utility-kills_kills.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatal("expected original death and posthumous kill")
	}
	want := map[string][2]string{
		"has victim active weapon data": {"0", "1"},
		"is victim holding utility":     {"0", "1"},
		"is posthumous kill":            {"0", "1"},
		"posthumous death tick":         {"0", "100"},
		"posthumous delay seconds":      {"0.000000", "1.000000"},
	}
	for column, name := range rows[0] {
		if expected, exists := want[name]; exists {
			if rows[1][column] != expected[0] || rows[2][column] != expected[1] {
				t.Fatalf("incorrect CSV %s", name)
			}
			delete(want, name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing CSV columns: %v", want)
	}
}
