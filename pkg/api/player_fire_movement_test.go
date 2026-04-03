package api

import (
	"testing"
)

func TestPlayerCounterStrafingSuccessRate(t *testing.T) {
	player := &Player{SteamID64: 123}
	match := &Match{
		Shots: []*Shot{
			{PlayerSteamID64: player.SteamID64, IsPlayerRunning: false, RecoilIndex: 1},
			{PlayerSteamID64: player.SteamID64, IsPlayerRunning: true, RecoilIndex: 2},
			{PlayerSteamID64: player.SteamID64, IsPlayerRunning: true, RecoilIndex: 1},
			{PlayerSteamID64: player.SteamID64, IsPlayerRunning: false, RecoilIndex: 1},
			{PlayerSteamID64: player.SteamID64, IsPlayerRunning: false, IsPlayerControllingBot: true, RecoilIndex: 1},
			{PlayerSteamID64: 999, IsPlayerRunning: false, RecoilIndex: 1},
		},
	}
	player.match = match

	if got, want := player.CounterStrafingSuccessRate(), float32(66.66667); got != want {
		t.Fatalf("expected counter-strafing success rate to be %v but got %v", want, got)
	}

	if got := player.CounterStrafingSuccessRate(); got <= 0 {
		t.Fatalf("expected counter-strafing success rate to be positive but got %v", got)
	}
}

func TestPlayerCounterStrafingSuccessRate_NoFirstShots(t *testing.T) {
	player := &Player{
		SteamID64: 123,
		match: &Match{
			Shots: []*Shot{
				{PlayerSteamID64: 123, IsPlayerRunning: false, RecoilIndex: 2},
				{PlayerSteamID64: 123, IsPlayerRunning: true, RecoilIndex: 3},
			},
		},
	}

	if got := player.CounterStrafingSuccessRate(); got != 0 {
		t.Fatalf("expected counter-strafing success rate to be 0 but got %v", got)
	}
}

func TestPlayerCounterStrafingSuccessRate_NoShots(t *testing.T) {
	player := &Player{
		SteamID64: 123,
		match:     &Match{},
	}

	if got := player.CounterStrafingSuccessRate(); got != 0 {
		t.Fatalf("expected counter-strafing success rate to be 0 but got %v", got)
	}
}
