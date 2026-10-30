package api

import "github.com/akiver/cs-demo-analyzer/pkg/api/constants"

// Only positive player_death penetration is evidence here. It does not supply
// the penetration count of earlier, nonlethal hits against the same victim.
func matchKillPenetrations(match *Match) {
	type key struct {
		damageMatchFrameKey
		weapon constants.WeaponName
	}
	damages := map[key][]*Damage{}
	for _, damage := range match.Damages {
		if damage == nil || damage.VictimNewHealth != 0 || damage.HealthDamage <= 0 || damage.AttackerSteamID64 == 0 || damage.AttackerSteamID64 == damage.VictimSteamID64 {
			continue
		}
		k := key{newDamageMatchFrameKeyFromDamage(damage), damage.WeaponName}
		damages[k] = append(damages[k], damage)
	}
	kills := map[key][]*Kill{}
	for _, kill := range match.Kills {
		if kill == nil || kill.PenetratedObjects <= 0 {
			continue
		}
		k := key{newDamageMatchFrameKey(kill.RoundNumber, kill.Frame, kill.KillerSteamID64, kill.VictimSteamID64), kill.WeaponName}
		kills[k] = append(kills[k], kill)
	}
	for k, candidates := range damages {
		if len(candidates) != 1 || len(kills[k]) != 1 {
			continue
		}
		damage := candidates[0]
		if damage.HasBulletDamageData {
			continue
		}
		damage.HasKillPenetrationData = true
		damage.NumPenetrations = kills[k][0].PenetratedObjects
	}
}
