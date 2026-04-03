package api

import (
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func TestGenerateAwpHoldDeaths_FiltersVictimState(t *testing.T) {
	match := &Match{
		TickRate:  64,
		FrameRate: 64,
		Kills: []*Kill{
			{
				RoundNumber:            1,
				Frame:                  100,
				Tick:                   100,
				KillerName:             "killer-1",
				KillerSteamID64:        10,
				KillerSide:             common.TeamTerrorists,
				KillerTeamName:         "T",
				VictimName:             "victim-1",
				VictimSteamID64:        20,
				VictimSide:             common.TeamCounterTerrorists,
				VictimTeamName:         "CT",
				WeaponName:             constants.WeaponAK47,
				KillerX:                10,
				KillerY:                0,
				KillerZ:                0,
				KillerVelocityX:        25,
				VictimX:                0,
				VictimY:                0,
				VictimZ:                0,
				VictimYaw:              0,
				VictimActiveWeaponName: constants.WeaponAWP,
				VictimIsScoped:         true,
			},
			{
				RoundNumber:            1,
				Frame:                  200,
				Tick:                   200,
				KillerName:             "killer-2",
				KillerSteamID64:        11,
				KillerSide:             common.TeamTerrorists,
				KillerTeamName:         "T",
				VictimName:             "victim-2",
				VictimSteamID64:        21,
				VictimSide:             common.TeamCounterTerrorists,
				VictimTeamName:         "CT",
				WeaponName:             constants.WeaponAK47,
				KillerX:                10,
				KillerY:                0,
				KillerZ:                0,
				KillerVelocityX:        10,
				VictimX:                0,
				VictimY:                0,
				VictimZ:                0,
				VictimYaw:              0,
				VictimActiveWeaponName: constants.WeaponAWP,
				VictimIsScoped:         false,
			},
		},
	}

	generateAwpHoldDeaths(match)

	if got, want := len(match.AwpHoldDeaths), 1; got != want {
		t.Fatalf("expected %d awp hold death, got %d", want, got)
	}

	row := match.AwpHoldDeaths[0]
	if row.VictimSteamID64 != 20 {
		t.Fatalf("expected qualified victim steamID 20, got %d", row.VictimSteamID64)
	}
	if !row.IsVictimScoped || !row.IsVictimFacingKiller || !row.IsVictimSlow {
		t.Fatalf("expected victim to satisfy scoped/facing/slow conditions, got scoped=%v facing=%v slow=%v", row.IsVictimScoped, row.IsVictimFacingKiller, row.IsVictimSlow)
	}
}

func TestGenerateAwpHoldDeaths_UsesNegativeOffsetsForRecentPreDeathShot(t *testing.T) {
	match := qualifiedAwpHoldMatch()
	match.Shots = []*Shot{
		{
			RoundNumber:     1,
			Frame:           98,
			Tick:            98,
			WeaponName:      constants.WeaponAWP,
			PlayerSteamID64: 20,
		},
	}

	generateAwpHoldDeaths(match)

	if got, want := len(match.AwpHoldDeaths), 1; got != want {
		t.Fatalf("expected %d awp hold death, got %d", want, got)
	}

	row := match.AwpHoldDeaths[0]
	if row.ShotOffsetFrame != -2 || row.ShotOffsetTick != -2 {
		t.Fatalf("expected negative pre-death offsets (-2, -2), got (%d, %d)", row.ShotOffsetFrame, row.ShotOffsetTick)
	}
	if !row.HasVictimAwpShotAroundDeath {
		t.Fatalf("expected victim pre-death awp shot to be detected")
	}
}

func TestGenerateAwpHoldDeaths_UsesPositiveOffsetsWhenVictimDoesNotShoot(t *testing.T) {
	match := qualifiedAwpHoldMatch()

	generateAwpHoldDeaths(match)

	if got, want := len(match.AwpHoldDeaths), 1; got != want {
		t.Fatalf("expected %d awp hold death, got %d", want, got)
	}

	row := match.AwpHoldDeaths[0]
	if row.ShotOffsetFrame <= 0 || row.ShotOffsetTick <= 0 {
		t.Fatalf("expected positive offsets when victim does not shoot, got (%d, %d)", row.ShotOffsetFrame, row.ShotOffsetTick)
	}
	if row.HasVictimAwpShotAroundDeath {
		t.Fatalf("expected no victim awp shot around death")
	}
}

func TestGenerateAwpHoldDeaths_UsesNegativeTickSentinelForSameTickPreDeathShot(t *testing.T) {
	match := qualifiedAwpHoldMatch()
	match.Shots = []*Shot{
		{
			RoundNumber:     1,
			Frame:           100,
			Tick:            100,
			WeaponName:      constants.WeaponAWP,
			PlayerSteamID64: 20,
		},
	}

	generateAwpHoldDeaths(match)

	if got, want := len(match.AwpHoldDeaths), 1; got != want {
		t.Fatalf("expected %d awp hold death, got %d", want, got)
	}

	row := match.AwpHoldDeaths[0]
	if row.ShotOffsetFrame != 0 || row.ShotOffsetTick != -1 {
		t.Fatalf("expected same-tick pre-death shot to use (0, -1) sentinel, got (%d, %d)", row.ShotOffsetFrame, row.ShotOffsetTick)
	}
	if row.ShotOffsetMs != 0 {
		t.Fatalf("expected same-tick pre-death shot to use 0ms offset, got %f", row.ShotOffsetMs)
	}
	if !row.HasVictimAwpShotAroundDeath {
		t.Fatalf("expected same-tick victim awp shot to be detected")
	}
}

func TestGenerateAwpHoldDeaths_KeepsKillVelocityWhenSnapshotVelocityIsUnavailable(t *testing.T) {
	match := qualifiedAwpHoldMatch()
	match.Kills[0].VictimVelocityX = 90
	match.PlayerPositions = []*PlayerPosition{
		{
			RoundNumber:      1,
			Tick:             100,
			Frame:            100,
			SteamID64:        20,
			X:                0,
			Y:                0,
			Z:                0,
			Yaw:              0,
			ActiveWeaponName: constants.WeaponAWP,
			IsScoping:        true,
		},
	}

	generateAwpHoldDeaths(match)

	if got := len(match.AwpHoldDeaths); got != 0 {
		t.Fatalf("expected no awp hold death when kill snapshot says victim is moving too fast, got %d", got)
	}
}

func qualifiedAwpHoldMatch() *Match {
	return &Match{
		TickRate:  64,
		FrameRate: 64,
		Kills: []*Kill{
			{
				RoundNumber:            1,
				Frame:                  100,
				Tick:                   100,
				KillerName:             "killer",
				KillerSteamID64:        10,
				KillerSide:             common.TeamTerrorists,
				KillerTeamName:         "T",
				VictimName:             "victim",
				VictimSteamID64:        20,
				VictimSide:             common.TeamCounterTerrorists,
				VictimTeamName:         "CT",
				WeaponName:             constants.WeaponAK47,
				KillerX:                10,
				KillerY:                0,
				KillerZ:                0,
				KillerVelocityX:        30,
				VictimX:                0,
				VictimY:                0,
				VictimZ:                0,
				VictimYaw:              0,
				VictimActiveWeaponName: constants.WeaponAWP,
				VictimIsScoped:         true,
			},
		},
	}
}

func TestGenerateAwpHoldDeaths_DoesNotRequirePlayerPositions(t *testing.T) {
	match := qualifiedAwpHoldMatch()
	match.PlayerPositions = nil

	generateAwpHoldDeaths(match)

	if got, want := len(match.AwpHoldDeaths), 1; got != want {
		t.Fatalf("expected %d awp hold death without player positions, got %d", want, got)
	}

	row := match.AwpHoldDeaths[0]
	if row.KillerSpeed2D <= 0 {
		t.Fatalf("expected killer speed to be recorded, got %f", row.KillerSpeed2D)
	}
	if row.VictimWeaponName != constants.WeaponAWP {
		t.Fatalf("expected victim weapon to be awp, got %s", row.VictimWeaponName)
	}
}

func TestPlayerAwpHoldCounts(t *testing.T) {
	match := &Match{
		AwpHoldDeaths: []*AwpHoldDeath{
			{KillerSteamID64: 10, VictimSteamID64: 20},
			{KillerSteamID64: 10, VictimSteamID64: 21},
			{KillerSteamID64: 11, VictimSteamID64: 10},
		},
	}

	killer := &Player{SteamID64: 10, match: match}
	victim := &Player{SteamID64: 20, match: match}

	if got, want := killer.AwpHoldKillCount(), 2; got != want {
		t.Fatalf("expected awp hold kill count %d, got %d", want, got)
	}
	if got, want := killer.AwpHoldDeathCount(), 1; got != want {
		t.Fatalf("expected awp hold death count %d, got %d", want, got)
	}
	if got, want := victim.AwpHoldKillCount(), 0; got != want {
		t.Fatalf("expected victim awp hold kill count %d, got %d", want, got)
	}
	if got, want := victim.AwpHoldDeathCount(), 1; got != want {
		t.Fatalf("expected victim awp hold death count %d, got %d", want, got)
	}
}
