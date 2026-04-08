package api

import (
	"math"
	"sort"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/akiver/cs-demo-analyzer/pkg/api/funData"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

const (
	awpHoldReactionWindowSeconds    = 1.0
	awpHoldEmptyShotWindowSeconds   = 0.5
	awpHoldFacingAngleThreshold     = 10.0
	defaultTickRateForDerivedTables = 64.0
)

type AwpHoldDeath struct {
	Frame                           int                  `json:"frame"`
	Tick                            int                  `json:"tick"`
	RoundNumber                     int                  `json:"roundNumber"`
	KillerName                      string               `json:"killerName"`
	KillerSteamID64                 uint64               `json:"killerSteamId"`
	KillerSide                      common.Team          `json:"killerSide"`
	KillerTeamName                  string               `json:"killerTeamName"`
	VictimName                      string               `json:"victimName"`
	VictimSteamID64                 uint64               `json:"victimSteamId"`
	VictimSide                      common.Team          `json:"victimSide"`
	VictimTeamName                  string               `json:"victimTeamName"`
	KillerWeaponName                constants.WeaponName `json:"killerWeaponName"`
	VictimReactionShotFrame         int                  `json:"victimReactionFrame"`
	HasPreDeathVictimAwpShot        bool                 `json:"hasPreDeathVictimAwpShot"`
	HasPostDeathVictimAttackTrigger bool                 `json:"hasPostDeathVictimAttackTrigger"`
	ShotOffsetFrame                 int                  `json:"offsetFrame"`
	VictimX                         float64              `json:"victimX"`
	VictimY                         float64              `json:"victimY"`
	VictimZ                         float64              `json:"victimZ"`
	KillerX                         float64              `json:"killerX"`
	KillerY                         float64              `json:"killerY"`
	KillerZ                         float64              `json:"killerZ"`
	VictimVelocityX                 float64              `json:"victimVelocityX"`
	VictimVelocityY                 float64              `json:"victimVelocityY"`
	VictimVelocityZ                 float64              `json:"victimVelocityZ"`
	KillerVelocityX                 float64              `json:"killerVelocityX"`
	KillerVelocityY                 float64              `json:"killerVelocityY"`
	KillerVelocityZ                 float64              `json:"killerVelocityZ"`
	KillerSpeed2D                   float64              `json:"killerSpeed2d"`
	KillerSpeedBucket               string               `json:"killerSpeedBucket"`
	VictimSpeed2D                   float64              `json:"victimSpeed2d"`
	VictimSpeedBucket               string               `json:"victimSpeedBucket"`
	IsVictimSlow                    bool                 `json:"isVictimSlow"`
	IsVictimScoped                  bool                 `json:"isVictimScoped"`
	IsVictimFacingKiller            bool                 `json:"isVictimFacingKiller"`
	VictimFacingKillerAngleDeg      float64              `json:"victimFacingKillerAngleDeg"`
}

type roundPlayerKey struct {
	roundNumber int
	steamID64   uint64
}

type playerPositionSnapshot struct {
	position    *PlayerPosition
	velocity    r3.Vector
	hasVelocity bool
}

func generateAwpHoldDeaths(match *Match) {
	if match == nil {
		return
	}

	tickRate := match.TickRate
	if tickRate <= 0 {
		tickRate = defaultTickRateForDerivedTables
	}
	frameRate := match.FrameRate
	if frameRate <= 0 {
		frameRate = tickRate
	}
	reactionWindowTicks := max(1, int(tickRate*awpHoldReactionWindowSeconds))
	reactionWindowFrames := max(1, int(frameRate*awpHoldReactionWindowSeconds))
	emptyShotWindowTicks := max(1, int(tickRate*awpHoldEmptyShotWindowSeconds))
	emptyShotWindowFrames := max(1, int(frameRate*awpHoldEmptyShotWindowSeconds))

	victimAwpShotsByRoundPlayer := buildVictimAwpShotsByRoundPlayer(match)
	victimButtonsByRoundPlayer := buildPlayerButtonsByRoundPlayer(match)
	playerPositionsByRoundPlayer := buildPlayerPositionsByRoundPlayer(match)

	derived := make([]*AwpHoldDeath, 0, len(match.Kills))
	for _, kill := range match.Kills {
		if kill == nil || kill.KillerSteamID64 == 0 || kill.VictimSteamID64 == 0 || kill.KillerSteamID64 == kill.VictimSteamID64 {
			continue
		}

		key := roundPlayerKey{roundNumber: kill.RoundNumber, steamID64: kill.VictimSteamID64}
		victimAwpShots := victimAwpShotsByRoundPlayer[key]
		victimReactionShot, offsetFrame, _ := nearestShotOffset(
			kill.Frame,
			kill.Tick,
			victimAwpShots,
			emptyShotWindowFrames,
			emptyShotWindowTicks,
		)
		hasPostDeathAttack := hasPostDeathAttackWithinWindow(
			kill.Frame,
			kill.Tick,
			victimButtonsByRoundPlayer[key],
			reactionWindowFrames,
			reactionWindowTicks,
		)
		victimSnapshot := nearestPlayerSnapshotAtOrBefore(kill.Tick, playerPositionsByRoundPlayer[key], tickRate)
		killerSnapshot := nearestPlayerSnapshotAtOrBefore(kill.Tick, playerPositionsByRoundPlayer[roundPlayerKey{roundNumber: kill.RoundNumber, steamID64: kill.KillerSteamID64}], tickRate)

		victimPosition := r3.Vector{X: kill.VictimX, Y: kill.VictimY, Z: kill.VictimZ}
		if victimSnapshot.position != nil {
			victimPosition = r3.Vector{X: victimSnapshot.position.X, Y: victimSnapshot.position.Y, Z: victimSnapshot.position.Z}
		}
		killerPosition := r3.Vector{X: kill.KillerX, Y: kill.KillerY, Z: kill.KillerZ}
		if killerSnapshot.position != nil {
			killerPosition = r3.Vector{X: killerSnapshot.position.X, Y: killerSnapshot.position.Y, Z: killerSnapshot.position.Z}
		}

		victimWeaponName := kill.VictimActiveWeaponName
		isVictimScoped := kill.VictimIsScoped
		victimVelocity := r3.Vector{X: kill.VictimVelocityX, Y: kill.VictimVelocityY, Z: kill.VictimVelocityZ}
		killerVelocity := r3.Vector{X: kill.KillerVelocityX, Y: kill.KillerVelocityY, Z: kill.KillerVelocityZ}
		victimYaw := kill.VictimYaw
		if victimSnapshot.position != nil {
			victimWeaponName = victimSnapshot.position.ActiveWeaponName
			isVictimScoped = victimSnapshot.position.IsScoping
			if victimSnapshot.hasVelocity {
				victimVelocity = victimSnapshot.velocity
			}
			victimYaw = victimSnapshot.position.Yaw
		}
		if killerSnapshot.position != nil && killerSnapshot.hasVelocity {
			killerVelocity = killerSnapshot.velocity
		}
		isVictimFacingKiller, victimFacingKillerAngleDeg := victimFacingKiller(victimYaw, victimPosition, killerPosition)

		victimSpeed2D := math.Sqrt(victimVelocity.X*victimVelocity.X + victimVelocity.Y*victimVelocity.Y)
		killerSpeed2D := math.Sqrt(killerVelocity.X*killerVelocity.X + killerVelocity.Y*killerVelocity.Y)
		isVictimSlow := victimSpeed2D <= constants.WeaponAccurateSpeed[constants.WeaponAWP]
		if victimWeaponName != constants.WeaponAWP || !isVictimScoped || !isVictimFacingKiller || !isVictimSlow {
			continue
		}

		reactionShotFrame := 0
		hasPreDeathVictimAwpShot := victimReactionShot != nil
		hasPostDeathVictimAttackTrigger := hasPostDeathAttack
		if victimReactionShot != nil {
			reactionShotFrame = victimReactionShot.Frame
		}

		derived = append(derived, &AwpHoldDeath{
			Frame:                           kill.Frame,
			Tick:                            kill.Tick,
			RoundNumber:                     kill.RoundNumber,
			KillerName:                      kill.KillerName,
			KillerSteamID64:                 kill.KillerSteamID64,
			KillerSide:                      kill.KillerSide,
			KillerTeamName:                  kill.KillerTeamName,
			VictimName:                      kill.VictimName,
			VictimSteamID64:                 kill.VictimSteamID64,
			VictimSide:                      kill.VictimSide,
			VictimTeamName:                  kill.VictimTeamName,
			KillerWeaponName:                kill.WeaponName,
			VictimReactionShotFrame:         reactionShotFrame,
			HasPreDeathVictimAwpShot:        hasPreDeathVictimAwpShot,
			HasPostDeathVictimAttackTrigger: hasPostDeathVictimAttackTrigger,
			ShotOffsetFrame:                 offsetFrame,
			VictimX:                         victimPosition.X,
			VictimY:                         victimPosition.Y,
			VictimZ:                         victimPosition.Z,
			KillerX:                         killerPosition.X,
			KillerY:                         killerPosition.Y,
			KillerZ:                         killerPosition.Z,
			VictimVelocityX:                 victimVelocity.X,
			VictimVelocityY:                 victimVelocity.Y,
			VictimVelocityZ:                 victimVelocity.Z,
			KillerVelocityX:                 killerVelocity.X,
			KillerVelocityY:                 killerVelocity.Y,
			KillerVelocityZ:                 killerVelocity.Z,
			KillerSpeed2D:                   killerSpeed2D,
			KillerSpeedBucket:               classifyMovementBucket(killerSpeed2D),
			VictimSpeed2D:                   victimSpeed2D,
			VictimSpeedBucket:               classifyMovementBucket(victimSpeed2D),
			IsVictimSlow:                    isVictimSlow,
			IsVictimScoped:                  isVictimScoped,
			IsVictimFacingKiller:            isVictimFacingKiller,
			VictimFacingKillerAngleDeg:      victimFacingKillerAngleDeg,
		})
	}

	match.AwpHoldDeaths = derived
}

func buildVictimAwpShotsByRoundPlayer(match *Match) map[roundPlayerKey][]*Shot {
	shotsByRoundPlayer := make(map[roundPlayerKey][]*Shot)
	for _, shot := range match.Shots {
		if shot == nil || shot.WeaponName != constants.WeaponAWP {
			continue
		}

		key := roundPlayerKey{roundNumber: shot.RoundNumber, steamID64: shot.PlayerSteamID64}
		shotsByRoundPlayer[key] = append(shotsByRoundPlayer[key], shot)
	}

	for key := range shotsByRoundPlayer {
		sort.Slice(shotsByRoundPlayer[key], func(i int, j int) bool {
			if shotsByRoundPlayer[key][i].Frame == shotsByRoundPlayer[key][j].Frame {
				return shotsByRoundPlayer[key][i].Tick < shotsByRoundPlayer[key][j].Tick
			}

			return shotsByRoundPlayer[key][i].Frame < shotsByRoundPlayer[key][j].Frame
		})
	}

	return shotsByRoundPlayer
}

func buildPlayerButtonsByRoundPlayer(match *Match) map[roundPlayerKey][]*funData.PlayerButtons {
	buttonsByRoundPlayer := make(map[roundPlayerKey][]*funData.PlayerButtons)
	for _, buttons := range match.PlayerButtons {
		if buttons == nil {
			continue
		}

		key := roundPlayerKey{roundNumber: buttons.RoundNumber, steamID64: buttons.SteamID64}
		buttonsByRoundPlayer[key] = append(buttonsByRoundPlayer[key], buttons)
	}

	for key := range buttonsByRoundPlayer {
		sort.Slice(buttonsByRoundPlayer[key], func(i int, j int) bool {
			if buttonsByRoundPlayer[key][i].Frame == buttonsByRoundPlayer[key][j].Frame {
				return buttonsByRoundPlayer[key][i].Tick < buttonsByRoundPlayer[key][j].Tick
			}

			return buttonsByRoundPlayer[key][i].Frame < buttonsByRoundPlayer[key][j].Frame
		})
	}

	return buttonsByRoundPlayer
}

func buildPlayerPositionsByRoundPlayer(match *Match) map[roundPlayerKey][]*PlayerPosition {
	positionsByRoundPlayer := make(map[roundPlayerKey][]*PlayerPosition)
	for _, position := range match.PlayerPositions {
		if position == nil {
			continue
		}

		key := roundPlayerKey{roundNumber: position.RoundNumber, steamID64: position.SteamID64}
		positionsByRoundPlayer[key] = append(positionsByRoundPlayer[key], position)
	}

	for key := range positionsByRoundPlayer {
		sort.Slice(positionsByRoundPlayer[key], func(i int, j int) bool {
			if positionsByRoundPlayer[key][i].Frame == positionsByRoundPlayer[key][j].Frame {
				return positionsByRoundPlayer[key][i].Tick < positionsByRoundPlayer[key][j].Tick
			}

			return positionsByRoundPlayer[key][i].Frame < positionsByRoundPlayer[key][j].Frame
		})
	}

	return positionsByRoundPlayer
}

func nearestShotOffset(killFrame int, killTick int, shots []*Shot, emptyShotWindowFrames int, emptyShotWindowTicks int) (*Shot, int, int) {
	if len(shots) == 0 {
		return nil, 0, 0
	}

	firstAfterIndex := sort.Search(len(shots), func(index int) bool {
		if shots[index].Frame == killFrame {
			return shots[index].Tick > killTick
		}

		return shots[index].Frame > killFrame
	})

	var nearestBefore *Shot
	beforeFrameDiff := emptyShotWindowFrames + 1
	beforeTickDiff := emptyShotWindowTicks + 1
	if firstAfterIndex > 0 {
		candidate := shots[firstAfterIndex-1]
		frameDiff := killFrame - candidate.Frame
		diff := killTick - candidate.Tick
		if frameDiff >= 0 && frameDiff <= emptyShotWindowFrames && diff >= 0 && diff <= emptyShotWindowTicks {
			nearestBefore = candidate
			beforeFrameDiff = frameDiff
			beforeTickDiff = diff
		}
	}

	if nearestBefore != nil {
		beforeOffsetFrame := -beforeFrameDiff
		beforeOffsetTick := -beforeTickDiff
		if beforeFrameDiff == 0 && beforeTickDiff == 0 {
			beforeOffsetTick = -1
		}

		return nearestBefore, beforeOffsetFrame, beforeOffsetTick
	}

	return nil, 0, 0
}

func hasPostDeathAttackWithinWindow(killFrame int, killTick int, buttons []*funData.PlayerButtons, reactionWindowFrames int, reactionWindowTicks int) bool {
	if len(buttons) == 0 {
		return false
	}

	firstAfterIndex := sort.Search(len(buttons), func(index int) bool {
		if buttons[index].Frame == killFrame {
			return buttons[index].Tick > killTick
		}

		return buttons[index].Frame > killFrame
	})

	var previousMask uint64
	if firstAfterIndex > 0 {
		previousMask = buttons[firstAfterIndex-1].Buttons
	}

	for index := firstAfterIndex; index < len(buttons); index++ {
		candidate := buttons[index]
		frameDiff := candidate.Frame - killFrame
		tickDiff := candidate.Tick - killTick
		if frameDiff < 0 || tickDiff <= 0 {
			continue
		}
		if frameDiff > reactionWindowFrames || tickDiff > reactionWindowTicks {
			break
		}

		hasAttack := candidate.Buttons&uint64(common.ButtonAttack) != 0
		hadAttack := previousMask&uint64(common.ButtonAttack) != 0
		if hasAttack && !hadAttack {
			return true
		}

		previousMask = candidate.Buttons
	}

	return false
}

func nearestPlayerSnapshotAtOrBefore(tick int, positions []*PlayerPosition, tickRate float64) playerPositionSnapshot {
	if len(positions) == 0 {
		return playerPositionSnapshot{}
	}

	firstAfterIndex := sort.Search(len(positions), func(index int) bool {
		return positions[index].Tick > tick
	})
	positionIndex := firstAfterIndex - 1
	if positionIndex < 0 {
		return playerPositionSnapshot{}
	}

	position := positions[positionIndex]
	velocity := r3.Vector{}
	hasVelocity := false
	if positionIndex > 0 {
		previousPosition := positions[positionIndex-1]
		tickDelta := position.Tick - previousPosition.Tick
		if tickDelta > 0 {
			secondsDelta := float64(tickDelta) / tickRate
			if secondsDelta > 0 {
				hasVelocity = true
				velocity = r3.Vector{
					X: (position.X - previousPosition.X) / secondsDelta,
					Y: (position.Y - previousPosition.Y) / secondsDelta,
					Z: (position.Z - previousPosition.Z) / secondsDelta,
				}
			}
		}
	}

	return playerPositionSnapshot{position: position, velocity: velocity, hasVelocity: hasVelocity}
}

func victimFacingKiller(victimYaw float32, victimPosition r3.Vector, killerPosition r3.Vector) (bool, float64) {
	directionToKiller := r3.Vector{X: killerPosition.X - victimPosition.X, Y: killerPosition.Y - victimPosition.Y}
	distance := math.Sqrt(directionToKiller.X*directionToKiller.X + directionToKiller.Y*directionToKiller.Y)
	if distance == 0 {
		return false, -1
	}

	directionToKiller.X /= distance
	directionToKiller.Y /= distance

	victimYawRadians := float64(victimYaw) * (math.Pi / 180.0)
	victimForward := r3.Vector{X: math.Cos(victimYawRadians), Y: math.Sin(victimYawRadians)}

	dot := victimForward.X*directionToKiller.X + victimForward.Y*directionToKiller.Y
	if dot > 1 {
		dot = 1
	} else if dot < -1 {
		dot = -1
	}

	angleDeg := math.Acos(dot) * (180.0 / math.Pi)
	return angleDeg <= awpHoldFacingAngleThreshold, angleDeg
}

func classifyMovementBucket(speed2D float64) string {
	if speed2D == 0 {
		return "standing"
	}

	if speed2D < 80 {
		return "step"
	}

	if speed2D < 180 {
		return "walk"
	}

	return "run"
}
