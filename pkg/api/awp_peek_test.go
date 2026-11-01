package api

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

func awpPeekFixture() (*Kill, map[roundPlayerKey][]*Shot) {
	kill := &Kill{Tick: 128, Frame: 128, RoundNumber: 1, KillerSteamID64: 1, VictimSteamID64: 3,
		KillerSide: common.TeamCounterTerrorists, VictimSide: common.TeamTerrorists,
		WeaponName: constants.WeaponAWP, HasVictimActiveWeaponData: true, VictimActiveWeaponName: constants.WeaponAWP}
	candidate := &awpPeekCandidate{}
	for tick := 64; tick <= 128; tick++ {
		victim := awpPeekSample{tick: tick, frame: tick, round: 1, eye: r3.Vector{Z: 64},
			aim: r3.Vector{X: 1}, weaponID: "defender", scoped: true}
		y, speed := float64(min(tick, 112)-112)*2, 128.0
		if tick >= 112 {
			speed = 0
		}
		killer := awpPeekSample{tick: tick, frame: tick, round: 1, position: r3.Vector{X: 1000, Y: y},
			eye: r3.Vector{X: 1000, Y: y, Z: 64}, aim: r3.Vector{X: -1}, weaponID: "attacker", speed: speed}
		if tick == 128 {
			candidate.killer, candidate.victim = killer, victim
		} else {
			candidate.killerHistory = append(candidate.killerHistory, killer)
			candidate.victimHistory = append(candidate.victimHistory, victim)
		}
	}
	kill.awpPeekCandidate = candidate
	shots := map[roundPlayerKey][]*Shot{{1, 1}: {{Tick: 128, Frame: 128, RoundNumber: 1, PlayerSteamID64: 1,
		WeaponName: constants.WeaponAWP, WeaponID: "attacker"}}}
	return kill, shots
}

func TestAWPPeekCounterStrafeAndHolderShot(t *testing.T) {
	for _, fired := range []bool{false, true} {
		kill, shots := awpPeekFixture()
		if fired {
			shots[roundPlayerKey{1, 3}] = []*Shot{{Tick: 116, Frame: 116, RoundNumber: 1, PlayerSteamID64: 3, WeaponName: constants.WeaponAWP, WeaponID: "defender"}}
			for i := range kill.awpPeekCandidate.victimHistory {
				if kill.awpPeekCandidate.victimHistory[i].tick >= 116 {
					kill.awpPeekCandidate.victimHistory[i].scoped = false
				}
			}
			kill.awpPeekCandidate.victim.scoped = false
		}
		known, reason, proof := classifyAWPPeek(kill, shots, 64)
		if !known || reason != "qualified" || proof == nil {
			t.Fatalf("fired=%v: %v %s %#v", fired, known, reason, proof)
		}
		if proof.HoldSeconds != 0.5 || proof.PeekDisplacementUnits != 32 || proof.ShotSpeed != 0 ||
			proof.StopToShotSeconds != 17.0/64 || proof.HolderFiredBeforeDeath != fired || proof.FacingAngleDegrees != 0 {
			t.Fatalf("wrong pre-shot evidence: %+v", proof)
		}
	}
}

func TestAWPPeekRejectsOrdinaryAWPKills(t *testing.T) {
	tests := []struct {
		name, reason string
		edit         func(*Kill, map[roundPlayerKey][]*Shot)
	}{
		{"other killer weapon", "not_awp_duel", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.WeaponName = constants.WeaponAK47 }},
		{"other held weapon", "not_awp_duel", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.VictimActiveWeaponName = constants.WeaponScout }},
		{"team", "excluded", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.VictimSide = k.KillerSide }},
		{"suicide", "excluded", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.VictimSteamID64 = k.KillerSteamID64 }},
		{"world", "excluded", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.KillerSteamID64 = 0 }},
		{"killer bot control", "excluded", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.IsKillerControllingBot = true }},
		{"victim bot control", "excluded", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.IsVictimControllingBot = true }},
		{"smoke", "excluded", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.IsThroughSmoke = true }},
		{"wall", "excluded", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.PenetratedObjects = 1 }},
		{"blinded killer", "excluded", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.IsKillerBlinded = true }},
		{"blinded victim", "excluded", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.IsVictimBlinded = true }},
		{"airborne", "excluded", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.IsKillerAirborne = true }},
		{"unscoped", "holder_not_stable", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.awpPeekCandidate.victim.scoped = false }},
		{"holder moving", "holder_not_stable", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.awpPeekCandidate.victimHistory[40].speed = 11 }},
		{"holder drift", "holder_not_stable", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.awpPeekCandidate.victimHistory[40].position.X = 9 }},
		{"holder turns", "holder_not_stable", func(k *Kill, _ map[roundPlayerKey][]*Shot) {
			k.awpPeekCandidate.victim.aim = r3.Vector{X: math.Cos(6 * math.Pi / 180), Y: math.Sin(6 * math.Pi / 180)}
		}},
		{"back turned", "not_facing", func(k *Kill, _ map[roundPlayerKey][]*Shot) {
			for i := range k.awpPeekCandidate.victimHistory {
				k.awpPeekCandidate.victimHistory[i].aim = r3.Vector{X: -1}
			}
			k.awpPeekCandidate.victim.aim = r3.Vector{X: -1}
		}},
		{"vertical mismatch", "not_facing", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.awpPeekCandidate.killer.eye.Z = 1000 }},
		{"still attacker", "peeker_not_moving", func(k *Kill, _ map[roundPlayerKey][]*Shot) {
			for i := range k.awpPeekCandidate.killerHistory {
				k.awpPeekCandidate.killerHistory[i].speed = 0
			}
		}},
		{"small movement", "peeker_not_moving", func(k *Kill, _ map[roundPlayerKey][]*Shot) {
			for i := range k.awpPeekCandidate.killerHistory {
				k.awpPeekCandidate.killerHistory[i].position.Y *= 0.1
			}
		}},
		{"long stopped attacker", "peeker_stopped_too_long", func(k *Kill, _ map[roundPlayerKey][]*Shot) {
			for i := range k.awpPeekCandidate.killerHistory {
				s := &k.awpPeekCandidate.killerHistory[i]
				s.position.Y = float64(min(s.tick, 108)-108) * 2
				if s.tick >= 108 {
					s.speed = 0
				}
			}
		}},
		{"attacker previously flashed", "excluded", func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.awpPeekCandidate.killerHistory[40].blinded = true }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kill, shots := awpPeekFixture()
			tt.edit(kill, shots)
			known, reason, proof := classifyAWPPeek(kill, shots, 64)
			if !known || reason != tt.reason || proof != nil {
				t.Fatalf("%v %s %+v", known, reason, proof)
			}
		})
	}
}

func TestAWPPeekInclusiveBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name            string
		inside, outside float64
		edit            func(*awpPeekCandidate, float64)
	}{
		{"displacement", 16, 15.99, func(c *awpPeekCandidate, value float64) {
			for i := range c.killerHistory {
				c.killerHistory[i].position.Y *= value / 32
			}
		}},
		{"holder radius", 8, 8.01, func(c *awpPeekCandidate, value float64) { c.victim.position.X = value }},
		{"holder speed", 10, 10.01, func(c *awpPeekCandidate, value float64) { c.victim.speed = value }},
		{"holder aim", 5, 5.01, func(c *awpPeekCandidate, value float64) {
			c.victim.aim = r3.Vector{X: math.Cos(value * math.Pi / 180), Y: math.Sin(value * math.Pi / 180)}
		}},
		{"facing", 10, 10.01, func(c *awpPeekCandidate, value float64) {
			aim := r3.Vector{X: math.Cos(value * math.Pi / 180), Y: math.Sin(value * math.Pi / 180)}
			for i := range c.victimHistory {
				c.victimHistory[i].aim = aim
			}
			c.victim.aim = aim
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, inside := range []bool{true, false} {
				kill, shots := awpPeekFixture()
				value := tt.outside
				if inside {
					value = tt.inside
				}
				tt.edit(kill.awpPeekCandidate, value)
				known, reason, proof := classifyAWPPeek(kill, shots, 64)
				if !known || (proof != nil) != inside {
					t.Fatalf("value=%v: %v %s %+v", value, known, reason, proof)
				}
			}
		})
	}
	for _, stopTick := range []int{99, 98} {
		kill, shots := awpPeekFixture()
		for i := range kill.awpPeekCandidate.killerHistory {
			sample := &kill.awpPeekCandidate.killerHistory[i]
			sample.position.Y = float64(min(sample.tick, stopTick)-stopTick) * 2
			if sample.tick >= stopTick {
				sample.speed = 0
			}
		}
		known, reason, proof := classifyAWPPeek(kill, shots, 100)
		if !known || (proof != nil) != (stopTick == 99) {
			t.Fatalf("stopTick=%d: %v %s %+v", stopTick, known, reason, proof)
		}
		if proof != nil && proof.StopToShotSeconds != 0.3 {
			t.Fatal("0.3 second equality lost")
		}
	}
	for _, gap := range []int{8, 9} {
		kill, shots := awpPeekFixture()
		history := kill.awpPeekCandidate.victimHistory
		kill.awpPeekCandidate.victimHistory = append(history[:37:37], history[36+gap:]...)
		known, reason, proof := classifyAWPPeek(kill, shots, 64)
		if (proof != nil) != (gap == 8) || known != (gap == 8) {
			t.Fatalf("gap=%d: %v %s %+v", gap, known, reason, proof)
		}
	}
}

func TestAWPPeekMissingDataStaysUnknown(t *testing.T) {
	for _, edit := range []func(*Kill, map[roundPlayerKey][]*Shot){
		func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.HasVictimActiveWeaponData = false },
		func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.awpPeekCandidate = nil },
		func(_ *Kill, s map[roundPlayerKey][]*Shot) { delete(s, roundPlayerKey{1, 1}) },
		func(_ *Kill, s map[roundPlayerKey][]*Shot) { s[roundPlayerKey{1, 1}][0].Frame++ },
		func(_ *Kill, s map[roundPlayerKey][]*Shot) { s[roundPlayerKey{1, 1}][0].WeaponID = "other awp" },
		func(_ *Kill, s map[roundPlayerKey][]*Shot) {
			s[roundPlayerKey{1, 1}] = append(s[roundPlayerKey{1, 1}], s[roundPlayerKey{1, 1}][0])
		},
		func(k *Kill, _ map[roundPlayerKey][]*Shot) {
			k.awpPeekCandidate.killerHistory = k.awpPeekCandidate.killerHistory[40:]
		},
		func(k *Kill, _ map[roundPlayerKey][]*Shot) {
			k.awpPeekCandidate.victimHistory = k.awpPeekCandidate.victimHistory[:40]
		},
		func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.awpPeekCandidate.killerHistory[40].round++ },
		func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.awpPeekCandidate.killerHistory[40].speed = math.NaN() },
		func(k *Kill, _ map[roundPlayerKey][]*Shot) { k.awpPeekCandidate.killer.eye.Z = math.Inf(1) },
		func(k *Kill, _ map[roundPlayerKey][]*Shot) {
			h := k.awpPeekCandidate.killerHistory
			k.awpPeekCandidate.killerHistory = append(h[:35:35], h[49:]...)
		},
	} {
		kill, shots := awpPeekFixture()
		edit(kill, shots)
		if known, reason, proof := classifyAWPPeek(kill, shots, 64); known || proof != nil {
			t.Fatalf("%v %s %+v", known, reason, proof)
		}
	}
	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		kill, shots := awpPeekFixture()
		if known, _, proof := classifyAWPPeek(kill, shots, rate); known || proof != nil {
			t.Fatal("invalid rate accepted")
		}
	}
}

func TestAWPPeekHolderShotDoesNotExcuseTurningOrMissingTail(t *testing.T) {
	for _, missingTail := range []bool{false, true} {
		kill, shots := awpPeekFixture()
		shots[roundPlayerKey{1, 3}] = []*Shot{{Tick: 112, Frame: 112, WeaponID: "defender"}}
		if missingTail {
			kill.awpPeekCandidate.victimHistory = kill.awpPeekCandidate.victimHistory[:49]
		} else {
			kill.awpPeekCandidate.victim.aim = r3.Vector{Y: 1}
		}
		_, _, proof := classifyAWPPeek(kill, shots, 64)
		if proof != nil {
			t.Fatal("recent shot bypassed hold/tail requirements")
		}
	}
}

func TestAWPPeekHistoryBoundaries(t *testing.T) {
	kill, _ := awpPeekFixture()
	base := kill.awpPeekCandidate.victim
	var history []awpPeekSample
	for i := 1; i <= 500; i++ {
		sample := base
		sample.tick, sample.frame = i, i
		history = appendAWPPeekSample(history, sample, 64)
	}
	if len(history) != 81 {
		t.Fatalf("history not bounded: %d", len(history))
	}
	for _, change := range []func(*awpPeekSample){
		func(s *awpPeekSample) { s.round++ }, func(s *awpPeekSample) { s.weaponID = "another awp" },
		func(s *awpPeekSample) { s.tick += 10 }, func(s *awpPeekSample) { s.frame = 499 },
	} {
		sample := base
		sample.tick, sample.frame = 501, 501
		change(&sample)
		if got := appendAWPPeekSample(history, sample, 64); len(got) != 1 {
			t.Fatal("history crossed a boundary")
		}
	}
	sample := base
	sample.tick, sample.frame = 500, 501
	history = appendAWPPeekSample(history, sample, 64)
	if len(history) != 81 || history[len(history)-1].frame != 501 {
		t.Fatal("duplicate tick not replaced")
	}
	analyzer := &Analyzer{awpPeekHistories: map[uint64][]awpPeekSample{1: history}}
	analyzer.resetBehaviorTracking()
	if len(analyzer.awpPeekHistories) != 0 {
		t.Fatal("round reset retained history")
	}
}

func TestAWPPeekExportsAndSymmetricCounts(t *testing.T) {
	analyzer, _ := utilityKillFixture()
	match := analyzer.match
	positive, shots := awpPeekFixture()
	negative, _ := awpPeekFixture()
	negative.IsThroughSmoke = true
	unknown, _ := awpPeekFixture()
	unknown.awpPeekCandidate = nil
	match.Kills = []*Kill{positive, negative, unknown}
	match.Shots = shots[roundPlayerKey{1, 1}]
	markAWPPeekKills(match)
	markAWPPeekKills(match)
	attacker := &Player{SteamID64: 1, Team: match.TeamA, match: match}
	defender := &Player{SteamID64: 3, Team: match.TeamB, match: match}
	match.PlayersBySteamID = map[uint64]*Player{1: attacker, 3: defender}
	if attacker.AWPPeekKillCount() != 1 || attacker.AWPPeekDeathCount() != 0 || defender.AWPPeekDeathCount() != 1 || defender.AWPPeekKillCount() != 0 {
		t.Fatal("asymmetric or duplicated counts")
	}
	encoded, err := json.Marshal(attacker)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["awpPeekKillCount"] != float64(1) || fields["awpPeekDeathCount"] != float64(0) {
		t.Fatal("missing JSON count")
	}
	encoded, err = json.Marshal(positive)
	if err != nil {
		t.Fatal(err)
	}
	fields = nil
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	proof, ok := fields["awpPeek"].(map[string]any)
	if !ok || proof["peekDisplacementUnits"] != float64(32) || fields["isAwpPeekKill"] != true || fields["hasAwpPeekData"] != true {
		t.Fatal("missing JSON proof")
	}
	dir := t.TempDir()
	if err := exportMatchToCSV(match, dir); err != nil {
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
		var result []map[string]string
		for _, row := range rows[1:] {
			values := map[string]string{}
			for i, name := range rows[0] {
				values[name] = row[i]
			}
			result = append(result, values)
		}
		return result
	}
	rows := read("kills")
	if rows[0]["is awp peek kill"] != "1" || rows[0]["awp peek model"] != "awp_peek_v1" || rows[0]["awp peek displacement units"] != "32.000000" ||
		rows[1]["has awp peek data"] != "1" || rows[1]["awp peek reason"] != "excluded" || rows[1]["awp peek hold seconds"] != "" ||
		rows[2]["has awp peek data"] != "0" || rows[2]["awp peek model"] != "" {
		t.Fatal("wrong CSV proof/availability")
	}
	for _, row := range read("players") {
		if row["steamid"] == "1" && row["awp peek kill count"] != "1" {
			t.Fatal("attacker CSV")
		}
		if row["steamid"] == "3" && row["awp peek death count"] != "1" {
			t.Fatal("defender CSV")
		}
	}
	match.Kills = match.Kills[1:]
	if attacker.AWPPeekKillCount() != 0 || defender.AWPPeekDeathCount() != 0 {
		t.Fatal("count retained a removed round/event")
	}
}
