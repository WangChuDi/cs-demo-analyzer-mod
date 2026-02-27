package api

import (
	"math"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type UtilityType string

const (
	UtilityTypeSmoke      UtilityType = "smoke"
	UtilityTypeFlash      UtilityType = "flash"
	UtilityTypeHE         UtilityType = "he"
	UtilityTypeDecoy      UtilityType = "decoy"
	UtilityTypeMolotov    UtilityType = "molotov"
	UtilityTypeIncendiary UtilityType = "incendiary"
)

type UtilityThrowType string

const (
	UtilityThrowTypeLeftClick   UtilityThrowType = "left_click"
	UtilityThrowTypeRightClick  UtilityThrowType = "right_click"
	UtilityThrowTypeDoubleClick UtilityThrowType = "double_click"
)

type Utility struct {
	Frame            int              `json:"frame"`
	Tick             int              `json:"tick"`
	RoundNumber      int              `json:"roundNumber"`
	UtilityType      UtilityType      `json:"utilityType"`
	HasAttack     bool             `json:"hasAttack"`
	HasAttack2    bool             `json:"hasAttack2"`
	HasJump       bool             `json:"hasJump"`
	HasForward    bool             `json:"hasForward"`
	HasBack       bool             `json:"hasBack"`
	HasMoveLeft   bool             `json:"hasMoveLeft"`
	HasMoveRight  bool             `json:"hasMoveRight"`
	HasWalk       bool             `json:"hasWalk"`
	PinPulledTick int              `json:"pinPulledTick"`
	MouseTypeByStrength UtilityThrowType `json:"mouseTypeByStrength"`
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
	ThrowerSpeed2D   float64          `json:"throwerSpeed2d"`
	ThrowerSpeedType string           `json:"throwerSpeedType"`
	ThrowerPitch     float32          `json:"throwerPitch"`
	ThrowerYaw       float32          `json:"throwerYaw"`
	IsJumpThrow      bool             `json:"isJumpThrow"`
	ThrowStrength    float64          `json:"throwStrength"`
	InitialVelocityX float64          `json:"initialVelocityX"`
	InitialVelocityY float64          `json:"initialVelocityY"`
	InitialVelocityZ float64          `json:"initialVelocityZ"`
	InitialSpeed     float64          `json:"initialSpeed"`
	InitialPositionX float64          `json:"initialPositionX"`
	InitialPositionY float64          `json:"initialPositionY"`
	InitialPositionZ float64          `json:"initialPositionZ"`
}

func newUtilityFromShot(analyzer *Analyzer, shot *Shot, thrower *common.Player, weaponEntity st.Entity) *Utility {
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
	case constants.WeaponMolotov:
		utilityType = UtilityTypeMolotov
	case constants.WeaponIncendiary:
		utilityType = UtilityTypeIncendiary
	default:
		return nil
	}

	utility := &Utility{
		Frame:            shot.Frame,
		Tick:             shot.Tick,
		RoundNumber:      shot.RoundNumber,
		UtilityType:      utilityType,
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
		ThrowerSpeed2D:   getSpeed2D(shot.PlayerVelocityX, shot.PlayerVelocityY),
		ThrowerSpeedType: classifyThrowerSpeedType(getSpeed2D(shot.PlayerVelocityX, shot.PlayerVelocityY)),
		ThrowerPitch:     shot.Pitch,
		ThrowerYaw:       shot.Yaw,
	}

	utility.PinPulledTick = getPinPulledTick(analyzer, weaponEntity)
	applyUtilityThrowButtons(analyzer, utility)
	return utility
}

func getUtilityIsJumpThrow(projectile *common.GrenadeProjectile) bool {
	if projectile == nil || projectile.WeaponInstance == nil || projectile.WeaponInstance.Entity == nil {
		return false
	}
	if val, ok := projectile.WeaponInstance.Entity.PropertyValue("m_bJumpThrow"); ok {
		return val.BoolVal()
	}
	return false
}

func getUtilityThrowStrength(projectile *common.GrenadeProjectile) float64 {
	if projectile == nil || projectile.WeaponInstance == nil || projectile.WeaponInstance.Entity == nil {
		return 0
	}
	if val, ok := projectile.WeaponInstance.Entity.PropertyValue("m_flThrowStrength"); ok {
		return float64(val.Float())
	}
	return 0
}

func getPinPulledTick(analyzer *Analyzer, weaponEntity st.Entity) int {
	if weaponEntity == nil {
		return 0
	}
	val, ok := weaponEntity.PropertyValue("m_fPinPullTime")
	if !ok {
		return 0
	}
	pinPullTime := float64(val.Float())
	if pinPullTime <= 0 {
		return 0
	}

	tickTime := analyzer.parser.TickTime().Seconds()
	if tickTime <= 0 {
		return 0
	}

	currentTick := analyzer.currentTick()
	currentTime := analyzer.parser.CurrentTime().Seconds()
	elapsed := currentTime - pinPullTime
	ticksAgo := int(elapsed / tickTime)
	pinTick := currentTick - ticksAgo
	if pinTick < 0 {
		return 0
	}
	return pinTick
}
func classifyThrowTypeByStrength(utility *Utility) UtilityThrowType {
	if utility.ThrowStrength == 1.0 {
		return UtilityThrowTypeLeftClick
	}
	if utility.ThrowStrength == 0.5 {
		return UtilityThrowTypeDoubleClick
	}
	// ThrowStrength == 0: game bug, fallback to calculated_strength
	calcStrength := calcThrowStrength(utility)
	if calcStrength > 500 {
		return UtilityThrowTypeLeftClick
	}
	if calcStrength > 300 {
		return UtilityThrowTypeDoubleClick
	}
	return UtilityThrowTypeRightClick
}

func calcThrowStrength(utility *Utility) float64 {
	baseX := utility.InitialVelocityX - 1.25*utility.ThrowerVelocityX
	baseY := utility.InitialVelocityY - 1.25*utility.ThrowerVelocityY
	baseSpeedXY := math.Sqrt(baseX*baseX + baseY*baseY)
	pitchRad := float64(utility.ThrowerPitch) * math.Pi / 180.0
	cosPitch := math.Cos(pitchRad)
	if math.Abs(cosPitch) < 1e-6 {
		cosPitch = 1e-6
	}
	return baseSpeedXY / cosPitch
}
func classifyThrowerSpeedType(speed2D float64) string {
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

func getUtilityInitialVelocity(projectile *common.GrenadeProjectile) r3.Vector {
	if projectile == nil || projectile.Entity == nil {
		return r3.Vector{}
	}
	if val, ok := projectile.Entity.PropertyValue("m_vInitialVelocity"); ok {
		return val.R3Vec()
	}
	return r3.Vector{}
}

func getUtilityInitialPosition(projectile *common.GrenadeProjectile) r3.Vector {
	if projectile == nil || projectile.Entity == nil {
		return r3.Vector{}
	}
	if val, ok := projectile.Entity.PropertyValue("m_vInitialPosition"); ok {
		return val.R3Vec()
	}
	return r3.Vector{}
}

func getUtilityInitialSpeed(velocity r3.Vector) float64 {
	return math.Sqrt(velocity.X*velocity.X + velocity.Y*velocity.Y + velocity.Z*velocity.Z)
}

func getSpeed2D(x float64, y float64) float64 {
	return math.Sqrt(x*x + y*y)
}

type throwButtonState struct {
	hasAttack    bool
	hasAttack2   bool
	hasJump      bool
	hasForward   bool
	hasBack      bool
	hasMoveLeft  bool
	hasMoveRight bool
	hasWalk      bool
}

func applyUtilityThrowButtons(analyzer *Analyzer, utility *Utility) {
	if analyzer == nil || analyzer.match == nil || utility == nil {
		return
	}
	fallbackStart := getThrowButtonWindowStartTick(analyzer, utility.Tick)
	startTick := fallbackStart
	if utility.PinPulledTick > 0 && utility.PinPulledTick > startTick {
		startTick = utility.PinPulledTick
	}
	buttonState := getThrowButtonState(analyzer.match, utility.ThrowerSteamID64, startTick, utility.Tick)
	applyButtonStateToUtility(utility, buttonState)
}

func getThrowButtonWindowStartTick(analyzer *Analyzer, endTick int) int {
	tickRate := analyzer.parser.TickRate()
	if tickRate <= 0 {
		tickTime := analyzer.parser.TickTime().Seconds()
		if tickTime > 0 {
			tickRate = 1 / tickTime
		}
	}
	windowTicks := int(tickRate / 2)
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
	if mask&uint64(common.ButtonAttack) != 0 {
		state.hasAttack = true
	}
	if mask&uint64(common.ButtonAttack2) != 0 {
		state.hasAttack2 = true
	}
	if mask&uint64(common.ButtonJump) != 0 {
		state.hasJump = true
	}
	if mask&uint64(common.ButtonForward) != 0 {
		state.hasForward = true
	}
	if mask&uint64(common.ButtonBack) != 0 {
		state.hasBack = true
	}
	if mask&uint64(common.ButtonMoveLeft) != 0 {
		state.hasMoveLeft = true
	}
	if mask&uint64(common.ButtonMoveRight) != 0 {
		state.hasMoveRight = true
	}
	if mask&uint64(common.ButtonSpeed) != 0 {
		state.hasWalk = true
	}
}

func applyButtonStateToUtility(utility *Utility, state throwButtonState) {
	utility.HasAttack = state.hasAttack
	utility.HasAttack2 = state.hasAttack2
	utility.HasJump = state.hasJump
	utility.HasForward = state.hasForward
	utility.HasBack = state.hasBack
	utility.HasMoveLeft = state.hasMoveLeft
	utility.HasMoveRight = state.hasMoveRight
	utility.HasWalk = state.hasWalk
}
