package api

import (
	"math"

	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

const (
	flashEstimateModel             = "cs2_client_14194_cpu"
	flashBuildUpSeconds            = 0.09444444626569748
	flashFullWhiteRemainingSeconds = 3.0
)

// FlashEstimate snapshots a conditional CS2 CPU rendering model at the kill.
// Client clock/frame phase and spectator scaling are not recorded by GOTV:
// elapsed uses demo ticks, scale is assumed to be 1, and pixels are unverified.
type FlashEstimate struct {
	Model           string  `json:"model"`
	Duration        float64 `json:"duration"`
	Elapsed         float64 `json:"elapsed"`
	Remaining       float64 `json:"remaining"`
	MaxAlpha        float64 `json:"maxAlpha"`
	Phase           string  `json:"phase"`
	OverlayAlpha    float64 `json:"overlayAlpha"`
	ScreenshotAlpha float64 `json:"screenshotAlpha"`
	IsFullWhite     bool    `json:"isFullWhite"`
}

func (analyzer *Analyzer) estimateKillerFlash(player *common.Player) *FlashEstimate {
	if !analyzer.isSource2 || player == nil {
		return nil
	}
	pawn := player.PlayerPawnEntity()
	if pawn == nil {
		return nil
	}
	alpha, exists := pawn.PropertyValue("m_flFlashMaxAlpha")
	if !exists || alpha.Any == nil {
		return nil
	}
	return estimateFlashAtTick(float64(player.FlashDuration), player.FlashTick,
		analyzer.currentTick(), analyzer.match.TickRate, float64(alpha.Float()))
}

func estimateFlashAtTick(duration float64, flashTick, tick int, tickRate, maxAlpha float64) *FlashEstimate {
	for _, value := range []float64{duration, tickRate, maxAlpha} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil
		}
	}
	if duration <= 0 || tickRate <= 0 || maxAlpha < 0 || maxAlpha > 255 || flashTick > tick {
		return nil
	}
	elapsed := (float64(tick) - float64(flashTick)) / tickRate
	if math.IsInf(elapsed, 0) {
		return nil
	}
	estimate := &FlashEstimate{
		Model: flashEstimateModel, Duration: duration, Elapsed: elapsed,
		Remaining: math.Max(0, duration-elapsed), MaxAlpha: maxAlpha,
		Phase: "buildup",
	}
	if elapsed < flashBuildUpSeconds {
		alpha := math.Min(maxAlpha, elapsed/flashBuildUpSeconds*maxAlpha) / 255
		estimate.OverlayAlpha, estimate.ScreenshotAlpha = alpha, alpha
		return estimate
	}

	estimate.Phase = "decay"
	ratio := math.Min(1, estimate.Remaining/flashFullWhiteRemainingSeconds)
	estimate.OverlayAlpha = maxAlpha / 255 * ratio * ratio
	estimate.ScreenshotAlpha = maxAlpha / 255 * math.Min(1, estimate.Remaining/duration)
	// At exactly three seconds the quadratic expression also reaches one.
	estimate.IsFullWhite = maxAlpha == 255 && estimate.Remaining >= flashFullWhiteRemainingSeconds
	return estimate
}

// FullWhiteKillCount counts enemy kills selected by the conditional flash model.
// A missing estimate is unavailable, not an observed zero opacity.
func (player *Player) FullWhiteKillCount() int {
	count := 0
	for _, kill := range player.match.Kills {
		if kill.isEnemyKill() && kill.KillerSteamID64 == player.SteamID64 &&
			kill.KillerFlashEstimate != nil && kill.KillerFlashEstimate.IsFullWhite {
			count++
		}
	}
	return count
}
