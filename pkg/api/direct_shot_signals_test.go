package api

import (
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	"google.golang.org/protobuf/proto"
)

func TestDirectAimPunchUniqueFrameAndPlayer(t *testing.T) {
	shot := func(frame int, steam uint64) *Shot {
		return &Shot{Frame: frame, RoundNumber: 1, PlayerSteamID64: steam, WeaponType: constants.WeaponTypeRifle}
	}
	zero, observed, other := shot(100, 1), shot(101, 1), shot(101, 2)
	ambiguous1, ambiguous2, missing, wrongRound := shot(102, 1), shot(102, 1), shot(103, 1), shot(104, 1)
	matchFireBulletAimPunch([]*Shot{zero, observed, other, ambiguous1, ambiguous2, missing, wrongRound}, []fireBulletAimPunch{
		{hasAngle: true, frame: 100, round: 1, steamID: 1},
		{hasAngle: true, frame: 101, round: 1, steamID: 1, angle: r3.Vector{X: -1.5, Y: .25}},
		{hasAngle: true, frame: 102, round: 1, steamID: 1, angle: r3.Vector{X: 5}},
		{hasAngle: true, frame: 104, round: 2, steamID: 1, angle: r3.Vector{X: 5}},
	})
	if !zero.HasAimPunchData || zero.AimPunchAngleX != 0 || zero.AimPunchDataSource != "fire_bullets" {
		t.Fatal("observed zero was lost")
	}
	if !observed.HasAimPunchData || observed.AimPunchAngleX != -1.5 || observed.AimPunchAngleY != .25 {
		t.Fatal("direct angle was not recovered")
	}
	for _, skipped := range []*Shot{other, ambiguous1, ambiguous2, missing, wrongRound} {
		if skipped.HasAimPunchData {
			t.Fatal("unrelated or ambiguous shot was overwritten")
		}
	}
	matchFireBulletAimPunch([]*Shot{missing}, []fireBulletAimPunch{{hasAngle: true, frame: 103, round: 1, steamID: 1}, {hasAngle: true, frame: 103, round: 1, steamID: 1}})
	if missing.HasAimPunchData {
		t.Fatal("multiple raw messages were treated as a unique shot")
	}
	matchFireBulletAimPunch([]*Shot{missing}, []fireBulletAimPunch{{frame: 103, round: 1, steamID: 1}})
	if missing.HasAimPunchData {
		t.Fatal("missing Extra was treated as observed zero")
	}
	matchFireBulletAimPunch([]*Shot{missing}, []fireBulletAimPunch{{hasAngle: true, frame: 103, round: 1, steamID: 1}, {frame: 103, round: 1, steamID: 1}})
	if missing.HasAimPunchData {
		t.Fatal("a message with missing Extra did not make the association ambiguous")
	}
	matchFireBulletAimPunch([]*Shot{missing}, []fireBulletAimPunch{{hasAngle: true, frame: 103, round: 1, steamID: 1, weaponID: "another weapon"}})
	if missing.HasAimPunchData {
		t.Fatal("different weapons were associated")
	}
}

func TestDirectNonlethalPenetrationAndObservedZero(t *testing.T) {
	for _, penetrations := range []int32{0, 2} {
		t.Run(string(rune('0'+penetrations)), func(t *testing.T) {
			damage := &Damage{Frame: 100, RoundNumber: 1, AttackerSteamID64: 1, VictimSteamID64: 2,
				WeaponType: constants.WeaponTypeRifle, HealthDamage: 19, VictimNewHealth: 81, HitGroup: events.HitGroupChest}
			hit := &msg.CMsgPlayerBulletHit{PenetrationCount: proto.Int32(penetrations), Damage: proto.Int32(19), HitGroup: proto.Int32(int32(events.HitGroupChest)), IsKill: proto.Bool(false)}
			match := &Match{Damages: []*Damage{damage}}
			matchPlayerBulletHits(match.Damages, []playerBulletHit{{key: newDamageMatchFrameKeyFromDamage(damage), message: hit}})
			markHeuristicWallbangDamages(match)
			if !damage.HasPlayerBulletHitData || damage.HasBulletDamageData || damage.HasKillPenetrationData || damage.NumPenetrations != int(penetrations) || damage.IsWallbang != (penetrations > 0) || damage.WallbangDataSource != "player_bullet_hit" {
				t.Fatalf("incorrect nonlethal evidence: %+v", damage)
			}
		})
	}
}

func TestDirectPenetrationRejectsAmbiguityAndConflictingEvidence(t *testing.T) {
	for _, scenario := range []string{"missing penetration", "negative penetration", "wrong damage", "wrong hitgroup", "wrong kill", "different frame", "different victim", "different round", "duplicate hurt", "duplicate hit", "duplicate incomplete hit", "grenade", "bullet damage wins"} {
		t.Run(scenario, func(t *testing.T) {
			d := &Damage{Frame: 100, RoundNumber: 1, AttackerSteamID64: 1, VictimSteamID64: 2, WeaponType: constants.WeaponTypeRifle, HealthDamage: 19, VictimNewHealth: 81, HitGroup: events.HitGroupChest}
			m := &msg.CMsgPlayerBulletHit{PenetrationCount: proto.Int32(2), Damage: proto.Int32(19), HitGroup: proto.Int32(int32(events.HitGroupChest)), IsKill: proto.Bool(false)}
			key := newDamageMatchFrameKeyFromDamage(d)
			damages := []*Damage{d}
			switch scenario {
			case "missing penetration":
				m.PenetrationCount = nil
			case "negative penetration":
				m.PenetrationCount = proto.Int32(-1)
			case "wrong damage":
				m.Damage = proto.Int32(20)
			case "wrong hitgroup":
				m.HitGroup = proto.Int32(int32(events.HitGroupHead))
			case "wrong kill":
				m.IsKill = proto.Bool(true)
			case "different frame":
				key.frame++
			case "different victim":
				key.victimSteamID64++
			case "different round":
				key.roundNumber++
			case "duplicate hurt":
				copy := *d
				damages = append(damages, &copy)
			case "grenade":
				d.WeaponType = constants.WeaponTypeGrenade
			case "bullet damage wins":
				d.setBulletDamageData(0)
			}
			hits := []playerBulletHit{{key: key, message: m}}
			if scenario == "duplicate hit" {
				hits = append(hits, hits[0])
			}
			if scenario == "duplicate incomplete hit" {
				hits = append(hits, playerBulletHit{key: key, message: &msg.CMsgPlayerBulletHit{}})
			}
			matchPlayerBulletHits(damages, hits)
			if d.HasPlayerBulletHitData || d.NumPenetrations != 0 {
				t.Fatalf("conflicting evidence was used: %+v", d)
			}
		})
	}
}
