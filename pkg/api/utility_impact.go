package api

import (
	"strings"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

// v6 dispatches the original generic player_death immediately after its typed
// Kill. Read that event's weapon name: typed equipment merges inferno into the
// grenade family and may reuse a thrown equipment instance's OriginalString.
func (analyzer *Analyzer) applyUtilityImpactDeathEvent(event events.GenericGameEvent) {
	if event.Name != "player_death" {
		return
	}
	kill := analyzer.pendingUtilityImpactKill
	analyzer.pendingUtilityImpactKill = nil
	if kill == nil || kill.Frame != analyzer.parser.CurrentFrame() || kill.Tick != analyzer.currentTick() {
		return
	}
	if weapon := event.Data["weapon"]; weapon != nil {
		kill.setUtilityImpactData(weapon.GetValString())
	}
}

// Stock CS2 fire damage is named inferno; the Molotov/incendiary projectile
// names identify the physical hit. Flashbang/smoke have no damaging detonation.
// HE/decoy cannot be classified from the mapped family or damage amount alone;
// their lifecycle evidence is resolved after parsing all events.
func (kill *Kill) setUtilityImpactData(rawWeapon string) {
	kill.RawWeaponName = rawWeapon
	kill.HasUtilityImpactData, kill.IsUtilityImpactKill = false, false
	kill.UtilityImpactSource = utilityImpactSourceUnavailable
	kill.UtilityImpactProjectileID, kill.UtilityImpactEffectTick, kill.UtilityImpactEffect = "", 0, ""
	weapon := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(rawWeapon)), "weapon_")
	switch weapon {
	case "flashbang", "flashbang_projectile", "flashbangprojectile",
		"smokegrenade", "smokegrenade_projectile", "smokegrenadeprojectile",
		"molotov", "molotovgrenade", "molotov_projectile", "molotovprojectile",
		"incgrenade", "incendiarygrenade", "incgrenade_projectile", "incgrenadeprojectile":
		kill.HasUtilityImpactData, kill.IsUtilityImpactKill = true, true
	case "inferno":
		kill.HasUtilityImpactData = true
	default:
		equipment := common.MapEquipment(weapon)
		if equipment != common.EqUnknown && equipment.Class() != common.EqClassGrenade {
			kill.HasUtilityImpactData = true
		}
	}
	if kill.HasUtilityImpactData {
		kill.UtilityImpactSource = utilityImpactSourceRawWeapon
	}
}

// UtilityImpactDeathCount counts enemy projectile-impact deaths with known
// cause. Unclassified HE/decoy deaths are not inferred from their damage amount.
func (player *Player) UtilityImpactDeathCount() int {
	count := 0
	for _, kill := range player.match.Kills {
		if kill.isEnemyKill() && kill.VictimSteamID64 == player.SteamID64 &&
			kill.HasUtilityImpactData && kill.IsUtilityImpactKill {
			count++
		}
	}
	return count
}
