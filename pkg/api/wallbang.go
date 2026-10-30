package api

import (
	"math"
	"sort"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

type wallbangShotKey struct {
	round    int
	attacker uint64
	weaponID string
}

type shotIndexEntry struct {
	frame int
	tick  int
	shot  *Shot
}

func markHeuristicWallbangDamages(match *Match) {
	if match == nil || len(match.Damages) == 0 {
		return
	}

	shotIndex := buildWallbangHeuristicShotIndex(match.Shots)

	for _, damage := range match.Damages {
		if damage == nil {
			continue
		}
		if damage.HasBulletDamageData {
			// An observed zero is authoritative too; never override it with a heuristic.
			damage.IsWallbang = damage.isWallbang()
			damage.WallbangDataSource = "bullet_damage"
			continue
		}
		damage.IsWallbang, damage.WallbangDataSource = classifyHeuristicWallbangDamage(damage, shotIndex)
	}
}

func classifyHeuristicWallbangDamage(damage *Damage, shotIndex map[wallbangShotKey][]shotIndexEntry) (bool, string) {
	if damage == nil {
		return false, "unavailable"
	}

	if damage.AttackerSteamID64 == 0 || damage.VictimSteamID64 == 0 {
		return false, "unavailable"
	}

	model, exists := constants.HeuristicWallbangWeaponDamageModels[damage.WeaponName]
	if !exists {
		return false, "unavailable"
	}

	if !constants.HeuristicWallbangIncludeHeadshots && isHeadHit(damage.HitGroup) {
		return false, "unavailable"
	}

	matchedShot := nearestShotForWallbangDamage(damage, shotIndex)
	if matchedShot == nil {
		return false, "unavailable"
	}

	victimPosition := damage.victimPosition
	if victimPosition == nil {
		return false, "unavailable"
	}
	distance := euclideanDistance3D(matchedShot.X, matchedShot.Y, matchedShot.Z, victimPosition.X, victimPosition.Y, victimPosition.Z)

	expected := expectedHealthDamageHeuristic(model, damage.HitGroup, distance, damage.VictimArmor, victimPosition.HasHelmet)
	expected = capExpectedHealthDamage(expected, damage.VictimHealth)
	observed := float64(damage.HealthDamage)

	delta := expected - observed
	if delta <= 0 || expected <= 0 {
		return false, "heuristic"
	}

	if delta < constants.HeuristicWallbangMinDelta {
		return false, "heuristic"
	}

	lossRatio := delta / expected
	if lossRatio < constants.HeuristicWallbangMinLossRatio {
		return false, "heuristic"
	}

	return true, "heuristic"
}

func buildWallbangHeuristicShotIndex(shots []*Shot) map[wallbangShotKey][]shotIndexEntry {
	index := make(map[wallbangShotKey][]shotIndexEntry)
	for _, shot := range shots {
		if shot.PlayerSteamID64 == 0 || shot.WeaponID == "" {
			continue
		}

		key := wallbangShotKey{round: shot.RoundNumber, attacker: shot.PlayerSteamID64, weaponID: shot.WeaponID}
		index[key] = append(index[key], shotIndexEntry{frame: shot.Frame, tick: shot.Tick, shot: shot})
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

func nearestShotForWallbangDamage(damage *Damage, shotIndex map[wallbangShotKey][]shotIndexEntry) *Shot {
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
		if frameDelta > constants.HeuristicDamageAttributionMaxShotFrameDistance {
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
