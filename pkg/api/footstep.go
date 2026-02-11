package api

import (
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
)

type Footstep struct {
	Frame                  int         `json:"frame"`
	Tick                   int         `json:"tick"`
	RoundNumber            int         `json:"roundNumber"`
	X                      float64     `json:"x"`
	Y                      float64     `json:"y"`
	Z                      float64     `json:"z"`
	PlayerName             string      `json:"playerName"`
	PlayerSteamID64        uint64      `json:"playerSteamId"`
	PlayerTeamName         string      `json:"playerTeamName"`
	PlayerSide             common.Team `json:"playerSide"`
	IsPlayerControllingBot bool        `json:"isPlayerControllingBot"`
	PlayerVelocityX        float64     `json:"playerVelocityX"`
	PlayerVelocityY        float64     `json:"playerVelocityY"`
	PlayerVelocityZ        float64     `json:"playerVelocityZ"`
	Yaw                    float32     `json:"yaw"`
	Pitch                  float32     `json:"pitch"`
}

func newFootstep(analyzer *Analyzer, event events.Footstep, velocityX, velocityY, velocityZ float64) *Footstep {
	player := event.Player
	if player == nil {
		return nil
	}

	playerPos := player.Position()

	return &Footstep{
		Frame:                  analyzer.parser.CurrentFrame(),
		Tick:                   analyzer.currentTick(),
		RoundNumber:            analyzer.currentRound.Number,
		X:                      playerPos.X,
		Y:                      playerPos.Y,
		Z:                      playerPos.Z,
		PlayerName:             player.Name,
		PlayerSteamID64:        player.SteamID64,
		PlayerTeamName:         analyzer.match.Team(player.Team).Name,
		PlayerSide:             player.Team,
		IsPlayerControllingBot: player.IsControllingBot(),
		PlayerVelocityX:        velocityX,
		PlayerVelocityY:        velocityY,
		PlayerVelocityZ:        velocityZ,
		Yaw:                    player.ViewDirectionX(),
		Pitch:                  player.ViewDirectionY(),
	}
}
