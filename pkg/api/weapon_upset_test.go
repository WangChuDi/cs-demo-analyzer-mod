package api

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

func weaponUpsetFixture() *Kill {
	return &Kill{KillerSteamID64: 1, VictimSteamID64: 3, KillerSide: common.TeamCounterTerrorists, VictimSide: common.TeamTerrorists,
		WeaponName: constants.WeaponMP9, WeaponType: constants.WeaponTypeSMG, VictimActiveWeaponName: constants.WeaponAK47,
		HasVictimActiveWeaponData: true, hasWeaponUpsetPositions: true, KillerX: 2048}
}

func TestWeaponUpsetPistolsAgainstAllPrimaryClasses(t *testing.T) {
	for _, pistol := range []constants.WeaponName{constants.WeaponGlock, constants.WeaponUSP, constants.WeaponP2000,
		constants.WeaponP250, constants.WeaponDeagle, constants.WeaponRevolver, constants.WeaponFiveSeven,
		constants.WeaponTec9, constants.WeaponCZ75, constants.WeaponDualBerettas} {
		for _, primary := range []constants.WeaponName{constants.WeaponAK47, constants.WeaponAWP, constants.WeaponMP9,
			constants.WeaponNova, constants.WeaponNegev, constants.WeaponScout, constants.WeaponG3SG1, constants.WeaponScar20} {
			kill := weaponUpsetFixture()
			kill.WeaponName, kill.WeaponType, kill.VictimActiveWeaponName = pistol, constants.WeaponTypePistol, primary
			kill.hasWeaponUpsetPositions = false // The category rule needs neither range nor a versioned damage model.
			known, upset, proof := classifyWeaponUpset(kill, 0)
			if !known || !upset || proof == nil || proof.Rule != "pistol_vs_primary" || proof.ModelSource != "" {
				t.Fatalf("%s vs %s not classified: %v %v %+v", pistol, primary, known, upset, proof)
			}
		}
	}
}

func TestWeaponUpsetRangeFollowsModelAndVersion(t *testing.T) {
	for _, weapon := range []constants.WeaponName{constants.WeaponMac10, constants.WeaponMP5, constants.WeaponMP7,
		constants.WeaponMP9, constants.WeaponP90, constants.WeaponPPBizon, constants.WeaponUMP45} {
		threshold, model, _, known := smgUpsetRange(weapon, 14192)
		if !known || math.Abs(math.Pow(model.RangeModifier, threshold/500)-0.75) > 1e-12 {
			t.Fatalf("wrong 25%% loss boundary for %s: %v", weapon, threshold)
		}
		for _, factor := range []float64{0.999, 1.001} {
			kill := weaponUpsetFixture()
			kill.WeaponName, kill.KillerX = weapon, threshold*factor
			known, upset, proof := classifyWeaponUpset(kill, 14192)
			if !known || upset != (factor > 1) || proof == nil || proof.RangeThresholdUnits != threshold || proof.DamageRetentionThreshold != 0.75 {
				t.Fatalf("range boundary failed for %s at factor %v: %+v", weapon, factor, proof)
			}
		}
	}
	old, _, oldSource, _ := smgUpsetRange(constants.WeaponMP7, 14129)
	latest, _, latestSource, _ := smgUpsetRange(constants.WeaponMP7, 14130)
	if old >= latest || oldSource == latestSource {
		t.Fatal("MP7 versioned falloff was ignored")
	}
	kill := weaponUpsetFixture()
	kill.WeaponName, kill.KillerX = constants.WeaponUMP45, 500
	if known, upset, _ := classifyWeaponUpset(kill, 14192); !known || !upset {
		t.Fatal("exact 25% boundary must count")
	}
	kill.WeaponName, kill.KillerX, kill.Distance = constants.WeaponMP9, 100, 10000
	if known, upset, _ := classifyWeaponUpset(kill, 14192); !known || upset {
		t.Fatal("legacy distance field was used as world units")
	}
}

func TestWeaponUpsetExclusionsAndUnavailableData(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Kill)
		known  bool
	}{
		{"pistol victim", func(k *Kill) { k.VictimActiveWeaponName = constants.WeaponDeagle }, true},
		{"knife victim", func(k *Kill) { k.VictimActiveWeaponName = constants.WeaponKnife }, true},
		{"grenade victim", func(k *Kill) { k.VictimActiveWeaponName = constants.WeaponSmoke }, true},
		{"SMG versus sniper outside first rules", func(k *Kill) { k.VictimActiveWeaponName = constants.WeaponAWP }, true},
		{"rifle attacker", func(k *Kill) { k.WeaponName = constants.WeaponAK47; k.WeaponType = constants.WeaponTypeRifle }, true},
		{"missing held weapon", func(k *Kill) { k.HasVictimActiveWeaponData = false }, false},
		{"unknown held weapon", func(k *Kill) { k.VictimActiveWeaponName = constants.WeaponUnknown }, false},
		{"unknown attack type", func(k *Kill) { k.WeaponType = constants.WeaponTypeUnknown }, false},
		{"missing positions", func(k *Kill) { k.hasWeaponUpsetPositions = false }, false},
		{"NaN position", func(k *Kill) { k.KillerX = math.NaN() }, false},
		{"infinite position", func(k *Kill) { k.VictimZ = math.Inf(1) }, false},
		{"team kill", func(k *Kill) { k.KillerSide = k.VictimSide }, false},
		{"suicide", func(k *Kill) { k.KillerSteamID64 = k.VictimSteamID64 }, false},
		{"no attacker", func(k *Kill) { k.KillerSteamID64 = 0 }, false},
		{"controlling bot", func(k *Kill) { k.IsVictimControllingBot = true }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kill := weaponUpsetFixture()
			tc.change(kill)
			known, upset, _ := classifyWeaponUpset(kill, 14192)
			if known != tc.known || upset {
				t.Fatalf("unexpected classification: %v %v", known, upset)
			}
		})
	}
	for _, protocol := range []int{0, 14004, 14193} {
		if known, upset, _ := classifyWeaponUpset(weaponUpsetFixture(), protocol); known || upset {
			t.Fatal("unsupported model was assumed")
		}
	}
}

func TestWeaponUpsetSelfFlashCacheDemo(t *testing.T) {
	path := os.Getenv("CSDA_UPSET_CACHE_DEMO")
	if path == "" {
		t.Skip("set CSDA_UPSET_CACHE_DEMO to the Aurora-MOUZ Cache demo")
	}
	match, err := AnalyzeDemo(path, AnalyzeDemoOptions{Source: constants.DemoSourceValve})
	if err != nil {
		t.Fatal(err)
	}
	kills, deaths, flashes := 0, 0, 0
	for _, player := range match.PlayersBySteamID {
		kills += player.WeaponUpsetKillCount()
		deaths += player.WeaponUpsetDeathCount()
		flashes += player.SelfFlashStats().SelfFlashCount
	}
	if kills != 3 || deaths != 3 || flashes != 26 {
		t.Fatalf("unexpected real totals: kills=%d deaths=%d self=%d", kills, deaths, flashes)
	}
	foundNear := false
	for _, kill := range match.Kills {
		if kill.Tick == 18879 {
			foundNear = true
			if !kill.HasWeaponUpsetData || kill.IsWeaponUpsetKill || kill.WeaponUpset == nil || kill.WeaponUpset.DamageRetention <= 0.75 {
				t.Fatal("near MP9 kill was counted or lost range evidence")
			}
		}
	}
	if !foundNear {
		t.Fatal("missing real SMG negative")
	}
}

func TestWeaponUpsetAndSelfFlashExports(t *testing.T) {
	analyzer, _ := utilityKillFixture()
	match := analyzer.match
	match.NetworkProtocol = 14192
	long := weaponUpsetFixture()
	near := weaponUpsetFixture()
	near.KillerX = 100
	pistol := weaponUpsetFixture()
	pistol.WeaponName, pistol.WeaponType = constants.WeaponDeagle, constants.WeaponTypePistol
	unknown := weaponUpsetFixture()
	unknown.HasVictimActiveWeaponData = false
	match.Kills = []*Kill{pistol, long, near, unknown}
	markWeaponUpsetKills(match)
	markWeaponUpsetKills(match)
	player := &Player{SteamID64: 1, Team: match.TeamA, match: match}
	victim := &Player{SteamID64: 3, Team: match.TeamB, match: match}
	match.PlayersBySteamID = map[uint64]*Player{1: player, 3: victim}
	match.PlayersFlashed = []*PlayerFlashed{{FlasherSteamID64: 1, FlashedSteamID64: 1, Duration: 2.5}}
	if player.WeaponUpsetKillCount() != 2 || player.WeaponUpsetDeathCount() != 0 || victim.WeaponUpsetDeathCount() != 2 {
		t.Fatal("killer/victim totals are not symmetric or resolution duplicated totals")
	}
	encoded, err := json.Marshal(player)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]float64{"weaponUpsetKillCount": 2, "weaponUpsetDeathCount": 0, "selfFlashCount": 1, "selfFlashDuration": 2.5} {
		if fields[key] != want {
			t.Fatalf("JSON %s = %v", key, fields[key])
		}
	}
	encoded, err = json.Marshal(long)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	proof, ok := fields["weaponUpset"].(map[string]any)
	if !ok || proof["rule"] != "smg_vs_rifle_at_range" || proof["distanceUnits"] != float64(2048) || proof["damageRetentionThreshold"] != 0.75 {
		t.Fatal("JSON range proof missing")
	}
	dir := t.TempDir()
	if err := exportMatchToCSV(match, dir); err != nil {
		t.Fatal(err)
	}
	read := func(suffix string) [][]string {
		f, err := os.Open(filepath.Join(dir, "utility-kills_"+suffix+".csv"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		rows, err := csv.NewReader(f).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		return rows
	}
	rows := read("kills")
	columns := map[string]int{}
	for i, name := range rows[0] {
		columns[name] = i
	}
	for name, want := range map[string][4]string{
		"has weapon upset data":     {"1", "1", "1", "0"},
		"is weapon upset kill":      {"1", "1", "0", "0"},
		"weapon upset rule":         {"pistol_vs_primary", "smg_vs_rifle_at_range", "smg_vs_rifle_at_range", ""},
		"weapon upset model source": {"", "cs2_weapons_14130", "cs2_weapons_14130", ""},
	} {
		col, ok := columns[name]
		if !ok {
			t.Fatalf("missing CSV %s", name)
		}
		for i, value := range want {
			if rows[i+1][col] != value {
				t.Fatalf("CSV %s row %d", name, i)
			}
		}
	}
	for _, name := range []string{"weapon upset distance units", "weapon upset range threshold units", "weapon upset damage retention", "weapon upset damage retention threshold"} {
		col, ok := columns[name]
		if !ok || rows[1][col] != "" || rows[2][col] == "" || rows[4][col] != "" {
			t.Fatalf("CSV range availability: %s", name)
		}
	}
	rows = read("players")
	for _, row := range rows[1:] {
		values := map[string]string{}
		for i, name := range rows[0] {
			values[name] = row[i]
		}
		if values["steamid"] == "1" {
			duration, err := strconv.ParseFloat(values["self flash duration"], 64)
			if values["weapon upset kill count"] != "2" || values["self flash count"] != "1" || err != nil || duration != 2.5 {
				t.Fatal("player CSV metrics missing")
			}
		}
		if values["steamid"] == "3" && values["weapon upset death count"] != "2" {
			t.Fatal("victim CSV total missing")
		}
	}
}
