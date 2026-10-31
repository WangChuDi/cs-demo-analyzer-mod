package api

import (
	"math"
	"sort"
	"strconv"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
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
		damage.HasWallbangHeuristicData = false
		damage.WallbangExpectedDamage, damage.WallbangDamageDeficit = 0, 0
		damage.WallbangHeuristicReason, damage.WallbangModelSource = "", ""
		if damage.HasBulletDamageData {
			// An observed zero is authoritative too; never override it with a heuristic.
			damage.IsWallbang = damage.isWallbang()
			damage.WallbangDataSource = "bullet_damage"
			continue
		}
		if damage.HasKillPenetrationData {
			damage.IsWallbang = damage.isWallbang()
			damage.WallbangDataSource = "kill_event"
			continue
		}
		if damage.HasPlayerBulletHitData {
			damage.IsWallbang = damage.isWallbang()
			damage.WallbangDataSource = "player_bullet_hit"
			continue
		}
		damage.IsWallbang, damage.WallbangDataSource = classifyHeuristicWallbangDamage(damage, shotIndex, match.NetworkProtocol)
	}
}

func classifyHeuristicWallbangDamage(damage *Damage, shotIndex map[wallbangShotKey][]shotIndexEntry, protocol int) (bool, string) {
	if damage == nil {
		return false, "unavailable"
	}
	unknown := func(reason string) (bool, string) {
		damage.WallbangHeuristicReason = reason
		return false, "unavailable"
	}
	if damage.AttackerSteamID64 == 0 || damage.VictimSteamID64 == 0 || damage.AttackerSteamID64 == damage.VictimSteamID64 {
		return unknown("invalid_players")
	}
	// Friendly-fire reduction and self damage must not look like penetration.
	if damage.AttackerSide == damage.VictimSide {
		return unknown("friendly_fire")
	}
	if !damage.HasReportedHealthDamage || damage.ReportedHealthDamage < 0 || damage.VictimHealth <= 0 {
		return unknown("missing_reported_damage")
	}
	if int(damage.HitGroup) < 1 || int(damage.HitGroup) > 7 {
		return unknown("unsupported_hitgroup")
	}
	model, source, exists := constants.HeuristicWallbangModel(damage.WeaponName, protocol)
	if !exists {
		return unknown("unsupported_weapon_or_version")
	}
	if damage.wallbangDamageScale != nil {
		scale := *damage.wallbangDamageScale
		if math.IsNaN(scale) || math.IsInf(scale, 0) || scale <= 0 {
			return unknown("invalid_damage_scale")
		}
		model.BaseDamage *= scale
	}
	matchedShot := uniqueWallbangShot(damage, shotIndex)
	if matchedShot == nil {
		return unknown("missing_or_ambiguous_shot")
	}
	victimPosition := damage.victimPosition
	if victimPosition == nil {
		return unknown("missing_position")
	}
	distance := euclideanDistance3D(matchedShot.X, matchedShot.Y, matchedShot.Z, victimPosition.X, victimPosition.Y, victimPosition.Z)
	if math.IsNaN(distance) || math.IsInf(distance, 0) {
		return unknown("invalid_position")
	}
	// Compare like with like: dmg_health is an integer damage report, while
	// HealthDamage is actual HP removed and can differ by one on CS2 servers.
	// Flooring here models the report, not the victim's health accumulator.
	expected := int(capExpectedHealthDamage(math.Floor(expectedHealthDamageHeuristic(model, damage.HitGroup, distance, damage.VictimArmor, victimPosition.HasHelmet)), damage.VictimHealth))
	observed := damage.ReportedHealthDamage
	if observed > damage.VictimHealth {
		observed = damage.VictimHealth // Overkill cannot diagnose penetration.
	}
	damage.HasWallbangHeuristicData = true
	damage.WallbangExpectedDamage, damage.WallbangDamageDeficit = expected, expected-observed
	damage.WallbangModelSource = source
	if observed > expected {
		return unknown("observed_above_expected")
	}
	if expected-observed >= 1 {
		damage.WallbangHeuristicReason = "damage_deficit"
		return true, "heuristic"
	}
	damage.WallbangHeuristicReason = "consistent_damage"
	return false, "heuristic"
}

// Bullets are hitscan. A shot from a previous frame, or multiple shots with
// the same player/weapon in one frame, cannot support a one-point comparison.
// The wider nearest-shot window is retained only for the existing first-shot metrics.
func uniqueWallbangShot(damage *Damage, index map[wallbangShotKey][]shotIndexEntry) *Shot {
	entries := index[wallbangShotKey{round: damage.RoundNumber, attacker: damage.AttackerSteamID64, weaponID: damage.WeaponUniqueID}]
	i := sort.Search(len(entries), func(i int) bool { return entries[i].frame >= damage.Frame })
	if i == len(entries) || entries[i].frame != damage.Frame || (i+1 < len(entries) && entries[i+1].frame == damage.Frame) {
		return nil
	}
	return entries[i].shot
}

// Snapshot scales at the hurt event; a later ConVar change must not rewrite
// earlier shots. Missing replicated scales use the standard game default (1),
// so even a successful comparison remains a heuristic.
func snapshotWallbangDamageScale(convars map[string]string, side common.Team, hitGroup events.HitGroup) *float64 {
	team := "t"
	if side == common.TeamCounterTerrorists {
		team = "ct"
	}
	part := "body"
	if isHeadHit(hitGroup) {
		part = "head"
	}
	scale := 1.0
	if text, exists := convars["mp_damage_scale_"+team+"_"+part]; exists {
		var err error
		scale, err = strconv.ParseFloat(text, 64)
		if err != nil {
			scale = math.NaN()
		}
	}
	return &scale
}

func buildWallbangHeuristicShotIndex(shots []*Shot) map[wallbangShotKey][]shotIndexEntry {
	index := make(map[wallbangShotKey][]shotIndexEntry)
	for _, shot := range shots {
		if shot == nil || shot.PlayerSteamID64 == 0 || shot.WeaponID == "" {
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
		// One armor point absorbs 1/0.5 health damage when armor runs out.
		healthDamage = postHitgroup - float64(victimArmor)/0.5
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
