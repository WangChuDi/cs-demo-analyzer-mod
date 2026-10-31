package api

import "math"

type SelfFlashStats struct {
	SelfFlashCount    int     `json:"selfFlashCount"`
	SelfFlashDuration float64 `json:"selfFlashDuration"`
}

// Duration is the sum of positive effective-duration values recorded in self
// flash events, not a union of intervals or the player's total full-white time.
func (player *Player) SelfFlashStats() SelfFlashStats {
	stats := SelfFlashStats{}
	if player.SteamID64 == 0 {
		return stats
	}
	for _, flash := range player.match.PlayersFlashed {
		if flash == nil || flash.FlasherSteamID64 != player.SteamID64 || flash.FlashedSteamID64 != player.SteamID64 ||
			flash.IsFlasherControllingBot || flash.IsFlashedControllingBot || flash.Duration <= 0 {
			continue
		}
		duration := float64(flash.Duration)
		if math.IsNaN(duration) || math.IsInf(duration, 0) {
			continue
		}
		stats.SelfFlashCount++
		stats.SelfFlashDuration += duration
	}
	return stats
}
