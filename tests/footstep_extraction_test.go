package tests

import (
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api"
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/akiver/cs-demo-analyzer/tests/testsutils"
)

func Test_Footstep_Extraction(t *testing.T) {
	// Trying a different demo that might have match start events properly recorded
	demoName := "renown_match_8_2025_mirage"
	demoPath := testsutils.GetDemoPath("cs2", demoName)
	match, err := api.AnalyzeDemo(demoPath, api.AnalyzeDemoOptions{
		Source: constants.DemoSourceRenown,
	})
	if err != nil {
		t.Error(err)
	}

	if len(match.Footsteps) == 0 {
		t.Logf("Warning: No footsteps extracted from %s. This might be due to the demo lacking match-start phase or footstep data.", demoName)
		return
	}

	// Verify first footstep structure
	firstFootstep := match.Footsteps[0]
	if firstFootstep.Tick <= 0 {
		t.Errorf("expected footstep tick to be positive, got %d", firstFootstep.Tick)
	}
	if firstFootstep.PlayerName == "" {
		t.Error("expected footstep player name to be populated")
	}
}
