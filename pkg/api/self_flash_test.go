package api

import (
	"math"
	"testing"
)

func TestSelfFlashStatsOnlyCountsValidSelfEvents(t *testing.T) {
	match := &Match{PlayersFlashed: []*PlayerFlashed{
		{Tick: 100, FlasherSteamID64: 1, FlashedSteamID64: 1, Duration: 2},
		{Tick: 101, FlasherSteamID64: 1, FlashedSteamID64: 1, Duration: 3},
		{FlasherSteamID64: 1, FlashedSteamID64: 2, Duration: 5},
		{FlasherSteamID64: 2, FlashedSteamID64: 1, Duration: 5},
		{FlasherSteamID64: 1, FlashedSteamID64: 1, Duration: 5, IsFlashedControllingBot: true},
		{FlasherSteamID64: 1, FlashedSteamID64: 1, Duration: 5, IsFlasherControllingBot: true},
		{FlasherSteamID64: 1, FlashedSteamID64: 1, Duration: 0},
		{FlasherSteamID64: 1, FlashedSteamID64: 1, Duration: -1},
		{FlasherSteamID64: 1, FlashedSteamID64: 1, Duration: float32(math.NaN())},
		{FlasherSteamID64: 1, FlashedSteamID64: 1, Duration: float32(math.Inf(1))},
		{FlasherSteamID64: 0, FlashedSteamID64: 0, Duration: 5}, nil,
	}}
	stats := (&Player{SteamID64: 1, match: match}).SelfFlashStats()
	// Overlapping event durations are explicitly summed, not called a union of
	// blindness intervals. No arbitrary minimum such as one second is imposed.
	if stats.SelfFlashCount != 2 || stats.SelfFlashDuration != 5 {
		t.Fatalf("wrong self-event sum: %+v", stats)
	}
	if stats := (&Player{match: match}).SelfFlashStats(); stats.SelfFlashCount != 0 {
		t.Fatal("unknown identity counted")
	}
}
