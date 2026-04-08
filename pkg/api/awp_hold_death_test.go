package api

import (
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/akiver/cs-demo-analyzer/pkg/api/funData"
	"github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
)

func TestNearestShotOffset_UsesOnlyPreDeathShotsWithinWindow(t *testing.T) {
	shots := []*Shot{
		{Frame: 95, Tick: 95, WeaponName: constants.WeaponAWP},
		{Frame: 101, Tick: 101, WeaponName: constants.WeaponAWP},
	}

	shot, offsetFrame, offsetTick := nearestShotOffset(100, 100, shots, 16, 16)

	if shot == nil {
		t.Fatalf("expected pre-death shot match")
	}
	if shot.Frame != 95 || shot.Tick != 95 {
		t.Fatalf("expected nearest pre-death shot at 95/95, got %d/%d", shot.Frame, shot.Tick)
	}
	if offsetFrame != -5 || offsetTick != -5 {
		t.Fatalf("expected offsets -5/-5, got %d/%d", offsetFrame, offsetTick)
	}
}

func TestNearestShotOffset_IgnoresPostDeathShots(t *testing.T) {
	shots := []*Shot{
		{Frame: 101, Tick: 101, WeaponName: constants.WeaponAWP},
	}

	shot, offsetFrame, offsetTick := nearestShotOffset(100, 100, shots, 16, 16)

	if shot != nil || offsetFrame != 0 || offsetTick != 0 {
		t.Fatalf("expected post-death shot to be ignored, got shot=%v offsets=%d/%d", shot != nil, offsetFrame, offsetTick)
	}
}

func TestHasPostDeathAttackWithinWindow_DetectsAttackEdge(t *testing.T) {
	buttons := []*funData.PlayerButtons{
		{Frame: 99, Tick: 99, Buttons: 0},
		{Frame: 101, Tick: 101, Buttons: uint64(common.ButtonAttack)},
	}

	if !hasPostDeathAttackWithinWindow(100, 100, buttons, 128, 128) {
		t.Fatalf("expected post-death attack edge to be detected")
	}
}

func TestHasPostDeathAttackWithinWindow_RequiresNewAttackPress(t *testing.T) {
	buttons := []*funData.PlayerButtons{
		{Frame: 99, Tick: 99, Buttons: uint64(common.ButtonAttack)},
		{Frame: 101, Tick: 101, Buttons: uint64(common.ButtonAttack)},
	}

	if hasPostDeathAttackWithinWindow(100, 100, buttons, 128, 128) {
		t.Fatalf("expected held pre-death attack to not count as new post-death attack")
	}
}

func TestAwpHoldDeathReactionExportFields_BlanksAttackOnlyReaction(t *testing.T) {
	event := &AwpHoldDeath{
		VictimReactionShotFrame:         0,
		HasPreDeathVictimAwpShot:        false,
		HasPostDeathVictimAttackTrigger: true,
		ShotOffsetFrame:                 0,
	}

	reactionFrame, offsetFrame := awpHoldDeathReactionExportFields(event)

	if reactionFrame != "" || offsetFrame != "" {
		t.Fatalf("expected attack-only reaction export fields to be blank, got %q %q", reactionFrame, offsetFrame)
	}
}
