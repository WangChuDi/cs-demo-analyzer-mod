package api

import (
	"math"
	"slices"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

const (
	awpPeekHoldSeconds     = 0.5
	awpPeekWindowSeconds   = 0.5
	awpPeekStopSeconds     = 0.3
	awpPeekMaxGap          = 0.125
	awpPeekMaxHoldSpeed    = 10.0
	awpPeekHoldRadius      = 8.0
	awpPeekHoldAngle       = 5.0
	awpPeekFacingAngle     = 10.0
	awpPeekMinDisplacement = 16.0
	awpPeekHistorySeconds  = 1.25
)

// AWPPeekEvidence describes an estimated hold-versus-peek duel, not visibility
// or the time when either client first rendered its opponent.
type AWPPeekEvidence struct {
	Model                  string  `json:"model"`
	ShotTick               int     `json:"shotTick"`
	HoldStartTick          int     `json:"holdStartTick"`
	HoldEndTick            int     `json:"holdEndTick"`
	HoldSeconds            float64 `json:"holdSeconds"`
	HoldMaxSpeed           float64 `json:"holdMaxSpeed"`
	HoldPositionDrift      float64 `json:"holdPositionDrift"`
	HoldViewDriftDegrees   float64 `json:"holdViewDriftDegrees"`
	FacingAngleDegrees     float64 `json:"facingAngleDegrees"`
	PeekDisplacementUnits  float64 `json:"peekDisplacementUnits"`
	StopToShotSeconds      float64 `json:"stopToShotSeconds"`
	ShotSpeed              float64 `json:"shotSpeed"`
	HolderFiredBeforeDeath bool    `json:"holderFiredBeforeDeath"`
}

type awpPeekSample struct {
	tick, frame, round        int
	position, eye, aim        r3.Vector
	speed                     float64
	weaponID                  string
	scoped, airborne, blinded bool
}

type awpPeekCandidate struct {
	killer, victim               awpPeekSample
	killerHistory, victimHistory []awpPeekSample
}

func awpPeekAngle(a, b r3.Vector) float64 {
	length := a.Norm() * b.Norm()
	if length == 0 {
		return math.NaN()
	}
	return math.Acos(math.Max(-1, math.Min(1, a.Dot(b)/length))) * 180 / math.Pi
}

func validAWPPeekSample(sample awpPeekSample) bool {
	for _, value := range []float64{sample.position.X, sample.position.Y, sample.position.Z,
		sample.eye.X, sample.eye.Y, sample.eye.Z, sample.aim.X, sample.aim.Y, sample.aim.Z, sample.speed} {
		if !summonFinite(value) {
			return false
		}
	}
	return sample.weaponID != "" && sample.aim.Norm() > 0 && sample.speed >= 0
}

func (analyzer *Analyzer) awpPeekSample(player *common.Player, frame int) (awpPeekSample, bool) {
	if !analyzer.isSource2 || player == nil || player.SteamID64 == 0 || player.IsBot || player.IsControllingBot() {
		return awpPeekSample{}, false
	}
	pawn := player.PlayerPawnEntity()
	weapon := player.ActiveWeapon()
	if pawn == nil || weapon == nil || weapon.Type != common.EqAWP || weapon.UniqueID() == 0 {
		return awpPeekSample{}, false
	}
	for _, name := range []string{"m_bIsScoped", "m_angEyeAngles", "m_fFlags"} {
		if value, ok := pawn.PropertyValue(name); !ok || value.Any == nil {
			return awpPeekSample{}, false
		}
	}
	eye, hasEye := player.PositionEyes()
	if !hasEye {
		return awpPeekSample{}, false
	}
	yaw, pitch := float64(player.ViewDirectionX())*math.Pi/180, float64(player.ViewDirectionY())*math.Pi/180
	velocity := getPlayerVelocity(player, analyzer)
	sample := awpPeekSample{tick: analyzer.currentTick(), frame: frame, round: analyzer.currentRound.Number,
		position: player.Position(), eye: eye, aim: r3.Vector{X: math.Cos(pitch) * math.Cos(yaw), Y: math.Cos(pitch) * math.Sin(yaw), Z: -math.Sin(pitch)},
		speed: math.Hypot(velocity.X, velocity.Y), weaponID: weapon.UniqueID().String(),
		scoped: player.IsScoped(), airborne: player.IsAirborne(), blinded: player.IsBlinded()}
	return sample, validAWPPeekSample(sample)
}

func appendAWPPeekSample(history []awpPeekSample, sample awpPeekSample, rate float64) []awpPeekSample {
	if !validAWPPeekSample(sample) || !summonFinite(rate) || rate <= 0 {
		return nil
	}
	if len(history) > 0 {
		last := history[len(history)-1]
		if sample.round != last.round || sample.weaponID != last.weaponID || sample.frame <= last.frame || sample.tick < last.tick ||
			float64(sample.tick-last.tick)/rate > awpPeekMaxGap {
			history = history[:0]
		}
	}
	if len(history) > 0 && history[len(history)-1].tick == sample.tick {
		history[len(history)-1] = sample
	} else {
		history = append(history, sample)
	}
	first := 0
	for first < len(history) && float64(sample.tick-history[first].tick)/rate > awpPeekHistorySeconds {
		first++
	}
	if first > 0 {
		history = history[:copy(history, history[first:])]
	}
	return history
}

func (analyzer *Analyzer) sampleAWPPeek() {
	if analyzer.parser.GameState().IsFreezetimePeriod() || analyzer.currentRound.EndTick > 0 {
		clear(analyzer.awpPeekHistories)
		return
	}
	if analyzer.awpPeekHistories == nil {
		analyzer.awpPeekHistories = map[uint64][]awpPeekSample{}
	}
	for _, player := range analyzer.parser.GameState().Participants().Playing() {
		if player == nil {
			continue
		}
		// v6 dispatches FrameDone after incrementing CurrentFrame.
		sample, known := analyzer.awpPeekSample(player, analyzer.parser.CurrentFrame()-1)
		if !known || !player.IsAlive() {
			delete(analyzer.awpPeekHistories, player.SteamID64)
			continue
		}
		analyzer.awpPeekHistories[player.SteamID64] = appendAWPPeekSample(analyzer.awpPeekHistories[player.SteamID64], sample, analyzer.parser.TickRate())
	}
}

func (analyzer *Analyzer) captureAWPPeek(kill *Kill, killer, victim *common.Player) {
	defer delete(analyzer.awpPeekHistories, kill.VictimSteamID64)
	if kill.WeaponName != constants.WeaponAWP || !kill.HasVictimActiveWeaponData || kill.VictimActiveWeaponName != constants.WeaponAWP || !kill.isEnemyKill() {
		return
	}
	killerSample, killerOK := analyzer.awpPeekSample(killer, kill.Frame)
	victimSample, victimOK := analyzer.awpPeekSample(victim, kill.Frame)
	if !killerOK || !victimOK {
		return
	}
	// Capture before FrameDone can replace the victim's live state. Classification
	// waits until all WeaponFire events exist, including later events in this frame.
	kill.awpPeekCandidate = &awpPeekCandidate{killer: killerSample, victim: victimSample,
		killerHistory: slices.Clone(analyzer.awpPeekHistories[kill.KillerSteamID64]),
		victimHistory: slices.Clone(analyzer.awpPeekHistories[kill.VictimSteamID64])}
}

// awpPeekWindow requires a complete, contiguous live history ending just before
// the reference frame. Same-tick earlier frames are valid; post-event frames are not.
func awpPeekWindow(history []awpPeekSample, end awpPeekSample, seconds, rate float64) ([]awpPeekSample, bool) {
	last := len(history) - 1
	for last >= 0 && (history[last].frame >= end.frame || history[last].tick > end.tick) {
		last--
	}
	if last < 0 || float64(end.tick-history[last].tick)/rate > awpPeekMaxGap {
		return nil, false
	}
	first := last
	for first > 0 && float64(end.tick-history[first].tick)/rate < seconds {
		first--
	}
	window := history[first : last+1]
	if float64(end.tick-window[0].tick)/rate < seconds || float64(end.tick-window[0].tick)/rate > seconds+awpPeekMaxGap {
		return nil, false
	}
	previous := window[0]
	for _, sample := range window {
		if !validAWPPeekSample(sample) || sample.round != end.round || sample.weaponID != end.weaponID ||
			sample.tick < previous.tick || sample.frame < previous.frame || float64(sample.tick-previous.tick)/rate > awpPeekMaxGap {
			return nil, false
		}
		previous = sample
	}
	return window, true
}

func classifyAWPPeek(kill *Kill, shots map[roundPlayerKey][]*Shot, rate float64) (bool, string, *AWPPeekEvidence) {
	if kill == nil {
		return false, "missing_kill", nil
	}
	if kill.WeaponName != constants.WeaponAWP {
		return true, "not_awp_duel", nil
	}
	if !kill.HasVictimActiveWeaponData {
		return false, "missing_active_weapon", nil
	}
	if kill.VictimActiveWeaponName != constants.WeaponAWP {
		return true, "not_awp_duel", nil
	}
	if !kill.isEnemyKill() || kill.IsKillerControllingBot || kill.IsVictimControllingBot || kill.IsThroughSmoke || kill.PenetratedObjects > 0 ||
		kill.IsKillerBlinded || kill.IsVictimBlinded || kill.IsKillerAirborne || kill.IsVictimAirborne {
		return true, "excluded", nil
	}
	candidate := kill.awpPeekCandidate
	if candidate == nil || !summonFinite(rate) || rate <= 0 {
		return false, "missing_live_state", nil
	}
	var fatalShot *Shot
	for _, shot := range shots[roundPlayerKey{kill.RoundNumber, kill.KillerSteamID64}] {
		if shot.Frame == kill.Frame && shot.Tick == kill.Tick && !shot.IsPlayerControllingBot {
			if fatalShot != nil {
				return false, "ambiguous_shot", nil
			}
			fatalShot = shot
		}
	}
	if fatalShot == nil {
		return false, "missing_shot", nil
	}
	killer, victim := candidate.killer, candidate.victim
	if !validAWPPeekSample(killer) || !validAWPPeekSample(victim) || killer.round != kill.RoundNumber || victim.round != kill.RoundNumber ||
		killer.tick != kill.Tick || victim.tick != kill.Tick || killer.frame != kill.Frame || victim.frame != kill.Frame {
		return false, "missing_live_state", nil
	}
	if fatalShot.WeaponID != killer.weaponID {
		return false, "missing_shot_weapon", nil
	}
	if killer.airborne || killer.blinded || victim.airborne || victim.blinded {
		return true, "excluded", nil
	}
	// Firing an AWP can clear scoped state. A recent recorded victim shot may
	// supply a pre-fire hold anchor, while the post-shot position/aim must stay held.
	holdEnd := victim
	var holderShot *Shot
	for _, shot := range shots[roundPlayerKey{kill.RoundNumber, kill.VictimSteamID64}] {
		if !shot.IsPlayerControllingBot && shot.WeaponID == victim.weaponID && shot.Frame <= fatalShot.Frame && shot.Tick <= fatalShot.Tick &&
			float64(fatalShot.Tick-shot.Tick)/rate <= awpPeekStopSeconds &&
			(holderShot == nil || shot.Frame > holderShot.Frame) {
			holderShot = shot
		}
	}
	if holderShot != nil {
		holdEnd.tick, holdEnd.frame = holderShot.Tick, holderShot.Frame
	}
	peek, peekOK := awpPeekWindow(candidate.killerHistory, killer, awpPeekWindowSeconds, rate)
	hold, holdOK := awpPeekWindow(candidate.victimHistory, holdEnd, awpPeekHoldSeconds, rate)
	if !peekOK || !holdOK {
		return false, "missing_history", nil
	}
	anchor := hold[0]
	heldToDeath, tailOK := awpPeekWindow(candidate.victimHistory, victim, float64(victim.tick-anchor.tick)/rate, rate)
	if !tailOK {
		return false, "missing_history", nil
	}
	evidence := &AWPPeekEvidence{Model: "awp_peek_v1", ShotTick: fatalShot.Tick, HoldStartTick: anchor.tick, HoldEndTick: holdEnd.tick,
		HoldSeconds: float64(holdEnd.tick-anchor.tick) / rate, ShotSpeed: killer.speed, HolderFiredBeforeDeath: holderShot != nil}
	// Include the tail and event-time state, allowing only a recorded shot to
	// explain scope loss. Moving away or turning after that shot still disqualifies.
	for _, sample := range append(slices.Clone(heldToDeath), victim) {
		if sample.airborne || sample.blinded || sample.speed > awpPeekMaxHoldSpeed ||
			(!sample.scoped && (holderShot == nil || sample.frame < holderShot.Frame)) {
			return true, "holder_not_stable", nil
		}
		evidence.HoldMaxSpeed = math.Max(evidence.HoldMaxSpeed, sample.speed)
		evidence.HoldPositionDrift = math.Max(evidence.HoldPositionDrift, sample.position.Sub(anchor.position).Norm())
		evidence.HoldViewDriftDegrees = math.Max(evidence.HoldViewDriftDegrees, awpPeekAngle(anchor.aim, sample.aim))
	}
	if evidence.HoldPositionDrift > awpPeekHoldRadius || evidence.HoldViewDriftDegrees > awpPeekHoldAngle+1e-9 {
		return true, "holder_not_stable", nil
	}
	evidence.FacingAngleDegrees = awpPeekAngle(anchor.aim, killer.eye.Sub(anchor.eye))
	if !summonFinite(evidence.FacingAngleDegrees) || evidence.FacingAngleDegrees > awpPeekFacingAngle+1e-9 {
		return true, "not_facing", nil
	}
	evidence.PeekDisplacementUnits = math.Hypot(killer.position.X-peek[0].position.X, killer.position.Y-peek[0].position.Y)
	lastMovingTick := -1
	for _, sample := range append(slices.Clone(peek), killer) {
		if sample.airborne || sample.blinded {
			return true, "excluded", nil
		}
		if sample.speed > awpPeekMaxHoldSpeed {
			lastMovingTick = sample.tick
		}
	}
	if lastMovingTick < 0 || evidence.PeekDisplacementUnits < awpPeekMinDisplacement {
		return true, "peeker_not_moving", nil
	}
	evidence.StopToShotSeconds = float64(killer.tick-lastMovingTick) / rate
	if evidence.StopToShotSeconds > awpPeekStopSeconds {
		return true, "peeker_stopped_too_long", nil
	}
	return true, "qualified", evidence
}

func markAWPPeekKills(match *Match) {
	if match == nil {
		return
	}
	shots := buildVictimAwpShotsByRoundPlayer(match)
	for _, kill := range match.Kills {
		if kill == nil {
			continue
		}
		if kill.awpPeekCandidate == nil && kill.AWPPeekReason != "" {
			continue
		}
		kill.HasAWPPeekData, kill.AWPPeekReason, kill.AWPPeek = classifyAWPPeek(kill, shots, match.TickRate)
		kill.IsAWPPeekKill = kill.AWPPeek != nil
		kill.awpPeekCandidate = nil
	}
}

func (player *Player) AWPPeekKillCount() int {
	count := 0
	for _, kill := range player.match.Kills {
		if kill.isEnemyKill() && kill.HasAWPPeekData && kill.IsAWPPeekKill && kill.KillerSteamID64 == player.SteamID64 {
			count++
		}
	}
	return count
}

func (player *Player) AWPPeekDeathCount() int {
	count := 0
	for _, kill := range player.match.Kills {
		if kill.isEnemyKill() && kill.HasAWPPeekData && kill.IsAWPPeekKill && kill.VictimSteamID64 == player.SteamID64 {
			count++
		}
	}
	return count
}
