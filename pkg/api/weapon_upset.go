package api

import (
	"math"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

// A per-model long-range boundary, selected as 25% range damage loss.
const weaponUpsetRetentionThreshold = 0.75

type WeaponUpsetEvidence struct {
	Rule                     string  `json:"rule"`
	DistanceUnits            float64 `json:"distanceUnits,omitempty"`
	RangeThresholdUnits      float64 `json:"rangeThresholdUnits,omitempty"`
	RangeModifier            float64 `json:"rangeModifier,omitempty"`
	DamageRetention          float64 `json:"damageRetention,omitempty"`
	DamageRetentionThreshold float64 `json:"damageRetentionThreshold,omitempty"`
	ModelSource              string  `json:"modelSource,omitempty"`
}

func weaponTypeByName(name constants.WeaponName) constants.WeaponType {
	for kind, known := range equipmentToWeaponName {
		if name == known {
			return getEquipmentWeaponType(common.Equipment{Type: kind})
		}
	}
	return constants.WeaponTypeUnknown
}

func smgUpsetRange(weapon constants.WeaponName, protocol int) (float64, constants.HeuristicWallbangWeaponDamageModel, string, bool) {
	model, source, known := constants.HeuristicWallbangModel(weapon, protocol)
	if !known || weaponTypeByName(weapon) != constants.WeaponTypeSMG || model.RangeModifier <= 0 || model.RangeModifier >= 1 || math.IsNaN(model.RangeModifier) {
		return 0, model, source, false
	}
	threshold := 500 * (math.Log(weaponUpsetRetentionThreshold) / math.Log(model.RangeModifier))
	return threshold, model, source, true
}

func classifyWeaponUpset(kill *Kill, protocol int) (bool, bool, *WeaponUpsetEvidence) {
	if !kill.isEnemyKill() || kill.IsKillerControllingBot || kill.IsVictimControllingBot || !kill.HasVictimActiveWeaponData {
		return false, false, nil
	}
	victimType := weaponTypeByName(kill.VictimActiveWeaponName)
	if victimType == constants.WeaponTypeUnknown || kill.WeaponType == constants.WeaponTypeUnknown || kill.WeaponType == "" {
		return false, false, nil
	}
	if kill.WeaponType == constants.WeaponTypePistol && isPrimaryWeaponType(victimType) {
		return true, true, &WeaponUpsetEvidence{Rule: "pistol_vs_primary"}
	}
	if kill.WeaponType != constants.WeaponTypeSMG || victimType != constants.WeaponTypeRifle {
		return true, false, nil
	}
	threshold, model, source, known := smgUpsetRange(kill.WeaponName, protocol)
	if !known || !kill.hasWeaponUpsetPositions {
		return false, false, nil
	}
	for _, value := range []float64{kill.KillerX, kill.KillerY, kill.KillerZ, kill.VictimX, kill.VictimY, kill.VictimZ} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false, false, nil
		}
	}
	// Kill.Distance follows a separate event/metre conversion path. Use raw
	// event-time world coordinates for the same units as the range model.
	distance := euclideanDistance3D(kill.KillerX, kill.KillerY, kill.KillerZ, kill.VictimX, kill.VictimY, kill.VictimZ)
	if math.IsNaN(distance) || math.IsInf(distance, 0) {
		return false, false, nil
	}
	evidence := &WeaponUpsetEvidence{Rule: "smg_vs_rifle_at_range", DistanceUnits: distance,
		RangeThresholdUnits: threshold, RangeModifier: model.RangeModifier,
		DamageRetention:          math.Pow(model.RangeModifier, distance/500),
		DamageRetentionThreshold: weaponUpsetRetentionThreshold, ModelSource: source}
	return true, evidence.DamageRetention <= weaponUpsetRetentionThreshold, evidence
}

func markWeaponUpsetKills(match *Match) {
	for _, kill := range match.Kills {
		if kill != nil {
			kill.HasWeaponUpsetData, kill.IsWeaponUpsetKill, kill.WeaponUpset = classifyWeaponUpset(kill, match.NetworkProtocol)
		}
	}
}

func (player *Player) WeaponUpsetKillCount() int {
	count := 0
	for _, kill := range player.match.Kills {
		if kill.isEnemyKill() && kill.KillerSteamID64 == player.SteamID64 && kill.HasWeaponUpsetData && kill.IsWeaponUpsetKill {
			count++
		}
	}
	return count
}

func (player *Player) WeaponUpsetDeathCount() int {
	count := 0
	for _, kill := range player.match.Kills {
		if kill.isEnemyKill() && kill.VictimSteamID64 == player.SteamID64 && kill.HasWeaponUpsetData && kill.IsWeaponUpsetKill {
			count++
		}
	}
	return count
}
