package api

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

func TestPlayerWallbangDamageTaken(t *testing.T) {
	player := &Player{SteamID64: 123}
	match := &Match{
		Damages: []*Damage{
			{VictimSteamID64: player.SteamID64, HealthDamage: 12},
			{VictimSteamID64: player.SteamID64, HealthDamage: 35, hasBulletDamageData: true, numPenetrations: 1},
			{VictimSteamID64: player.SteamID64, HealthDamage: 17, hasBulletDamageData: true, numPenetrations: 2},
			{VictimSteamID64: 999, HealthDamage: 99, hasBulletDamageData: true, numPenetrations: 3},
		},
	}
	player.match = match

	if got, want := player.TrueWallbangDamageTaken(), 52; got != want {
		t.Fatalf("expected true wallbang damage taken to be %d but got %d", want, got)
	}
}

func TestMarkHeuristicWallbangDamages(t *testing.T) {
	match := &Match{
		Damages: []*Damage{
			{
				RoundNumber:       1,
				Frame:             100,
				Tick:              100,
				AttackerSteamID64: 111,
				VictimSteamID64:   222,
				WeaponName:        "AK-47",
				WeaponUniqueID:    "weapon-1",
				HitGroup:          events.HitGroupChest,
				HealthDamage:      12,
				VictimHealth:      100,
				VictimArmor:       0,
			},
		},
		Shots: []*Shot{
			{
				RoundNumber:     1,
				Frame:           100,
				Tick:            100,
				PlayerSteamID64: 111,
				WeaponID:        "weapon-1",
				X:               0,
				Y:               0,
				Z:               0,
			},
		},
		PlayerPositions: []*PlayerPosition{
			{
				RoundNumber: 1,
				Frame:       100,
				Tick:        100,
				SteamID64:   222,
				X:           0,
				Y:           0,
				Z:           0,
				HasHelmet:   false,
			},
		},
	}

	markHeuristicWallbangDamages(match)

	if !match.Damages[0].IsWallbang {
		t.Fatalf("expected damage to be marked as wallbang")
	}
}

func TestMarkHeuristicWallbangDamages_KeepsTrueWallbang(t *testing.T) {
	match := &Match{
		Damages: []*Damage{
			{
				RoundNumber:         1,
				Frame:               100,
				Tick:                100,
				AttackerSteamID64:   111,
				VictimSteamID64:     222,
				WeaponName:          "AK-47",
				WeaponUniqueID:      "weapon-1",
				HitGroup:            events.HitGroupChest,
				HealthDamage:        12,
				VictimHealth:        100,
				VictimArmor:         0,
				hasBulletDamageData: true,
				numPenetrations:     1,
			},
		},
		Shots: []*Shot{
			{
				RoundNumber:     1,
				Frame:           100,
				Tick:            100,
				PlayerSteamID64: 111,
				WeaponID:        "weapon-1",
				X:               0,
				Y:               0,
				Z:               0,
			},
		},
		PlayerPositions: []*PlayerPosition{
			{
				RoundNumber: 1,
				Frame:       100,
				Tick:        100,
				SteamID64:   222,
				X:           0,
				Y:           0,
				Z:           0,
				HasHelmet:   false,
			},
		},
	}

	markHeuristicWallbangDamages(match)

	if !match.Damages[0].IsWallbang {
		t.Fatalf("expected parser-confirmed true wallbang damage to remain marked as wallbang")
	}
}

func TestMarkHeuristicWallbangDamages_UsesTrueSignalWithoutPositions(t *testing.T) {
	match := &Match{
		Damages: []*Damage{
			{
				RoundNumber:         1,
				Frame:               100,
				Tick:                100,
				AttackerSteamID64:   111,
				VictimSteamID64:     222,
				WeaponName:          "AK-47",
				WeaponUniqueID:      "weapon-1",
				HitGroup:            events.HitGroupChest,
				HealthDamage:        12,
				hasBulletDamageData: true,
				numPenetrations:     1,
			},
		},
	}

	markHeuristicWallbangDamages(match)

	if !match.Damages[0].IsWallbang {
		t.Fatalf("expected parser-confirmed wallbang damage to remain marked even without heuristic position data")
	}
}

func TestPlayerWallbangDamageAggregates(t *testing.T) {
	player := &Player{SteamID64: 123}
	match := &Match{
		Damages: []*Damage{
			{AttackerSteamID64: player.SteamID64, HealthDamage: 10, IsWallbang: true},
			{AttackerSteamID64: player.SteamID64, HealthDamage: 5, IsWallbang: false},
			{VictimSteamID64: player.SteamID64, HealthDamage: 7, IsWallbang: true},
			{VictimSteamID64: player.SteamID64, HealthDamage: 9, IsWallbang: false},
		},
	}
	player.match = match

	if got, want := player.WallbangDamageDealt(), 10; got != want {
		t.Fatalf("expected wallbang damage dealt to be %d but got %d", want, got)
	}

	if got, want := player.WallbangDamageTaken(), 7; got != want {
		t.Fatalf("expected wallbang damage taken to be %d but got %d", want, got)
	}
}

func TestPlayerFirstShotMetrics(t *testing.T) {
	player := &Player{SteamID64: 111}
	match := &Match{
		Shots: []*Shot{
			{
				RoundNumber:     1,
				Frame:           100,
				Tick:            100,
				PlayerSteamID64: player.SteamID64,
				WeaponID:        "weapon-1",
				RecoilIndex:     1,
			},
			{
				RoundNumber:     1,
				Frame:           102,
				Tick:            102,
				PlayerSteamID64: player.SteamID64,
				WeaponID:        "weapon-1",
				RecoilIndex:     2,
			},
			{
				RoundNumber:     1,
				Frame:           200,
				Tick:            200,
				PlayerSteamID64: player.SteamID64,
				WeaponID:        "weapon-2",
				RecoilIndex:     1,
			},
			{
				RoundNumber:     1,
				Frame:           300,
				Tick:            300,
				PlayerSteamID64: player.SteamID64,
				WeaponID:        "weapon-3",
				RecoilIndex:     1,
			},
		},
		Damages: []*Damage{
			{
				RoundNumber:       1,
				Frame:             101,
				Tick:              101,
				AttackerSteamID64: player.SteamID64,
				AttackerSide:      common.TeamTerrorists,
				VictimSteamID64:   222,
				VictimSide:        common.TeamCounterTerrorists,
				WeaponUniqueID:    "weapon-1",
			},
			{
				RoundNumber:       1,
				Frame:             103,
				Tick:              103,
				AttackerSteamID64: player.SteamID64,
				AttackerSide:      common.TeamTerrorists,
				VictimSteamID64:   333,
				VictimSide:        common.TeamCounterTerrorists,
				WeaponUniqueID:    "weapon-1",
			},
			{
				RoundNumber:       1,
				Frame:             201,
				Tick:              201,
				AttackerSteamID64: player.SteamID64,
				AttackerSide:      common.TeamTerrorists,
				VictimSteamID64:   444,
				VictimSide:        common.TeamCounterTerrorists,
				WeaponUniqueID:    "weapon-2",
			},
			{
				RoundNumber:       1,
				Frame:             301,
				Tick:              301,
				AttackerSteamID64: player.SteamID64,
				AttackerSide:      common.TeamTerrorists,
				VictimSteamID64:   555,
				VictimSide:        common.TeamTerrorists,
				WeaponUniqueID:    "weapon-3",
			},
		},
	}
	player.match = match

	if got, want := player.FirstShotCount(), 3; got != want {
		t.Fatalf("expected first shot count to be %d but got %d", want, got)
	}

	if got, want := player.FirstShotHitCount(), 2; got != want {
		t.Fatalf("expected first shot hit count to be %d but got %d", want, got)
	}

	if got, want := player.FirstShotAccuracy(), float32(66.666664); math.Abs(float64(got-want)) > 0.0001 {
		t.Fatalf("expected first shot accuracy to be %v but got %v", want, got)
	}
}

func TestPlayerFirstShotMetrics_DoesNotAttributeSprayDamageToFirstShot(t *testing.T) {
	player := &Player{SteamID64: 111}
	match := &Match{
		Shots: []*Shot{
			{
				RoundNumber:     1,
				Frame:           100,
				Tick:            100,
				PlayerSteamID64: player.SteamID64,
				WeaponID:        "weapon-1",
				RecoilIndex:     1,
			},
			{
				RoundNumber:     1,
				Frame:           102,
				Tick:            102,
				PlayerSteamID64: player.SteamID64,
				WeaponID:        "weapon-1",
				RecoilIndex:     2,
			},
		},
		Damages: []*Damage{
			{
				RoundNumber:       1,
				Frame:             103,
				Tick:              103,
				AttackerSteamID64: player.SteamID64,
				AttackerSide:      common.TeamTerrorists,
				VictimSteamID64:   222,
				VictimSide:        common.TeamCounterTerrorists,
				WeaponUniqueID:    "weapon-1",
			},
		},
	}
	player.match = match

	if got, want := player.FirstShotCount(), 1; got != want {
		t.Fatalf("expected first shot count to be %d but got %d", want, got)
	}

	if got, want := player.FirstShotHitCount(), 0; got != want {
		t.Fatalf("expected first shot hit count to be %d but got %d", want, got)
	}

	if got, want := player.FirstShotAccuracy(), float32(0); math.Abs(float64(got-want)) > 0.0001 {
		t.Fatalf("expected first shot accuracy to be %v but got %v", want, got)
	}
}

func TestPlayerFirstShotMetrics_RespectsAttributionFrameWindow(t *testing.T) {
	player := &Player{SteamID64: 111}
	match := &Match{
		Shots: []*Shot{
			{
				RoundNumber:     1,
				Frame:           100,
				Tick:            100,
				PlayerSteamID64: player.SteamID64,
				WeaponID:        "weapon-1",
				RecoilIndex:     1,
			},
		},
		Damages: []*Damage{
			{
				RoundNumber:       1,
				Frame:             148,
				Tick:              148,
				AttackerSteamID64: player.SteamID64,
				AttackerSide:      common.TeamTerrorists,
				VictimSteamID64:   222,
				VictimSide:        common.TeamCounterTerrorists,
				WeaponUniqueID:    "weapon-1",
			},
			{
				RoundNumber:       1,
				Frame:             149,
				Tick:              149,
				AttackerSteamID64: player.SteamID64,
				AttackerSide:      common.TeamTerrorists,
				VictimSteamID64:   333,
				VictimSide:        common.TeamCounterTerrorists,
				WeaponUniqueID:    "weapon-1",
			},
		},
	}
	player.match = match

	if got, want := player.FirstShotCount(), 1; got != want {
		t.Fatalf("expected first shot count to be %d but got %d", want, got)
	}

	if got, want := player.FirstShotHitCount(), 1; got != want {
		t.Fatalf("expected first shot hit count to be %d but got %d", want, got)
	}

	if got, want := player.FirstShotAccuracy(), float32(100); math.Abs(float64(got-want)) > 0.0001 {
		t.Fatalf("expected first shot accuracy to be %v but got %v", want, got)
	}
}

func TestPlayerMarshalJSONIncludesFirstShotMetrics(t *testing.T) {
	player := &Player{SteamID64: 111, Name: "shooter"}
	match := &Match{
		Shots: []*Shot{{
			RoundNumber:     1,
			Frame:           100,
			Tick:            100,
			PlayerSteamID64: player.SteamID64,
			WeaponID:        "weapon-1",
			RecoilIndex:     1,
		}},
		Damages: []*Damage{{
			RoundNumber:       1,
			Frame:             101,
			Tick:              101,
			AttackerSteamID64: player.SteamID64,
			AttackerSide:      common.TeamTerrorists,
			VictimSteamID64:   222,
			VictimSide:        common.TeamCounterTerrorists,
			WeaponUniqueID:    "weapon-1",
		}},
	}
	player.match = match

	payload, err := player.MarshalJSON()
	if err != nil {
		t.Fatalf("expected player JSON marshaling to succeed: %v", err)
	}

	var actual map[string]any
	if err := json.Unmarshal(payload, &actual); err != nil {
		t.Fatalf("expected marshaled player JSON to decode: %v", err)
	}

	if got, want := actual["firstShotCount"], float64(1); got != want {
		t.Fatalf("expected firstShotCount to be %v but got %v", want, got)
	}
	if got, want := actual["firstShotHitCount"], float64(1); got != want {
		t.Fatalf("expected firstShotHitCount to be %v but got %v", want, got)
	}
	if got, want := actual["firstShotAccuracy"], float64(100); got != want {
		t.Fatalf("expected firstShotAccuracy to be %v but got %v", want, got)
	}
}

func TestBulletDamageCorrelationSameFrame(t *testing.T) {
	analyzer := newBulletDamageTestAnalyzer()
	damage := &Damage{
		RoundNumber:       1,
		Frame:             100,
		Tick:              100,
		AttackerSteamID64: 111,
		VictimSteamID64:   222,
	}
	analyzer.match.Damages = append(analyzer.match.Damages, damage)

	analyzer.registerBulletDamageAtFrame(newBulletDamageEvent(111, 222, 2), 100)

	if !damage.hasBulletDamageData {
		t.Fatalf("expected same-frame damage to be matched with BulletDamage")
	}
	if got, want := damage.numPenetrations, 2; got != want {
		t.Fatalf("expected numPenetrations %d, got %d", want, got)
	}
}

func TestBulletDamageCorrelationBulletDamageBeforePlayerHurtSameFrame(t *testing.T) {
	analyzer := newBulletDamageTestAnalyzer()

	analyzer.registerBulletDamageAtFrame(newBulletDamageEvent(111, 222, 1), 200)

	damage := &Damage{
		RoundNumber:       1,
		Frame:             200,
		Tick:              201,
		AttackerSteamID64: 111,
		VictimSteamID64:   222,
	}
	analyzer.applyPendingBulletDamageToDamage(damage)

	if !damage.hasBulletDamageData {
		t.Fatalf("expected same-frame pending BulletDamage to be applied")
	}
	if got, want := damage.numPenetrations, 1; got != want {
		t.Fatalf("expected numPenetrations %d, got %d", want, got)
	}
}

func TestBulletDamageCorrelationPlayerHurtBeforeBulletDamageSameFrame(t *testing.T) {
	analyzer := newBulletDamageTestAnalyzer()
	damage := &Damage{
		RoundNumber:       1,
		Frame:             300,
		Tick:              300,
		AttackerSteamID64: 111,
		VictimSteamID64:   222,
	}
	analyzer.match.Damages = append(analyzer.match.Damages, damage)

	analyzer.registerBulletDamageAtFrame(newBulletDamageEvent(111, 222, 3), 300)

	if !damage.hasBulletDamageData {
		t.Fatalf("expected same-frame retroactive BulletDamage match")
	}
	if got, want := damage.numPenetrations, 3; got != want {
		t.Fatalf("expected numPenetrations %d, got %d", want, got)
	}
}

func TestBulletDamageCorrelationDifferentFrameDoesNotMatch(t *testing.T) {
	analyzer := newBulletDamageTestAnalyzer()
	damage := &Damage{
		RoundNumber:       1,
		Frame:             400,
		Tick:              400,
		AttackerSteamID64: 111,
		VictimSteamID64:   222,
	}
	analyzer.match.Damages = append(analyzer.match.Damages, damage)

	analyzer.registerBulletDamageAtFrame(newBulletDamageEvent(111, 222, 1), 401)

	if damage.hasBulletDamageData {
		t.Fatalf("expected different-frame BulletDamage not to match")
	}
	if got := len(analyzer.pendingBulletDamageByKey); got != 1 {
		t.Fatalf("expected one pending BulletDamage entry, got %d", got)
	}

	analyzer.applyPendingBulletDamageToDamage(damage)
	if damage.hasBulletDamageData {
		t.Fatalf("expected pending BulletDamage from different frame not to be applied")
	}
}

func newBulletDamageTestAnalyzer() *Analyzer {
	return &Analyzer{
		match: &Match{},
		currentRound: &Round{
			Number: 1,
		},
		pendingBulletDamageByKey: make(map[damageMatchFrameKey][]int),
	}
}

func newBulletDamageEvent(attackerSteamID64 uint64, victimSteamID64 uint64, numPenetrations int) events.BulletDamage {
	return events.BulletDamage{
		Attacker:        &common.Player{SteamID64: attackerSteamID64},
		Victim:          &common.Player{SteamID64: victimSteamID64},
		NumPenetrations: numPenetrations,
	}
}
