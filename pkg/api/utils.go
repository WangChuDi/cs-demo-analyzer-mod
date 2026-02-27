package api

import (
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

// getPlayerVelocity calculates player velocity using position deltas between ticks.
//
// Background: Frame vs Tick in Source 2 demos
//   - A "tick" is a discrete server simulation step (e.g. tick 31622, 31623, 31624).
//     At 64 tick rate, each tick = 1/64 seconds = 15.625ms.
//   - A "frame" is a parser iteration that fires the FrameDone event.
//     Multiple frames can share the same tick (duplicate-tick frames),
//     and ticks can be skipped (e.g. tick 31622 → 31624, gap of 1 tick).
//   - Entity properties (position, velocity) are updated per-tick, not per-frame.
//     So two consecutive frames with the same tick will have identical positions.
//
// Why position-delta instead of engine properties:
//   Engine properties m_vecVelocity / m_vecBaseVelocity are unreliable in CS2 demos
//   (often zero or stale). Position-delta is the only consistent method.
//
// Two-path velocity calculation:
//   Primary: velocity = (currentPos - lastPos) / (tickDelta * tickTime)
//     Uses the player's current position vs the last recorded position.
//     tickDelta accounts for tick gaps (not always 1).
//
//   Fallback: velocity = (lastPos - prevPos) / (tickDelta * tickTime)
//     When currentPos == lastPos (zero delta), this is NOT because the player
//     is stationary — it's because of entity update ordering:
//     Grenade WeaponFire events fire during entity creation (datatables.go:714-718),
//     BEFORE FrameDone, so the player pawn position hasn't been updated yet
//     for the current tick. In this case we use the previous two known positions.
//
// Position history is maintained in the FrameDone handler (analyzer.go):
//   - lastPlayersPosition / lastPlayersTick: most recent position and tick
//   - prevPlayersPosition / prevPlayersTick: position and tick before that
//   - Rotation (prev = last) only happens when tick changes, preventing
//     duplicate-tick frames from corrupting the two-frame history.
func getPlayerVelocity(p *common.Player, analyzer *Analyzer) r3.Vector {
	if p == nil {
		return r3.Vector{}
	}

	tickTime := analyzer.parser.TickTime().Seconds()
	if tickTime <= 0 {
		return r3.Vector{}
	}

	currentTick := analyzer.currentTick()
	lastPos, hasLast := analyzer.match.lastPlayersPosition[p.SteamID64]
	if hasLast {
		lastTick := analyzer.match.lastPlayersTick[p.SteamID64]
		tickDelta := currentTick - lastTick
		if tickDelta > 0 {
			elapsed := float64(tickDelta) * tickTime
			currentPos := p.Position()
			velocity := r3.Vector{
				X: (currentPos.X - lastPos.X) / elapsed,
				Y: (currentPos.Y - lastPos.Y) / elapsed,
				Z: (currentPos.Z - lastPos.Z) / elapsed,
			}
			if velocity.X != 0 || velocity.Y != 0 || velocity.Z != 0 {
				return velocity
			}
		}
		// currentPos - lastPos == 0, likely due to entity update ordering
		// (e.g. grenade projectile created before player pawn position updated).
		// Fallback to lastPos - prevPos as the velocity estimate.
		if prevPos, hasPrev := analyzer.match.prevPlayersPosition[p.SteamID64]; hasPrev {
			prevTick := analyzer.match.prevPlayersTick[p.SteamID64]
			lastTick := analyzer.match.lastPlayersTick[p.SteamID64]
			tickDelta := lastTick - prevTick
			if tickDelta > 0 {
				elapsed := float64(tickDelta) * tickTime
				return r3.Vector{
					X: (lastPos.X - prevPos.X) / elapsed,
					Y: (lastPos.Y - prevPos.Y) / elapsed,
					Z: (lastPos.Z - prevPos.Z) / elapsed,
				}
			}
		}
	}

	return r3.Vector{}
}

func getPlayerPositionEyes(p *common.Player) r3.Vector {
	if p == nil {
		return r3.Vector{}
	}
	pos := p.Position()

	var offset r3.Vector
	pawn := p.PlayerPawnEntity()
	if pawn != nil {
		if val, ok := pawn.PropertyValue("m_vecViewOffset"); ok {
			offset = val.R3Vec()
		}
	} else if p.Entity != nil {
		if val, ok := p.Entity.PropertyValue("m_vecViewOffset"); ok {
			offset = val.R3Vec()
		}
	}

	return pos.Add(offset)
}
