package api

import (
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
)

type ChickenDeath struct {
	Frame           int                  `json:"frame"`
	Tick            int                  `json:"tick"`
	RoundNumber     int                  `json:"roundNumber"`
	KillerSteamID   uint64               `json:"killerSteamId"`
	WeaponName      constants.WeaponName `json:"weaponName"`
	LeaderSteamID   uint64               `json:"leaderSteamId"`
	LeaderName      string               `json:"leaderName"`
}

func newChickenDeath(frame int, tick int, roundNumber int, killerSteamID uint64, weaponName constants.WeaponName, leaderSteamID uint64, leaderName string) *ChickenDeath {
	chickenDeath := &ChickenDeath{
		Frame:           frame,
		Tick:            tick,
		RoundNumber:     roundNumber,
		WeaponName:      weaponName,
		KillerSteamID:   killerSteamID,
		LeaderSteamID:   leaderSteamID,
		LeaderName:      leaderName,
	}

	return chickenDeath
}
