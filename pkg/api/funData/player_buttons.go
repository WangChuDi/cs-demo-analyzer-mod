package funData

import (
	"strings"

	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v4/pkg/demoinfocs/events"
)

type PlayerButtons struct {
	Frame       int
	Tick        int
	RoundNumber int
	SteamID64   uint64
	Name        string
	Buttons     uint64
	ButtonNames string
}

type buttonAction struct {
	mask common.ButtonBitMask
	name string
}

var buttons = []buttonAction{
	{common.ButtonAttack, "Attack"},
	{common.ButtonJump, "Jump"},
	{common.ButtonDuck, "Duck"},
	{common.ButtonForward, "Forward"},
	{common.ButtonBack, "Back"},
	{common.ButtonUse, "Use"},
	{common.ButtonTurnLeft, "TurnLeft"},
	{common.ButtonTurnRight, "TurnRight"},
	{common.ButtonMoveLeft, "MoveLeft"},
	{common.ButtonMoveRight, "MoveRight"},
	{common.ButtonAttack2, "Attack2"},
	{common.ButtonReload, "Reload"},
	{common.ButtonSpeed, "Speed"},
	{common.ButtonJoyAutoSprint, "JoyAutoSprint"},
	{common.ButtonUseOrReload, "UseOrReload"},
	{common.ButtonScore, "Score"},
	{common.ButtonZoom, "Zoom"},
	{common.ButtonLookAtWeapon, "LookAtWeapon"},
}

func NewPlayerButtons(frame int, tick int, roundNumber int, event events.PlayerButtonsStateUpdate) *PlayerButtons {
	actions := []string{}
	for _, btn := range buttons {
		if event.ButtonsState&uint64(btn.mask) != 0 {
			actions = append(actions, btn.name)
		}
	}

	return &PlayerButtons{
		Frame:       frame,
		Tick:        tick,
		RoundNumber: roundNumber,
		SteamID64:   event.Player.SteamID64,
		Name:        event.Player.Name,
		Buttons:     event.ButtonsState,
		ButtonNames: strings.Join(actions, ","),
	}
}
