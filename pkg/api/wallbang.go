package api

import (
	"math"
	"sort"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

type wallbangShotKey struct {
	round    int
	attacker uint64
	weaponID string
}

type wallbangPositionKey struct {
	round   int
	steamID uint64
}

type wallbangShotIndexEntry struct {
	frame int
	tick  int
	shot  *Shot
}

func markHeuristicWallbangDamages(match *Match) {
	if match == nil || len(match.Damages) == 0 || len(match.Shots) == 0 || len(match.PlayerPositions) == 0 {
		return
	}

	// The parser gives a reliable true wallbang signal for bullet penetrations only when BulletDamage can be
	// correlated to PlayerHurt in the same frame (see damage.isWallbang()). For non-lethal damage this direct signal
	// is frequently unavailable, so we approximate wallbang detection heuristically for outward reporting.
	shotIndex := buildWallbangHeuristicShotIndex(match.Shots)
	positionIndex := buildWallbangHeuristicPositionIndex(match.PlayerPositions)

	for _, damage := range match.Damages {
		isTrueWallbang := damage.isWallbang()
		damage.IsWallbang = isTrueWallbang || isSuspectedWallbangDamage(damage, shotIndex, positionIndex)
	}
}

func isSuspectedWallbangDamage(damage *Damage, shotIndex map[wallbangShotKey][]wallbangShotIndexEntry, positionIndex map[wallbangPositionKey][]*PlayerPosition) bool {
	if damage == nil {
		return false
	}

	if damage.AttackerSteamID64 == 0 || damage.VictimSteamID64 == 0 {
		return false
	}

	if damage.isWallbang() {
		return false
	}

	model, exists := constants.HeuristicWallbangWeaponDamageModels[damage.WeaponName]
	if !exists {
		return false
	}

	if !constants.HeuristicWallbangIncludeHeadshots && isHeadHit(damage.HitGroup) {
		return false
	}

	matchedShot := nearestShotForWallbangDamage(damage, shotIndex)
	if matchedShot == nil {
		return false
	}

	victimPosition := nearestVictimPositionForWallbangDamage(damage, positionIndex)
	distance := 0.0
	if victimPosition != nil {
		distance = euclideanDistance3D(matchedShot.X, matchedShot.Y, matchedShot.Z, victimPosition.X, victimPosition.Y, victimPosition.Z)
	}

	victimHasHelmet := resolveVictimHasHelmet(damage, victimPosition)
	expected := expectedHealthDamageHeuristic(model, damage.HitGroup, distance, damage.VictimArmor, victimHasHelmet)
	expected = capExpectedHealthDamage(expected, damage.VictimHealth)
	observed := float64(damage.HealthDamage)

	delta := expected - observed
	if delta <= 0 || expected <= 0 {
		return false
	}

	if delta < constants.HeuristicWallbangMinDelta {
		return false
	}

	lossRatio := delta / expected
	if lossRatio < constants.HeuristicWallbangMinLossRatio {
		return false
	}

	return true
}

func buildWallbangHeuristicShotIndex(shots []*Shot) map[wallbangShotKey][]wallbangShotIndexEntry {
	index := make(map[wallbangShotKey][]wallbangShotIndexEntry)
	for _, shot := range shots {
		if shot.PlayerSteamID64 == 0 || shot.WeaponID == "" {
			continue
		}

		key := wallbangShotKey{round: shot.RoundNumber, attacker: shot.PlayerSteamID64, weaponID: shot.WeaponID}
		index[key] = append(index[key], wallbangShotIndexEntry{frame: shot.Frame, tick: shot.Tick, shot: shot})
	}

	for key := range index {
		sort.Slice(index[key], func(i int, j int) bool {
			if index[key][i].frame == index[key][j].frame {
				return index[key][i].tick < index[key][j].tick
			}

			return index[key][i].frame < index[key][j].frame
		})
	}

	return index
}

func buildWallbangHeuristicPositionIndex(positions []*PlayerPosition) map[wallbangPositionKey][]*PlayerPosition {
	index := make(map[wallbangPositionKey][]*PlayerPosition)
	for _, position := range positions {
		if position.SteamID64 == 0 {
			continue
		}

		key := wallbangPositionKey{round: position.RoundNumber, steamID: position.SteamID64}
		index[key] = append(index[key], position)
	}

	for key := range index {
		sort.Slice(index[key], func(i int, j int) bool {
			if index[key][i].Frame == index[key][j].Frame {
				return index[key][i].Tick < index[key][j].Tick
			}

			return index[key][i].Frame < index[key][j].Frame
		})
	}

	return index
}

func nearestShotForWallbangDamage(damage *Damage, shotIndex map[wallbangShotKey][]wallbangShotIndexEntry) *Shot {
	key := wallbangShotKey{round: damage.RoundNumber, attacker: damage.AttackerSteamID64, weaponID: damage.WeaponUniqueID}
	entries, exists := shotIndex[key]
	if !exists || len(entries) == 0 {
		return nil
	}

	var best *Shot
	bestFrameDelta := math.MaxInt
	bestTickDelta := math.MaxInt

	for _, entry := range entries {
		if entry.frame > damage.Frame {
			continue
		}
		if entry.frame == damage.Frame && entry.tick > damage.Tick {
			continue
		}

		frameDelta := absInt(entry.frame - damage.Frame)
		if frameDelta > constants.HeuristicWallbangMaxShotFrameDistance {
			continue
		}

		tickDelta := absInt(entry.tick - damage.Tick)
		if frameDelta < bestFrameDelta || (frameDelta == bestFrameDelta && tickDelta < bestTickDelta) {
			best = entry.shot
			bestFrameDelta = frameDelta
			bestTickDelta = tickDelta
		}
	}

	return best
}

func nearestVictimPositionForWallbangDamage(damage *Damage, positionIndex map[wallbangPositionKey][]*PlayerPosition) *PlayerPosition {
	key := wallbangPositionKey{round: damage.RoundNumber, steamID: damage.VictimSteamID64}
	positions, exists := positionIndex[key]
	if !exists || len(positions) == 0 {
		return nil
	}

	var best *PlayerPosition
	bestFrameDelta := math.MaxInt
	bestTickDelta := math.MaxInt

	for _, position := range positions {
		if position.Frame > damage.Frame {
			continue
		}
		if position.Frame == damage.Frame && position.Tick > damage.Tick {
			continue
		}

		frameDelta := absInt(position.Frame - damage.Frame)
		if frameDelta > constants.HeuristicWallbangMaxVictimPositionFrameDelta {
			continue
		}

		tickDelta := absInt(position.Tick - damage.Tick)
		if frameDelta < bestFrameDelta || (frameDelta == bestFrameDelta && tickDelta < bestTickDelta) {
			best = position
			bestFrameDelta = frameDelta
			bestTickDelta = tickDelta
		}
	}

	return best
}

func expectedHealthDamageHeuristic(model constants.HeuristicWallbangWeaponDamageModel, hitGroup events.HitGroup, distance float64, victimArmor int, victimHasHelmet bool) float64 {
	if distance < 0 {
		distance = 0
	}

	expectedBase := model.BaseDamage * math.Pow(model.RangeModifier, distance/500.0)
	postHitgroup := expectedBase * hitgroupMultiplier(hitGroup, model.HeadMultiplier)

	if !isArmoredHit(hitGroup, victimHasHelmet) || victimArmor <= 0 {
		return clampFloat(postHitgroup, 0, postHitgroup)
	}

	healthDamage := postHitgroup * effectiveArmorHealthRatio(model.ArmorRatio)
	armorDamage := (postHitgroup - healthDamage) * 0.5
	if armorDamage > float64(victimArmor) {
		healthDamage = postHitgroup - float64(victimArmor)*effectiveArmorHealthRatio(model.ArmorRatio)
	}

	return clampFloat(healthDamage, 0, postHitgroup)
}

func resolveVictimHasHelmet(damage *Damage, victimPosition *PlayerPosition) bool {
	if !isHeadHit(damage.HitGroup) {
		if victimPosition != nil {
			return victimPosition.HasHelmet
		}

		return false
	}

	if victimPosition != nil && absInt(damage.Frame-victimPosition.Frame) <= constants.HeuristicWallbangHelmetSnapshotFrameWindow {
		return victimPosition.HasHelmet
	}

	if damage.VictimArmor > 0 {
		return true
	}

	return false
}

func effectiveArmorHealthRatio(raw float64) float64 {
	return raw / 2
}

func capExpectedHealthDamage(expected float64, victimHealth int) float64 {
	if expected < 0 {
		return 0
	}

	victimHealthAsFloat := float64(victimHealth)
	if expected > victimHealthAsFloat {
		return victimHealthAsFloat
	}

	return expected
}

func hitgroupMultiplier(hitGroup events.HitGroup, headMultiplier float64) float64 {
	if isHeadHit(hitGroup) {
		return headMultiplier
	}
	if isStomachHit(hitGroup) {
		return 1.25
	}
	if isLegHit(hitGroup) {
		return 0.75
	}

	return 1.0
}

func isHeadHit(hitGroup events.HitGroup) bool {
	return int(hitGroup) == 1
}

func isStomachHit(hitGroup events.HitGroup) bool {
	return int(hitGroup) == 3
}

func isLegHit(hitGroup events.HitGroup) bool {
	v := int(hitGroup)
	return v == 6 || v == 7
}

func isBodyArmorAffectedHit(hitGroup events.HitGroup) bool {
	v := int(hitGroup)
	return v == 2 || v == 3 || v == 4 || v == 5
}

func isArmoredHit(hitGroup events.HitGroup, victimHasHelmet bool) bool {
	if isHeadHit(hitGroup) {
		return victimHasHelmet
	}

	return isBodyArmorAffectedHit(hitGroup)
}

func clampFloat(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}

	return value
}

func euclideanDistance3D(ax float64, ay float64, az float64, bx float64, by float64, bz float64) float64 {
	dx := ax - bx
	dy := ay - by
	dz := az - bz

	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}

	return value
}
