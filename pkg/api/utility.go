package api

import (
	"math"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

type UtilityType string

const (
	UtilityTypeSmoke UtilityType = "smoke"
	UtilityTypeFlash UtilityType = "flash"
	UtilityTypeHE    UtilityType = "he"
	UtilityTypeDecoy UtilityType = "decoy"
)

type UtilityThrowType string

const (
	UtilityThrowTypeLeftClick    UtilityThrowType = "left_click"
	UtilityThrowTypeDoubleClick  UtilityThrowType = "double_click"
	UtilityThrowTypeJumpThrow    UtilityThrowType = "jump_throw"
	UtilityThrowTypeWJumpThrow   UtilityThrowType = "w_jump_throw"
	UtilityThrowTypeRunThrow     UtilityThrowType = "run_throw"
	UtilityThrowTypeRunJumpThrow UtilityThrowType = "run_jump_throw"
)

const throwStandingSpeedMax = 5.0

type Utility struct {
	Frame            int              `json:"frame"`
	Tick             int              `json:"tick"`
	RoundNumber      int              `json:"roundNumber"`
	UtilityType      UtilityType      `json:"utilityType"`
	ThrowType        UtilityThrowType `json:"throwType"`
	GrenadeID        string           `json:"grenadeId"`
	ProjectileID     int64            `json:"projectileId"`
	X                float64          `json:"x"`
	Y                float64          `json:"y"`
	Z                float64          `json:"z"`
	ThrowerSteamID64 uint64           `json:"throwerSteamId"`
	ThrowerName      string           `json:"throwerName"`
	ThrowerSide      common.Team      `json:"throwerSide"`
	ThrowerTeamName  string           `json:"throwerTeamName"`
	ThrowerVelocityX float64          `json:"throwerVelocityX"`
	ThrowerVelocityY float64          `json:"throwerVelocityY"`
	ThrowerVelocityZ float64          `json:"throwerVelocityZ"`
	ThrowerPitch     float32          `json:"throwerPitch"`
	ThrowerYaw       float32          `json:"throwerYaw"`
}

func newUtilityFromShot(analyzer *Analyzer, shot *Shot, thrower *common.Player) *Utility {
	if shot == nil {
		return nil
	}

	var utilityType UtilityType
	switch shot.WeaponName {
	case constants.WeaponSmoke:
		utilityType = UtilityTypeSmoke
	case constants.WeaponFlashbang:
		utilityType = UtilityTypeFlash
	case constants.WeaponHEGrenade:
		utilityType = UtilityTypeHE
	case constants.WeaponDecoy:
		utilityType = UtilityTypeDecoy
	default:
		return nil
	}

	throwType := classifyUtilityThrowType(analyzer, shot, thrower)

	return &Utility{
		Frame:            shot.Frame,
		Tick:             shot.Tick,
		RoundNumber:      shot.RoundNumber,
		UtilityType:      utilityType,
		ThrowType:        throwType,
		GrenadeID:        shot.WeaponID,
		ProjectileID:     shot.ProjectileID,
		X:                shot.X,
		Y:                shot.Y,
		Z:                shot.Z,
		ThrowerSteamID64: shot.PlayerSteamID64,
		ThrowerName:      shot.PlayerName,
		ThrowerSide:      shot.PlayerSide,
		ThrowerTeamName:  shot.PlayerTeamName,
		ThrowerVelocityX: shot.PlayerVelocityX,
		ThrowerVelocityY: shot.PlayerVelocityY,
		ThrowerVelocityZ: shot.PlayerVelocityZ,
		ThrowerPitch:     shot.Pitch,
		ThrowerYaw:       shot.Yaw,
	}
}

type throwButtonState struct {
	hasAttack           bool
	hasAttack2          bool
	hasAttackAndAttack2 bool
	hasJump             bool
	hasForward          bool
}

func classifyUtilityThrowType(analyzer *Analyzer, shot *Shot, thrower *common.Player) UtilityThrowType {
	if analyzer == nil || analyzer.match == nil || shot == nil {
		return UtilityThrowTypeLeftClick
	}

	speed2D := math.Sqrt(shot.PlayerVelocityX*shot.PlayerVelocityX + shot.PlayerVelocityY*shot.PlayerVelocityY)
	isAirborne := thrower != nil && thrower.IsAirborne()

	baseType := UtilityThrowTypeLeftClick
	if isAirborne {
		if speed2D > throwStandingSpeedMax {
			baseType = UtilityThrowTypeRunJumpThrow
		} else {
			baseType = UtilityThrowTypeJumpThrow
		}
	} else if speed2D > throwStandingSpeedMax {
		baseType = UtilityThrowTypeRunThrow
	}

	startTick := getThrowButtonWindowStartTick(analyzer, shot.Tick)
	buttonState := getThrowButtonState(analyzer.match, shot.PlayerSteamID64, startTick, shot.Tick)

	if buttonState.hasJump {
		if buttonState.hasForward {
			if speed2D > throwStandingSpeedMax {
				return UtilityThrowTypeRunJumpThrow
			}
			return UtilityThrowTypeWJumpThrow
		}
		if speed2D > throwStandingSpeedMax {
			return UtilityThrowTypeRunJumpThrow
		}
		return UtilityThrowTypeJumpThrow
	}

	if baseType == UtilityThrowTypeRunJumpThrow || baseType == UtilityThrowTypeJumpThrow {
		return baseType
	}

	if baseType == UtilityThrowTypeRunThrow {
		return UtilityThrowTypeRunThrow
	}

	if buttonState.hasAttackAndAttack2 {
		return UtilityThrowTypeDoubleClick
	}
	if buttonState.hasAttack {
		return UtilityThrowTypeLeftClick
	}

	return UtilityThrowTypeLeftClick
}

func getThrowButtonWindowStartTick(analyzer *Analyzer, endTick int) int {
	tickRate := analyzer.parser.TickRate()
	if tickRate <= 0 {
		tickTime := analyzer.parser.TickTime().Seconds()
		if tickTime > 0 {
			tickRate = 1 / tickTime
		}
	}
	windowTicks := int(tickRate)
	if windowTicks < 1 {
		windowTicks = 1
	}
	startTick := endTick - windowTicks
	if startTick < 0 {
		return 0
	}
	return startTick
}

func getThrowButtonState(match *Match, steamID64 uint64, startTick int, endTick int) throwButtonState {
	state := throwButtonState{}
	var lastState uint64
	lastStateTick := -1
	windowInitialized := false
	for _, buttons := range match.PlayerButtons {
		if buttons.SteamID64 != steamID64 {
			continue
		}
		if buttons.Tick < startTick {
			lastState = buttons.Buttons
			lastStateTick = buttons.Tick
			continue
		}
		if buttons.Tick > endTick {
			continue
		}
		if !windowInitialized {
			windowInitialized = true
			if lastStateTick >= 0 {
				updateThrowButtonStateFromMask(&state, lastState)
			}
		}
		mask := buttons.Buttons
		updateThrowButtonStateFromMask(&state, mask)
	}
	if !windowInitialized && lastStateTick >= 0 {
		updateThrowButtonStateFromMask(&state, lastState)
	}

	return state
}

func updateThrowButtonStateFromMask(state *throwButtonState, mask uint64) {
	hasAttack := mask&uint64(common.ButtonAttack) != 0
	hasAttack2 := mask&uint64(common.ButtonAttack2) != 0
	state.hasAttack = state.hasAttack || hasAttack
	state.hasAttack2 = state.hasAttack2 || hasAttack2
	if hasAttack && hasAttack2 {
		state.hasAttackAndAttack2 = true
	}
	if mask&uint64(common.ButtonJump) != 0 {
		state.hasJump = true
	}
	if mask&uint64(common.ButtonForward) != 0 {
		state.hasForward = true
	}
}
