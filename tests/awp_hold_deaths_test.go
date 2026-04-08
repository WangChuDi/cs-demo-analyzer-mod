package tests

import (
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api"
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/akiver/cs-demo-analyzer/tests/testsutils"
)

func TestAwpHoldDeaths_5EPlayMirage(t *testing.T) {
	demoName := "5eplay_g161_20231231135244670959707_2023_mirage"
	demoPath := testsutils.GetDemoPath("cs2", demoName)

	match, err := api.AnalyzeDemo(demoPath, api.AnalyzeDemoOptions{
		Source: constants.DemoSourceFiveEPlay,
	})
	if err != nil {
		t.Fatalf("analyze demo: %v", err)
	}

	if len(match.AwpHoldDeaths) == 0 {
		t.Fatalf("expected at least one awp hold death in %s", demoName)
	}

	killsByPlayer := make(map[uint64]int)
	deathsByPlayer := make(map[uint64]int)
	for index, event := range match.AwpHoldDeaths {
		if event == nil {
			t.Fatalf("expected awp hold death %d to be non-nil", index)
		}
		if !event.IsVictimScoped {
			t.Fatalf("expected awp hold death %d victim to be scoped", index)
		}
		if !event.IsVictimFacingKiller {
			t.Fatalf("expected awp hold death %d victim to be facing killer", index)
		}
		if !event.IsVictimSlow {
			t.Fatalf("expected awp hold death %d victim to be slow", index)
		}
		if event.ShotOffsetFrame < 0 && !event.HasPreDeathVictimAwpShot {
			t.Fatalf("expected negative shot offset frame to imply a nearby pre-death victim awp shot")
		}
		if event.HasPreDeathVictimAwpShot && event.VictimReactionShotFrame == 0 {
			t.Fatalf("expected pre-death victim awp shot flag to carry a reaction frame")
		}
		if event.HasPostDeathVictimAttackTrigger && event.HasPreDeathVictimAwpShot {
			// allowed, but the event should still be internally consistent with a valid reaction frame
			if event.VictimReactionShotFrame == 0 {
				t.Fatalf("expected combined reaction event to keep the pre-death reaction frame")
			}
		}

		killsByPlayer[event.KillerSteamID64]++
		deathsByPlayer[event.VictimSteamID64]++
	}

	totalKillCount := 0
	totalDeathCount := 0
	for steamID, player := range match.PlayersBySteamID {
		if player == nil {
			continue
		}

		playerKillCount := player.AwpHoldKillCount()
		playerDeathCount := player.AwpHoldDeathCount()
		if playerKillCount != killsByPlayer[steamID] {
			t.Fatalf("expected awp hold kill count %d for player %d, got %d", killsByPlayer[steamID], steamID, playerKillCount)
		}
		if playerDeathCount != deathsByPlayer[steamID] {
			t.Fatalf("expected awp hold death count %d for player %d, got %d", deathsByPlayer[steamID], steamID, playerDeathCount)
		}

		totalKillCount += playerKillCount
		totalDeathCount += playerDeathCount
	}

	if totalKillCount != len(match.AwpHoldDeaths) {
		t.Fatalf("expected summed player awp hold kills %d to equal event count %d", totalKillCount, len(match.AwpHoldDeaths))
	}
	if totalDeathCount != len(match.AwpHoldDeaths) {
		t.Fatalf("expected summed player awp hold deaths %d to equal event count %d", totalDeathCount, len(match.AwpHoldDeaths))
	}
}
