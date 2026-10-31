package api

// Counts describe observed samples, not completeness of the recording. Numeric
// defaults must be interpreted together with their per-event availability flags.
type MatchDataQuality struct {
	FootstepsStatus                 string `json:"footstepsStatus"`
	FootstepEventCount              int    `json:"footstepEventCount"`
	ShotCount                       int    `json:"shotCount"`
	AimPunchSampleCount             int    `json:"aimPunchSampleCount"`
	AimPunchBaseSampleCount         int    `json:"aimPunchBaseSampleCount"`
	SoundFootstepCount              int    `json:"soundFootstepCount"`
	KillPenetrationMatchedCount     int    `json:"killPenetrationMatchedCount"`
	ViewPunchSampleCount            int    `json:"viewPunchSampleCount"`
	RecoilIndexSampleCount          int    `json:"recoilIndexSampleCount"`
	DamageCount                     int    `json:"damageCount"`
	BulletDamageMatchedCount        int    `json:"bulletDamageMatchedCount"`
	WallbangHeuristicEvaluatedCount int    `json:"wallbangHeuristicEvaluatedCount"`
}

func (match *Match) DataQuality() MatchDataQuality {
	quality := MatchDataQuality{
		FootstepsStatus:    "not_observed",
		FootstepEventCount: len(match.Footsteps),
		ShotCount:          len(match.Shots),
		DamageCount:        len(match.Damages),
	}
	if len(match.Footsteps) > 0 {
		quality.FootstepsStatus = "observed"
	}
	for _, footstep := range match.Footsteps {
		if footstep != nil && footstep.DataSource == "sound_event" {
			quality.SoundFootstepCount++
		}
	}
	for _, shot := range match.Shots {
		if shot == nil {
			continue
		}
		if shot.HasAimPunchData {
			quality.AimPunchSampleCount++
		}
		if shot.AimPunchBase != nil {
			quality.AimPunchBaseSampleCount++
		}
		if shot.HasViewPunchData {
			quality.ViewPunchSampleCount++
		}
		if shot.HasRecoilIndexData {
			quality.RecoilIndexSampleCount++
		}
	}
	for _, damage := range match.Damages {
		if damage == nil {
			continue
		}
		if damage.HasBulletDamageData {
			quality.BulletDamageMatchedCount++
		}
		if damage.WallbangDataSource == "kill_event" {
			quality.KillPenetrationMatchedCount++
		}
		if damage.HasWallbangHeuristicData {
			quality.WallbangHeuristicEvaluatedCount++
		}
	}
	return quality
}
