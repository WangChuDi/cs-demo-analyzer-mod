package api

import (
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

type Utility struct {
	Frame            int         `json:"frame"`
	Tick             int         `json:"tick"`
	RoundNumber      int         `json:"roundNumber"`
	UtilityType      UtilityType `json:"utilityType"`
	GrenadeID        string      `json:"grenadeId"`
	ProjectileID     int64       `json:"projectileId"`
	X                float64     `json:"x"`
	Y                float64     `json:"y"`
	Z                float64     `json:"z"`
	ThrowerSteamID64 uint64      `json:"throwerSteamId"`
	ThrowerName      string      `json:"throwerName"`
	ThrowerSide      common.Team `json:"throwerSide"`
	ThrowerTeamName  string      `json:"throwerTeamName"`
	ThrowerVelocityX float64     `json:"throwerVelocityX"`
	ThrowerVelocityY float64     `json:"throwerVelocityY"`
	ThrowerVelocityZ float64     `json:"throwerVelocityZ"`
	ThrowerPitch     float32     `json:"throwerPitch"`
	ThrowerYaw       float32     `json:"throwerYaw"`
}

func newUtilityFromShot(shot *Shot) *Utility {
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

	return &Utility{
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
		ThrowerPitch:     shot.Pitch,
		ThrowerYaw:       shot.Yaw,
	}
}
