package api

import (
	"math"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
)

type fireBulletAimPunch struct {
	frame, round int
	steamID      uint64
	weaponID     string
	angle        r3.Vector
	hasAngle     bool
}

type playerBulletHit struct {
	key     damageMatchFrameKey
	message *msg.CMsgPlayerBulletHit
}

func (analyzer *Analyzer) registerDirectShotSignals() {
	parser := analyzer.parser
	parser.RegisterNetMessageHandler(func(message *msg.CMsgTEFireBullets) {
		if !analyzer.matchStarted() {
			return
		}
		// Player is a pawn handle, not a controller slot or entity ID.
		player := parser.GameState().Participants().FindByPawnHandle(uint64(message.GetPlayer()))
		if player == nil || player.SteamID64 == 0 {
			return
		}
		pawn, weapon := player.PlayerPawnEntity(), player.ActiveWeapon()
		if pawn == nil || weapon == nil || message.WeaponId == nil {
			return
		}
		activeHandle, hasActiveHandle := pawn.PropertyValue("m_pWeaponServices.m_hActiveWeapon")
		if !hasActiveHandle || activeHandle.Any == nil || activeHandle.UInt64() != uint64(message.GetWeaponId()) {
			return
		}
		sample := fireBulletAimPunch{
			frame: parser.CurrentFrame(), round: analyzer.currentRound.Number, steamID: player.SteamID64, weaponID: weapon.UniqueID().String(),
		}
		if angle := message.GetExtra().GetAimPunch(); angle != nil && finiteAngle(angle.GetX(), angle.GetY()) {
			sample.angle, sample.hasAngle = r3.Vector{X: float64(angle.GetX()), Y: float64(angle.GetY())}, true
		}
		// Retain messages without Extra too, so they still make a match ambiguous.
		analyzer.pendingFireBulletAimPunch = append(analyzer.pendingFireBulletAimPunch, sample)
	})
	parser.RegisterNetMessageHandler(func(message *msg.CMsgPlayerBulletHit) {
		if !analyzer.matchStarted() || message.GetAttackerSlot() < 0 || message.GetVictimSlot() < 0 {
			return
		}
		players := parser.GameState().Participants().ByEntityID()
		attacker, victim := players[int(message.GetAttackerSlot())+1], players[int(message.GetVictimSlot())+1]
		if attacker == nil || victim == nil || attacker.SteamID64 == 0 || victim.SteamID64 == 0 || attacker.SteamID64 == victim.SteamID64 {
			return
		}
		analyzer.pendingPlayerBulletHits = append(analyzer.pendingPlayerBulletHits, playerBulletHit{
			key: newDamageMatchFrameKey(analyzer.currentRound.Number, parser.CurrentFrame(), attacker.SteamID64, victim.SteamID64), message: message,
		})
	})
	parser.RegisterEventHandler(func(events.FrameDone) {
		if len(analyzer.pendingFireBulletAimPunch) == 0 && len(analyzer.pendingPlayerBulletHits) == 0 {
			return
		}
		frame := parser.CurrentFrame()
		shots := analyzer.match.Shots
		firstShot := len(shots)
		for firstShot > 0 && shots[firstShot-1].Frame >= frame {
			firstShot--
		}
		damages := analyzer.match.Damages
		firstDamage := len(damages)
		for firstDamage > 0 && damages[firstDamage-1].Frame >= frame {
			firstDamage--
		}
		matchFireBulletAimPunch(shots[firstShot:], analyzer.pendingFireBulletAimPunch)
		matchPlayerBulletHits(damages[firstDamage:], analyzer.pendingPlayerBulletHits)
		analyzer.pendingFireBulletAimPunch = analyzer.pendingFireBulletAimPunch[:0]
		analyzer.pendingPlayerBulletHits = analyzer.pendingPlayerBulletHits[:0]
	})
}

func finiteAngle(x, y float32) bool {
	return !math.IsNaN(float64(x)) && !math.IsNaN(float64(y)) && !math.IsInf(float64(x), 0) && !math.IsInf(float64(y), 0)
}

func matchFireBulletAimPunch(shots []*Shot, samples []fireBulletAimPunch) {
	type key struct {
		frame, round int
		steamID      uint64
		weaponID     string
	}
	byShot := map[key][]*Shot{}
	bySample := map[key][]fireBulletAimPunch{}
	for _, shot := range shots {
		if shot != nil && isBulletWeapon(shot.WeaponType) {
			k := key{shot.Frame, shot.RoundNumber, shot.PlayerSteamID64, shot.WeaponID}
			byShot[k] = append(byShot[k], shot)
		}
	}
	for _, sample := range samples {
		k := key{sample.frame, sample.round, sample.steamID, sample.weaponID}
		bySample[k] = append(bySample[k], sample)
	}
	for k, candidates := range byShot {
		if len(candidates) != 1 || len(bySample[k]) != 1 || !bySample[k][0].hasAngle {
			continue
		}
		shot, angle := candidates[0], bySample[k][0].angle
		shot.AimPunchAngleX, shot.AimPunchAngleY = angle.X, angle.Y
		shot.HasAimPunchData, shot.AimPunchDataSource = true, "fire_bullets"
	}
}

func matchPlayerBulletHits(damages []*Damage, hits []playerBulletHit) {
	byDamage := map[damageMatchFrameKey][]*Damage{}
	byHit := map[damageMatchFrameKey][]*msg.CMsgPlayerBulletHit{}
	for _, damage := range damages {
		if damage == nil {
			continue
		}
		if isBulletWeapon(damage.WeaponType) {
			k := newDamageMatchFrameKeyFromDamage(damage)
			byDamage[k] = append(byDamage[k], damage)
		}
	}
	for _, hit := range hits {
		byHit[hit.key] = append(byHit[hit.key], hit.message)
	}
	for k, candidates := range byDamage {
		if len(candidates) != 1 || len(byHit[k]) != 1 {
			continue
		}
		damage, hit := candidates[0], byHit[k][0]
		if damage.HasBulletDamageData || hit == nil || hit.PenetrationCount == nil || hit.GetPenetrationCount() < 0 {
			continue
		}
		if hit.Damage == nil || int(hit.GetDamage()) != damage.HealthDamage {
			continue
		}
		if hit.HitGroup != nil && int(hit.GetHitGroup()) != int(damage.HitGroup) {
			continue
		}
		if hit.IsKill != nil && hit.GetIsKill() != (damage.VictimNewHealth == 0) {
			continue
		}
		damage.HasPlayerBulletHitData = true
		damage.NumPenetrations = int(hit.GetPenetrationCount())
	}
}

func isBulletWeapon(weaponType constants.WeaponType) bool {
	switch weaponType {
	case constants.WeaponTypePistol, constants.WeaponTypeSMG, constants.WeaponTypeRifle, constants.WeaponTypeSniper, constants.WeaponTypeShotgun, constants.WeaponTypeMachineGun:
		return true
	default:
		return false
	}
}
