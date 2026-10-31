package api

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

func TestWeaponUsageCountsFiredModels(t *testing.T) {
	analyzer, _ := utilityKillFixture()
	match := analyzer.match
	add := func(name constants.WeaponName, kind constants.WeaponType) {
		match.Shots = append(match.Shots, &Shot{PlayerSteamID64: 3, WeaponName: name, WeaponType: kind})
	}
	for _, name := range []constants.WeaponName{constants.WeaponAK47, constants.WeaponAK47, constants.WeaponM4A1, constants.WeaponAWP} {
		add(name, constants.WeaponTypeRifle)
	}
	for _, name := range []constants.WeaponName{constants.WeaponDeagle, constants.WeaponGlock, constants.WeaponUSP, constants.WeaponP2000} {
		add(name, constants.WeaponTypePistol)
	}
	add(constants.WeaponKnife, constants.WeaponTypeMelee)
	add(constants.WeaponHEGrenade, constants.WeaponTypeGrenade)
	add(constants.WeaponZeus, constants.WeaponTypeMelee)
	add(constants.WeaponUnknown, constants.WeaponTypeRifle)
	match.Shots = append(match.Shots, nil,
		&Shot{PlayerSteamID64: 4, WeaponName: constants.WeaponNegev, WeaponType: constants.WeaponTypeMachineGun},
		&Shot{PlayerSteamID64: 3, WeaponName: constants.WeaponNegev, WeaponType: constants.WeaponTypeMachineGun, IsPlayerControllingBot: true})
	player := &Player{SteamID64: 3, match: match}
	usage := player.WeaponUsage()
	if usage.UsedWeaponCount != 4 || usage.UsedPrimaryWeaponCount != 3 || usage.UsedSecondaryWeaponCount != 1 ||
		!reflect.DeepEqual(usage.UsedWeapons, []constants.WeaponName{constants.WeaponAK47, constants.WeaponAWP, constants.WeaponDeagle, constants.WeaponM4A1}) {
		t.Fatalf("unexpected model usage: %+v", usage)
	}
	if empty := (&Player{SteamID64: 99, match: match}).WeaponUsage(); empty.UsedWeaponCount != 0 || empty.UsedWeapons == nil {
		t.Fatal("empty usage must be an empty list")
	}
}

func TestForgotArmorUsesFreezeEndLoadout(t *testing.T) {
	weapon := func(kind common.EquipmentType) *common.Equipment { return &common.Equipment{Type: kind} }
	for _, tc := range []struct {
		name                 string
		armor                int
		known                bool
		weapons              []*common.Equipment
		wantData, wantForgot bool
	}{
		{"rifle naked", 0, true, []*common.Equipment{weapon(common.EqAK47), weapon(common.EqGlock)}, true, true},
		{"inherited partial armor", 1, true, []*common.Equipment{weapon(common.EqAK47)}, true, false},
		{"bought armor", 100, true, []*common.Equipment{weapon(common.EqMP9)}, true, false},
		{"AWP exception", 0, true, []*common.Equipment{weapon(common.EqAWP)}, true, false},
		{"Scout exception", 0, true, []*common.Equipment{weapon(common.EqScout)}, true, false},
		{"autosniper exception", 0, true, []*common.Equipment{weapon(common.EqG3SG1)}, true, false},
		{"second autosniper exception", 0, true, []*common.Equipment{weapon(common.EqScar20)}, true, false},
		{"shotgun", 0, true, []*common.Equipment{weapon(common.EqNova)}, true, true},
		{"machine gun", 0, true, []*common.Equipment{weapon(common.EqNegev)}, true, true},
		{"pistol only", 0, true, []*common.Equipment{weapon(common.EqDeagle)}, true, false},
		{"no primary", 0, true, nil, true, false},
		{"armor unavailable", 0, false, []*common.Equipment{weapon(common.EqAK47)}, false, false},
		{"invalid armor", -1, true, []*common.Equipment{weapon(common.EqAK47)}, false, false},
		{"ambiguous primary", 0, true, []*common.Equipment{weapon(common.EqAK47), weapon(common.EqAWP)}, false, false},
		{"incomplete inventory", 0, true, []*common.Equipment{weapon(common.EqAK47), nil}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			economy := &PlayerEconomy{RoundNumber: 1, SteamID64: 3}
			economy.setArmorCheck(100, tc.armor, tc.known, tc.weapons)
			if economy.HasArmorCheckData != tc.wantData || economy.ForgotArmor != tc.wantForgot || economy.ArmorCheckTick != 100 {
				t.Fatalf("wrong loadout classification: %+v", economy)
			}
		})
	}
}

func summonFixture() (*summonTracker, *Kill) {
	tracker := &summonTracker{}
	for tick := 100; tick <= 164; tick++ {
		tracker.update(summonSample{tick: tick, frame: tick, round: 1, valid: true, gun: true,
			weapon: constants.WeaponAK47, weaponID: "old"}, 64)
	}
	kill := &Kill{Tick: 170, Frame: 170, RoundNumber: 1, KillerSteamID64: 2, VictimSteamID64: 3,
		KillerSide: common.TeamCounterTerrorists, VictimSide: common.TeamTerrorists,
		KillerX: 1000, HasVictimActiveWeaponData: true, VictimActiveWeaponName: constants.WeaponSmoke}
	return tracker, kill
}

func summonSwitch(tracker *summonTracker, name constants.WeaponName) {
	sample := tracker.last
	sample.tick, sample.frame, sample.weapon, sample.weaponID = 165, 165, name, "new"
	tracker.update(sample, 64)
}

func TestSummonEnemySwitchAndTurn(t *testing.T) {
	for _, name := range []constants.WeaponName{constants.WeaponKnife, constants.WeaponUSP, constants.WeaponHEGrenade,
		constants.WeaponSmoke, constants.WeaponBomb, constants.WeaponAWP} {
		t.Run(name.String(), func(t *testing.T) {
			tracker, kill := summonFixture()
			summonSwitch(tracker, name)
			kill.VictimActiveWeaponName = name
			evidence := tracker.evidence(kill, 64)
			if evidence == nil || !reflect.DeepEqual(evidence.Reasons, []string{"weapon_switch"}) ||
				evidence.TriggerTick != 165 || evidence.HoldSeconds != 1 || evidence.DelaySeconds != 5.0/64 || evidence.WeaponAfter != name {
				t.Fatalf("missing switch evidence: %+v", evidence)
			}
		})
	}
	tracker, kill := summonFixture()
	sample := tracker.last
	sample.tick, sample.frame, sample.yaw = 165, 165, 90
	tracker.update(sample, 64)
	kill.VictimYaw, kill.VictimActiveWeaponName = 90, constants.WeaponAK47
	if evidence := tracker.evidence(kill, 64); evidence == nil || !reflect.DeepEqual(evidence.Reasons, []string{"turn_away"}) {
		t.Fatalf("missing turn evidence: %+v", evidence)
	}
	if summonAngle(179, -179) != 2 {
		t.Fatal("yaw wrapping produced a false large turn")
	}
}

func TestSummonEnemyRejectsUncertainOrOrdinaryDeaths(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*summonTracker, *Kill)
	}{
		{"same tick", func(tr *summonTracker, k *Kill) { k.Tick = tr.action.tick }},
		{"same frame", func(tr *summonTracker, k *Kill) { k.Frame = tr.action.frame }},
		{"after window", func(_ *summonTracker, k *Kill) { k.Tick, k.Frame = 198, 198 }},
		{"missing samples", func(_ *summonTracker, k *Kill) { k.Tick, k.Frame = 180, 180 }},
		{"different round", func(_ *summonTracker, k *Kill) { k.RoundNumber++ }},
		{"team kill", func(_ *summonTracker, k *Kill) { k.KillerSide = k.VictimSide }},
		{"bot control", func(_ *summonTracker, k *Kill) { k.IsVictimControllingBot = true }},
		{"killer outside held angle", func(_ *summonTracker, k *Kill) { k.KillerX = -1000 }},
		{"returned to hold", func(_ *summonTracker, k *Kill) { k.VictimActiveWeaponName = constants.WeaponAK47 }},
		{"invalid yaw", func(_ *summonTracker, k *Kill) { k.VictimYaw = float32(math.NaN()) }},
		{"moving", func(tr *summonTracker, _ *Kill) { s := tr.last; s.tick++; s.frame++; s.speed = 200; tr.update(s, 64) }},
		{"invalid state", func(tr *summonTracker, _ *Kill) { s := tr.last; s.tick++; s.frame++; s.valid = false; tr.update(s, 64) }},
		{"returned before death", func(tr *summonTracker, _ *Kill) {
			s := tr.last
			s.tick++
			s.frame++
			s.weaponID = "old"
			tr.update(s, 64)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tracker, kill := summonFixture()
			summonSwitch(tracker, constants.WeaponSmoke)
			tc.change(tracker, kill)
			if evidence := tracker.evidence(kill, 64); evidence != nil {
				t.Fatalf("ordinary/uncertain death classified: %+v", evidence)
			}
		})
	}
	tracker, kill := summonFixture()
	if tracker.evidence(kill, 64) != nil {
		t.Fatal("holding without an action was counted")
	}
	// A live sample must confirm the turn, not only the death-event yaw.
	sample := tracker.last
	sample.tick, sample.frame, sample.yaw = 165, 165, 15
	tracker.update(sample, 64)
	kill.VictimYaw, kill.VictimActiveWeaponName = 90, constants.WeaponAK47
	if tracker.evidence(kill, 64) != nil {
		t.Fatal("death-only turn counted")
	}
}

func TestSummonEnemyInterruptionsAndWindowBoundary(t *testing.T) {
	tracker, kill := summonFixture()
	analyzer := &Analyzer{match: &Match{TickRate: 64}, summonTrackers: map[uint64]*summonTracker{3: tracker}}
	analyzer.interruptSummonHold(3)
	if analyzer.summonTrackers[3] != nil {
		t.Fatal("damage before distraction did not clear hold")
	}
	tracker, kill = summonFixture()
	analyzer.summonTrackers[3] = tracker
	summonSwitch(tracker, constants.WeaponSmoke)
	analyzer.interruptSummonHold(3)
	for tick := 166; tick < 197; tick++ {
		sample := tracker.last
		sample.tick, sample.frame = tick, tick
		tracker.update(sample, 64)
	}
	kill.Tick, kill.Frame = 197, 197
	analyzer.applySummonEnemyDeath(kill)
	if !kill.IsSummonEnemyDeath || kill.SummonEnemy.DelaySeconds != 0.5 || analyzer.summonTrackers[3] != nil {
		t.Fatal("boundary classification or death cleanup failed")
	}
}

func TestSummonEnemyNeedsContinuousHold(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*summonTracker)
	}{
		{"short hold", func(tr *summonTracker) { tr.startTick = 140 }},
		{"frame gap", func(tr *summonTracker) { tr.last.tick = 150 }},
		{"round change", func(tr *summonTracker) {
			sample := tr.last
			sample.round, sample.tick, sample.frame = 2, 165, 165
			tr.update(sample, 64)
		}},
		{"no gun", func(tr *summonTracker) { tr.holding = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tracker, kill := summonFixture()
			tc.change(tracker)
			summonSwitch(tracker, constants.WeaponSmoke)
			if tracker.evidence(kill, 64) != nil {
				t.Fatal("incomplete hold counted")
			}
		})
	}
}

func TestBehaviorStatsCacheDemo(t *testing.T) {
	path := os.Getenv("CSDA_BEHAVIOR_CACHE_DEMO")
	if path == "" {
		t.Skip("set CSDA_BEHAVIOR_CACHE_DEMO to the Aurora-MOUZ Cache demo")
	}
	match, err := AnalyzeDemo(path, AnalyzeDemoOptions{Source: constants.DemoSourceValve})
	if err != nil {
		t.Fatal(err)
	}
	if len(match.PlayerEconomies) != 150 {
		t.Fatalf("unexpected economy count: %d", len(match.PlayerEconomies))
	}
	for _, economy := range match.PlayerEconomies {
		if !economy.HasArmorCheckData || economy.ArmorCheckTick <= 0 || economy.ForgotArmor {
			t.Fatalf("lost/incorrect freeze-end loadout: %+v", economy)
		}
	}
	want := map[string]int{"XANTARES": 5, "kyxsan": 5, "Spinx": 5, "woxic": 6, "xertioN": 4, "PR": 6, "torzsi": 4, "Wicadia": 6, "Jimpphat": 6, "xelex": 6}
	for _, player := range match.PlayersBySteamID {
		if usage := player.WeaponUsage(); usage.UsedWeaponCount != want[player.Name] {
			t.Fatalf("wrong real weapon usage for %s: %+v", player.Name, usage)
		}
	}
}

func TestBehaviorStatsJSONAndCSV(t *testing.T) {
	analyzer, _ := utilityKillFixture()
	tracker, kill := summonFixture()
	summonSwitch(tracker, constants.WeaponSmoke)
	analyzer.summonTrackers = map[uint64]*summonTracker{3: tracker}
	analyzer.applySummonEnemyDeath(kill)
	analyzer.match.Kills = []*Kill{kill}
	player := &Player{SteamID64: 3, Name: "test", Team: analyzer.match.TeamA, match: analyzer.match}
	analyzer.match.PlayersBySteamID = map[uint64]*Player{3: player}
	analyzer.match.Shots = []*Shot{{PlayerSteamID64: 3, WeaponName: constants.WeaponDeagle, WeaponType: constants.WeaponTypePistol}}
	economy := &PlayerEconomy{SteamID64: 3, RoundNumber: 1}
	economy.setArmorCheck(80, 0, true, []*common.Equipment{{Type: common.EqAK47}})
	analyzer.match.PlayerEconomies = []*PlayerEconomy{economy}
	encoded, err := json.Marshal(player)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]float64{"usedWeaponCount": 1, "usedPrimaryWeaponCount": 0, "usedSecondaryWeaponCount": 1, "forgotArmorCount": 1, "summonEnemyDeathCount": 1} {
		if fields[key] != want {
			t.Fatalf("JSON %s: %v", key, fields[key])
		}
	}
	dir := t.TempDir()
	if err := exportMatchToCSV(analyzer.match, dir); err != nil {
		t.Fatal(err)
	}
	for suffix, want := range map[string]map[string]string{
		"players":         {"used weapon count": "1", "used secondary weapon count": "1", "used weapons": "Desert Eagle", "forgot armor count": "1", "summon enemy death count": "1"},
		"players_economy": {"has armor check data": "1", "armor at freeze end": "0", "primary weapon at freeze end": "AK-47", "forgot armor": "1"},
		"kills":           {"is summon enemy death": "1", "summon enemy trigger tick": "165", "summon enemy reasons": "weapon_switch", "summon enemy weapon before": "AK-47", "summon enemy weapon after": "Smoke Grenade"},
	} {
		file, err := os.Open(filepath.Join(dir, "utility-kills_"+suffix+".csv"))
		if err != nil {
			t.Fatal(err)
		}
		rows, err := csv.NewReader(file).ReadAll()
		file.Close()
		if err != nil || len(rows) != 2 {
			t.Fatalf("bad CSV %s: %v", suffix, err)
		}
		for i, name := range rows[0] {
			if expected, ok := want[name]; ok {
				if rows[1][i] != expected {
					t.Fatalf("CSV %s %s: %s", suffix, name, rows[1][i])
				}
				delete(want, name)
			}
		}
		if len(want) != 0 {
			t.Fatalf("missing CSV columns: %v", want)
		}
	}
}
