package api

import "testing"

func TestAwpHoldDeathReactionExportFields_BlanksMissingReactionShot(t *testing.T) {
	event := &AwpHoldDeath{
		VictimReactionShotFrame:  0,
		HasPreDeathVictimAwpShot: false,
		ShotOffsetFrame:          128,
	}

	reactionFrame, offsetFrame := awpHoldDeathReactionExportFields(event)

	if reactionFrame != "" || offsetFrame != "" {
		t.Fatalf("expected missing reaction-shot export fields to be blank, got %q %q", reactionFrame, offsetFrame)
	}
}

func TestAwpHoldDeathReactionExportFields_PreservesReactionShot(t *testing.T) {
	event := &AwpHoldDeath{
		VictimReactionShotFrame:  123,
		HasPreDeathVictimAwpShot: true,
		ShotOffsetFrame:          0,
	}

	reactionFrame, offsetFrame := awpHoldDeathReactionExportFields(event)

	if reactionFrame != "123" {
		t.Fatalf("expected reaction frame 123, got %q", reactionFrame)
	}
	if offsetFrame != "0" {
		t.Fatalf("expected offset frame 0, got %q", offsetFrame)
	}
}
