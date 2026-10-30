package api

import (
	"fmt"
	stdmath "math"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"

	"github.com/golang/geo/r3"
	common "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

type Shot struct {
	// Frame is retained internally for fork event correlation, not exported.
	Frame                  int                  `json:"-"`
	Tick                   int                  `json:"tick"`
	RoundNumber            int                  `json:"roundNumber"`
	WeaponName             constants.WeaponName `json:"weaponName"`
	WeaponType             constants.WeaponType `json:"weaponType"`
	WeaponID               string               `json:"weaponId"`
	ProjectileID           string               `json:"projectileId"` // Available only for grenades
	X                      float64              `json:"x"`
	Y                      float64              `json:"y"`
	Z                      float64              `json:"z"`
	PlayerName             string               `json:"playerName"`
	PlayerSteamID64        uint64               `json:"playerSteamId"`
	PlayerTeamName         string               `json:"playerTeamName"`
	PlayerSide             common.Team          `json:"playerSide"`
	IsPlayerControllingBot bool                 `json:"isPlayerControllingBot"`
	PlayerVelocityX        float64              `json:"playerVelocityX"`
	PlayerVelocityY        float64              `json:"playerVelocityY"`
	PlayerVelocityZ        float64              `json:"playerVelocityZ"`
	PlayerSpeed2D          float64              `json:"playerSpeed2D"`
	Yaw                    float32              `json:"yaw"`
	Pitch                  float32              `json:"pitch"`
	RecoilIndex            float32              `json:"recoilIndex"`
	AimPunchAngleX         float64              `json:"aimPunchAngleX"`
	AimPunchAngleY         float64              `json:"aimPunchAngleY"`
	HasAimPunchData        bool                 `json:"hasAimPunchData"`
	HasViewPunchData       bool                 `json:"hasViewPunchData"`
	HasRecoilIndexData     bool                 `json:"hasRecoilIndexData"`
	ViewPunchAngleX        float64              `json:"viewPunchAngleX"`
	ViewPunchAngleY        float64              `json:"viewPunchAngleY"`
	IsPlayerRunning        bool                 `json:"isPlayerRunning"`
	// AimPunchBase is the networked base state, not the instantaneous angle.
	// Its ticks use the server clock; Tick above uses the demo clock.
	AimPunchBase *AimPunchBase `json:"aimPunchBase,omitempty"`
}

type AimPunchBase struct {
	ServerTick               int       `json:"serverTick"`
	PredictableTick          int       `json:"predictableTick"`
	PredictableTickFraction  float32   `json:"predictableTickFraction"`
	PredictableAngle         r3.Vector `json:"predictableAngle"`
	PredictableAngleVelocity r3.Vector `json:"predictableAngleVelocity"`
	UnpredictableTick        int       `json:"unpredictableTick"`
	UnpredictableAngle       r3.Vector `json:"unpredictableAngle"`
}

func readAimPunchBase(pawn *st.Entity, serverTick int) *AimPunchBase {
	if pawn == nil {
		return nil
	}
	names := []string{"m_predictableBaseTick", "m_predictableBaseTickInterpAmount", "m_predictableBaseAngle", "m_predictableBaseAngleVel", "m_unpredictableBaseTick", "m_unpredictableBaseAngle"}
	values := make([]st.PropertyValue, len(names))
	for i, name := range names {
		value, exists := pawn.PropertyValue("m_pAimPunchServices." + name)
		if !exists || value.Any == nil {
			return nil
		}
		values[i] = value
	}
	return &AimPunchBase{ServerTick: serverTick, PredictableTick: int(values[0].UInt32()), PredictableTickFraction: values[1].Float(),
		PredictableAngle: values[2].R3Vec(), PredictableAngleVelocity: values[3].R3Vec(),
		UnpredictableTick: int(values[4].UInt32()), UnpredictableAngle: values[5].R3Vec()}
}

func newShot(analyzer *Analyzer, event events.WeaponFire) *Shot {
	shooter := event.Shooter
	if shooter == nil {
		return nil
	}

	var weaponEntity *st.Entity
	activeWeapon := shooter.ActiveWeapon()
	if activeWeapon != nil {
		weaponEntity = activeWeapon.Entity
	} else if event.Weapon.Entity != nil {
		weaponEntity = event.Weapon.Entity
	}

	var recoilIndex float32
	var hasRecoilIndexData bool
	if weaponEntity != nil {
		if prop, exists := weaponEntity.PropertyValue("m_flRecoilIndex"); exists && prop.Any != nil {
			recoilIndex = prop.Float()
			hasRecoilIndexData = true
		}
	}

	var aimPunchAngle r3.Vector
	var viewPunchAngle r3.Vector
	var hasAimPunchData, hasViewPunchData bool
	var aimPunchBase *AimPunchBase
	if analyzer.isSource2 {
		pawnEntity := shooter.PlayerPawnEntity()
		if pawnEntity != nil {
			aimPunchBase = readAimPunchBase(pawnEntity, analyzer.parser.GameState().IngameTick())
			// These properties can be absent in particular CS2 versions.
			if prop, exists := pawnEntity.PropertyValue("m_aimPunchAngle"); exists && prop.Any != nil {
				aimPunchAngle = prop.R3Vec()
				hasAimPunchData = true
			}
			if prop, exists := pawnEntity.PropertyValue("m_pCameraServices.m_vecCsViewPunchAngle"); exists && prop.Any != nil {
				viewPunchAngle = prop.R3Vec()
				hasViewPunchData = true
			}
		}
	} else {
		aimPunchAngle = shooter.Entity.PropertyValueMust("localdata.m_Local.m_aimPunchAngle").R3Vec()
		viewPunchAngle = shooter.Entity.PropertyValueMust("localdata.m_Local.m_viewPunchAngle").R3Vec()
		hasAimPunchData, hasViewPunchData = true, true
	}

	velocity := getPlayerVelocity(shooter, analyzer)

	yaw := shooter.ViewDirectionX()
	pitch := shooter.ViewDirectionY()

	var isPlayerRunning bool
	speed2D := stdmath.Sqrt(velocity.X*velocity.X + velocity.Y*velocity.Y)

	weaponName := equipmentToWeaponName[event.Weapon.Type]
	weaponType := getEquipmentWeaponType(*event.Weapon)
	if threshold, ok := constants.WeaponAccurateSpeed[weaponName]; ok {
		if speed2D > threshold {
			isPlayerRunning = true
		}
	} else {
		fmt.Printf("Error: Weapon '%s' not found in accurate speed constants\n", weaponName)
	}

	return &Shot{
		Frame:                  analyzer.parser.CurrentFrame(),
		Tick:                   analyzer.currentTick(),
		RoundNumber:            analyzer.currentRound.Number,
		WeaponName:             weaponName,
		WeaponType:             weaponType,
		WeaponID:               event.Weapon.UniqueID().String(),
		ProjectileID:           noProjectileID,
		X:                      shooter.Position().X,
		Y:                      shooter.Position().Y,
		Z:                      shooter.Position().Z,
		PlayerName:             shooter.Name,
		PlayerSteamID64:        shooter.SteamID64,
		PlayerTeamName:         analyzer.match.Team(shooter.Team).Name,
		PlayerSide:             shooter.Team,
		IsPlayerControllingBot: shooter.IsControllingBot(),
		PlayerVelocityX:        velocity.X,
		PlayerVelocityY:        velocity.Y,
		PlayerVelocityZ:        velocity.Z,
		PlayerSpeed2D:          speed2D,
		Yaw:                    yaw,
		Pitch:                  pitch,
		RecoilIndex:            recoilIndex,
		AimPunchAngleX:         aimPunchAngle.X,
		AimPunchAngleY:         aimPunchAngle.Y,
		HasAimPunchData:        hasAimPunchData,
		HasViewPunchData:       hasViewPunchData,
		HasRecoilIndexData:     hasRecoilIndexData,
		AimPunchBase:           aimPunchBase,
		ViewPunchAngleX:        viewPunchAngle.X,
		ViewPunchAngleY:        viewPunchAngle.Y,
		IsPlayerRunning:        isPlayerRunning,
	}
}
