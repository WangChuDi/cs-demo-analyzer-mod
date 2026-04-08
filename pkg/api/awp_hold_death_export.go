package api

import "github.com/akiver/cs-demo-analyzer/internal/converters"

func awpHoldDeathReactionExportFields(event *AwpHoldDeath) (reactionFrame string, offsetFrame string) {
	if event == nil || !event.HasPreDeathVictimAwpShot {
		return "", ""
	}

	return converters.IntToString(event.VictimReactionShotFrame),
		converters.IntToString(event.ShotOffsetFrame)
}
