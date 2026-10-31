package api

import (
	"slices"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
)

type WeaponUsageStats struct {
	UsedWeaponCount          int                    `json:"usedWeaponCount"`
	UsedPrimaryWeaponCount   int                    `json:"usedPrimaryWeaponCount"`
	UsedSecondaryWeaponCount int                    `json:"usedSecondaryWeaponCount"`
	UsedWeapons              []constants.WeaponName `json:"usedWeapons"`
}

func isPrimaryWeaponType(kind constants.WeaponType) bool {
	switch kind {
	case constants.WeaponTypeRifle, constants.WeaponTypeSniper, constants.WeaponTypeSMG,
		constants.WeaponTypeShotgun, constants.WeaponTypeMachineGun:
		return true
	}
	return false
}

func isFirearmType(kind constants.WeaponType) bool {
	return isPrimaryWeaponType(kind) || kind == constants.WeaponTypePistol
}

// WeaponUsage counts distinct firearm models actually fired, not inventory
// instances, pickups, skins or shots. Default pistols are excluded on both sides.
func (player *Player) WeaponUsage() WeaponUsageStats {
	usage := WeaponUsageStats{UsedWeapons: []constants.WeaponName{}}
	seen := map[constants.WeaponName]bool{}
	for _, shot := range player.match.Shots {
		if shot == nil || shot.PlayerSteamID64 != player.SteamID64 || shot.IsPlayerControllingBot || !isFirearmType(shot.WeaponType) {
			continue
		}
		switch shot.WeaponName {
		case "", constants.WeaponUnknown, constants.WeaponGlock, constants.WeaponUSP, constants.WeaponP2000:
			continue
		}
		if seen[shot.WeaponName] {
			continue
		}
		seen[shot.WeaponName] = true
		usage.UsedWeapons = append(usage.UsedWeapons, shot.WeaponName)
		if isPrimaryWeaponType(shot.WeaponType) {
			usage.UsedPrimaryWeaponCount++
		} else {
			usage.UsedSecondaryWeaponCount++
		}
	}
	slices.Sort(usage.UsedWeapons)
	usage.UsedWeaponCount = len(usage.UsedWeapons)
	return usage
}
