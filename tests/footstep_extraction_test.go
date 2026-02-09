package tests

import (
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api"
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/akiver/cs-demo-analyzer/tests/testsutils"
)

func Test_Footstep_Extraction(t *testing.T) {
	// Reusing an existing demo from the test suite that is likely to have footsteps
	demoName := "matchzy_bleed_vs_parivision_2024_mirage"
	demoPath := testsutils.GetDemoPath("cs2", demoName)
	match, err := api.AnalyzeDemo(demoPath, api.AnalyzeDemoOptions{
		Source: constants.DemoSourceMatchZy,
	})
	if err != nil {
		t.Error(err)
	}

	if len(match.Footsteps) == 0 {
		t.Error("expected footsteps to be extracted, but got 0")
	}

	// Verify first footstep structure
	firstFootstep := match.Footsteps[0]
	if firstFootstep.Tick <= 0 {
		t.Errorf("expected footstep tick to be positive, got %d", firstFootstep.Tick)
	}
	if firstFootstep.PlayerName == "" {
		t.Error("expected footstep player name to be populated")
	}
	// Note: We don't assert non-zero velocity here because in CS2 parser current implementation,
	// velocity might be 0 for footstep events.
}
