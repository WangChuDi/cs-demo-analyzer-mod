package api

import (
	"fmt"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

func smokeEventKey(event events.GrenadeEvent) string {
	if event.Projectile != nil {
		return "projectile:" + event.Projectile.UniqueID().String()
	}
	if event.Grenade != nil {
		return "weapon:" + event.Grenade.UniqueID().String()
	}
	return ""
}

func (analyzer *Analyzer) readSmokePlayer(player *common.Player, frame int) (smokePlayerSample, bool) {
	if !analyzer.isSource2 || player == nil || player.SteamID64 == 0 || player.IsBot || player.IsControllingBot() ||
		(player.Team != common.TeamTerrorists && player.Team != common.TeamCounterTerrorists) {
		return smokePlayerSample{}, false
	}
	eye, known := player.PositionEyes()
	if !known || !smokeFiniteVector(eye) {
		return smokePlayerSample{}, false
	}
	return smokePlayerSample{tick: analyzer.currentTick(), frame: frame, round: analyzer.currentRound.Number,
		id: player.SteamID64, name: player.Name, side: player.Team, eye: eye}, true
}

func (analyzer *Analyzer) sampleSmokeStats() {
	if analyzer.parser.GameState().IsFreezetimePeriod() || analyzer.currentRound.EndTick > 0 {
		analyzer.finishSmokeTracking("round_end")
		return
	}
	for _, player := range analyzer.parser.GameState().Participants().Playing() {
		if player == nil {
			continue
		}
		sample, known := analyzer.readSmokePlayer(player, analyzer.parser.CurrentFrame()-1)
		if !known || !player.IsAlive() {
			analyzer.breakSmokePlayer(player.SteamID64, "unavailable")
			continue
		}
		analyzer.updateSmokePlayer(sample, analyzer.parser.TickRate())
	}
}

func (analyzer *Analyzer) smokePlayerDeath(player *common.Player) {
	if player == nil {
		return
	}
	// A death event can already expose the corpse's changed view offset. End at
	// the last observed live sample instead of interpreting that change as motion.
	analyzer.breakSmokePlayer(player.SteamID64, "death")
}

func (analyzer *Analyzer) registerSmokeStatsHandlers() {
	analyzer.parser.RegisterEventHandler(func(events.RoundStart) { analyzer.finishSmokeTracking("round_end") })
	analyzer.parser.RegisterEventHandler(func(events.RoundEnd) { analyzer.finishSmokeTracking("round_end") })
	analyzer.parser.RegisterEventHandler(func(event events.SmokeStart) {
		if !analyzer.matchStarted() || !analyzer.isSource2 || !smokeFiniteVector(event.Position) {
			return
		}
		key, rate := smokeEventKey(event.GrenadeEvent), analyzer.parser.TickRate()
		if key == "" || rate <= 0 || !summonFinite(rate) {
			return
		}
		state := analyzer.smokeState()
		if old := state.clouds[key]; old != nil && old.EndReason == "" {
			if old.StartTick == analyzer.currentTick() {
				return
			}
			old.EndTick, old.EndReason = analyzer.currentTick(), "identity_reused"
		}
		cloud := &SmokeCloud{ID: fmt.Sprintf("%s@%d", key, analyzer.currentTick()), RoundNumber: analyzer.currentRound.Number,
			StartTick: analyzer.currentTick(), ReadyTick: float64(analyzer.currentTick()) + rate*smokeStatsBloomSeconds,
			X: event.Position.X, Y: event.Position.Y, Z: event.Position.Z}
		state.clouds[key] = cloud
		analyzer.match.SmokeClouds = append(analyzer.match.SmokeClouds, cloud)
	})
	analyzer.parser.RegisterEventHandler(func(event events.SmokeExpired) {
		state := analyzer.smokeState()
		if cloud := state.clouds[smokeEventKey(event.GrenadeEvent)]; cloud != nil && cloud.EndReason == "" {
			cloud.EndTick, cloud.EndReason = analyzer.currentTick(), "expired"
		}
	})
	addNade := func(kind string, event events.GrenadeEvent) {
		if !analyzer.matchStarted() || !smokeFiniteVector(event.Position) {
			return
		}
		side := common.TeamUnassigned
		if event.Thrower != nil {
			side = event.Thrower.Team
		}
		state := analyzer.smokeState()
		state.supports = append(state.supports, &smokeSupport{round: analyzer.currentRound.Number,
			SmokeAssistance: SmokeAssistance{Kind: kind, Tick: analyzer.currentTick(), EndTick: analyzer.currentTick(), Side: side, Position: event.Position}})
	}
	analyzer.parser.RegisterEventHandler(func(event events.FlashExplode) { addNade("flash", event.GrenadeEvent) })
	analyzer.parser.RegisterEventHandler(func(event events.HeExplode) { addNade("he", event.GrenadeEvent) })
	analyzer.parser.RegisterEventHandler(func(event events.InfernoStart) {
		if !analyzer.matchStarted() || event.Inferno == nil || event.Inferno.Entity == nil {
			return
		}
		position := event.Inferno.Entity.Position()
		if !smokeFiniteVector(position) {
			return
		}
		side := common.TeamUnassigned
		if thrower := event.Inferno.Thrower(); thrower != nil {
			side = thrower.Team
		}
		support := &smokeSupport{round: analyzer.currentRound.Number, SmokeAssistance: SmokeAssistance{Kind: "inferno", Tick: analyzer.currentTick(), Side: side, Position: position}}
		state := analyzer.smokeState()
		state.supports = append(state.supports, support)
		state.fires[event.Inferno.UniqueID().String()] = support
	})
	endFire := func(inferno *common.Inferno) {
		if inferno == nil {
			return
		}
		if fire := analyzer.smokeState().fires[inferno.UniqueID().String()]; fire != nil && fire.EndTick == 0 {
			fire.EndTick = analyzer.currentTick()
		}
	}
	analyzer.parser.RegisterEventHandler(func(event events.InfernoFireOut) { endFire(event.Inferno) })
	analyzer.parser.RegisterEventHandler(func(event events.InfernoExpired) { endFire(event.Inferno) })
}
