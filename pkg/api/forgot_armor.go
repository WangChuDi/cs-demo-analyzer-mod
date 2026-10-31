package api

import (
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

func (economy *PlayerEconomy) setArmorCheck(tick, armor int, hasArmor bool, weapons []*common.Equipment) {
	economy.ArmorCheckTick = tick
	economy.ArmorAtFreezeEnd = armor
	economy.HasArmorCheckData = hasArmor && armor >= 0 && armor <= 100
	economy.PrimaryWeaponAtFreezeEnd = ""
	economy.ForgotArmor = false
	for _, weapon := range weapons {
		if weapon == nil {
			economy.HasArmorCheckData = false
			continue
		}
		if !isPrimaryWeaponType(getEquipmentWeaponType(*weapon)) {
			continue
		}
		name, known := equipmentToWeaponName[weapon.Type]
		if !known || economy.PrimaryWeaponAtFreezeEnd != "" {
			economy.HasArmorCheckData = false
		}
		economy.PrimaryWeaponAtFreezeEnd = name
	}
	if economy.HasArmorCheckData && armor == 0 && economy.PrimaryWeaponAtFreezeEnd != "" {
		for _, weapon := range weapons {
			if isPrimaryWeaponType(getEquipmentWeaponType(*weapon)) {
				economy.ForgotArmor = getEquipmentWeaponType(*weapon) != constants.WeaponTypeSniper
				break
			}
		}
	}
}

// Capture once at the frame containing the freeze-end transition. The existing
// seven-second economy refresh must not overwrite this loadout snapshot.
func (analyzer *Analyzer) captureArmorChecks() {
	if analyzer.armorCheckRound == 0 {
		return
	}
	defer func() { analyzer.armorCheckRound = 0 }()
	if analyzer.currentRound.Number != analyzer.armorCheckRound || analyzer.parser.GameState().IsFreezetimePeriod() {
		return
	}
	for _, player := range analyzer.parser.GameState().Participants().Playing() {
		if player == nil || player.IsBot || player.IsControllingBot() || !player.IsAlive() || player.SteamID64 == 0 {
			continue
		}
		economy := analyzer.match.GetPlayerEconomyAtRound(player.Name, player.SteamID64, analyzer.currentRound.Number)
		if economy == nil {
			continue
		}
		entity := player.Entity
		if analyzer.isSource2 {
			entity = player.PlayerPawnEntity()
		}
		if entity == nil {
			continue
		}
		armor, known := entityIntProperty(entity, "m_ArmorValue")
		economy.setArmorCheck(analyzer.armorCheckTick, armor, known, player.Weapons())
	}
}

func (player *Player) ForgotArmorCount() int {
	count := 0
	for _, economy := range player.match.PlayerEconomies {
		if economy.SteamID64 == player.SteamID64 && economy.HasArmorCheckData && economy.ForgotArmor {
			count++
		}
	}
	return count
}
