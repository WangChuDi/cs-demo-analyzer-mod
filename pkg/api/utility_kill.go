package api

import (
	"math"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

type posthumousDeath struct {
	tick, frame, round int
}

func (kill *Kill) isEnemyKill() bool {
	return kill != nil && kill.KillerSteamID64 != 0 && kill.VictimSteamID64 != 0 &&
		!kill.IsSuicide() && !kill.IsTeamKill() &&
		(kill.KillerSide == common.TeamTerrorists || kill.KillerSide == common.TeamCounterTerrorists) &&
		(kill.VictimSide == common.TeamTerrorists || kill.VictimSide == common.TeamCounterTerrorists)
}

func isThrowableUtility(weapon constants.WeaponName) bool {
	switch weapon {
	case constants.WeaponDecoy, constants.WeaponFlashbang, constants.WeaponHEGrenade,
		constants.WeaponIncendiary, constants.WeaponMolotov, constants.WeaponSmoke:
		return true
	}
	return false
}

func (analyzer *Analyzer) registerUtilityKillHandlers() {
	analyzer.parser.RegisterEventHandler(func(events.RoundStart) {
		clear(analyzer.posthumousDeaths)
		analyzer.pendingUtilityImpactKill = nil
	})
	// v6 has no typed PlayerSpawn event. When recorded, this generic event
	// invalidates the previous life even if the same controller/pawn is reused.
	analyzer.parser.RegisterEventHandler(func(event events.GenericGameEvent) {
		if event.Name == "player_death" {
			analyzer.applyUtilityImpactDeathEvent(event)
			return
		}
		if event.Name != "player_spawn" || event.Data["userid"] == nil {
			return
		}
		player := analyzer.parser.GameState().Participants().ByUserID()[int(event.Data["userid"].GetValShort())]
		if player != nil {
			analyzer.clearPosthumousLife(player.SteamID64)
			delete(analyzer.summonTrackers, player.SteamID64)
		}
	})
}

func (analyzer *Analyzer) clearPosthumousLife(steamID uint64) {
	delete(analyzer.posthumousDeaths, steamID)
}

// trackUtilityKill uses only event-time state, independently of -positions.
// Requiring a strictly earlier tick excludes same-tick grenade trades, even
// when the death event was dispatched first or arrived in an earlier frame.
func (analyzer *Analyzer) trackUtilityKill(kill *Kill, killerAlive bool) {
	kill.IsVictimHoldingUtility = kill.HasVictimActiveWeaponData && isThrowableUtility(kill.VictimActiveWeaponName)
	kill.IsPosthumousKill, kill.PosthumousDeathTick, kill.PosthumousDelaySeconds = false, 0, 0
	if killerAlive || kill.IsKillerControllingBot {
		analyzer.clearPosthumousLife(kill.KillerSteamID64)
	}
	death, observed := analyzer.posthumousDeaths[kill.KillerSteamID64]
	rate := analyzer.match.TickRate
	if kill.isEnemyKill() && isThrowableUtility(kill.WeaponName) && !killerAlive && !kill.IsKillerControllingBot &&
		observed && death.round == kill.RoundNumber && death.tick < kill.Tick && death.frame <= kill.Frame &&
		rate > 0 && !math.IsNaN(rate) && !math.IsInf(rate, 0) {
		delay := (float64(kill.Tick) - float64(death.tick)) / rate
		if !math.IsInf(delay, 0) {
			kill.IsPosthumousKill = true
			kill.PosthumousDeathTick = death.tick
			kill.PosthumousDelaySeconds = delay
		}
	}
	// Record all observed player deaths, including environmental/self deaths;
	// the later kill must still be an enemy utility kill to qualify.
	if kill.VictimSteamID64 != 0 && !kill.IsVictimControllingBot {
		if analyzer.posthumousDeaths == nil {
			analyzer.posthumousDeaths = make(map[uint64]posthumousDeath)
		}
		analyzer.posthumousDeaths[kill.VictimSteamID64] = posthumousDeath{tick: kill.Tick, frame: kill.Frame, round: kill.RoundNumber}
	}
}

func (player *Player) UtilityInHandDeathCount() int {
	count := 0
	for _, kill := range player.match.Kills {
		if kill.isEnemyKill() && kill.VictimSteamID64 == player.SteamID64 && kill.IsVictimHoldingUtility {
			count++
		}
	}
	return count
}

func (player *Player) PosthumousKillCount() int {
	count := 0
	for _, kill := range player.match.Kills {
		if kill.isEnemyKill() && kill.KillerSteamID64 == player.SteamID64 && kill.IsPosthumousKill {
			count++
		}
	}
	return count
}
