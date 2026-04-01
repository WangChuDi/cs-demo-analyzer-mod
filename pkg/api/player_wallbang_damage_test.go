package api

import (
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
