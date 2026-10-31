package api

import (
	"math"

	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
)

const (
	summonHoldSeconds = 1.0
	summonDeathWindow = 0.5
	summonMaxGap      = 0.125
	summonMaxSpeed    = 34.0
	summonHoldJitter  = 10.0
	summonTurnAngle   = 60.0
	summonHoldRadius  = 16.0
)

type SummonEnemyEvidence struct {
	TriggerTick           int                  `json:"triggerTick"`
	DelaySeconds          float64              `json:"delaySeconds"`
	HoldSeconds           float64              `json:"holdSeconds"`
	Reasons               []string             `json:"reasons"`
	WeaponBefore          constants.WeaponName `json:"weaponBefore"`
	WeaponAfter           constants.WeaponName `json:"weaponAfter"`
	TurnAngle             float64              `json:"turnAngle"`
	HoldFacingKillerAngle float64              `json:"holdFacingKillerAngle"`
}

type summonSample struct {
	tick, frame, round int
	position           r3.Vector
	yaw, speed         float64
	weapon             constants.WeaponName
	weaponID           string
	gun, valid         bool
}

type summonAction struct {
	before      summonSample
	tick, frame int
	holdSeconds float64
	switched    bool
}

type summonTracker struct {
	last, anchor summonSample
	hasLast      bool
	startTick    int
	holding      bool
	action       *summonAction
}

func summonAngle(left, right float64) float64 {
	return math.Abs(math.Remainder(left-right, 360))
}

func summonFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func (tracker *summonTracker) start(sample summonSample) {
	*tracker = summonTracker{last: sample, hasLast: true, anchor: sample, startTick: sample.tick, holding: sample.valid && sample.gun}
}

func (tracker *summonTracker) update(sample summonSample, rate float64) {
	if !sample.valid || !summonFinite(rate) || rate <= 0 || !summonFinite(sample.yaw) || !summonFinite(sample.speed) ||
		!summonFinite(sample.position.X) || !summonFinite(sample.position.Y) || !summonFinite(sample.position.Z) ||
		sample.speed > summonMaxSpeed || sample.weaponID == "" {
		*tracker = summonTracker{}
		return
	}
	if !tracker.hasLast || sample.round != tracker.last.round || sample.tick < tracker.last.tick || sample.frame <= tracker.last.frame ||
		float64(sample.tick-tracker.last.tick)/rate > summonMaxGap {
		tracker.start(sample)
		return
	}
	previous := tracker.last
	tracker.last = sample
	angle := summonAngle(sample.yaw, tracker.anchor.yaw)
	distance := sample.position.Sub(tracker.anchor.position).Norm()
	if tracker.action != nil {
		if float64(sample.tick-tracker.action.tick)/rate > summonDeathWindow || distance > summonHoldRadius ||
			(sample.weaponID == tracker.anchor.weaponID && angle <= summonHoldJitter) {
			tracker.start(sample)
			return
		}
		tracker.action.switched = tracker.action.switched || sample.weaponID != tracker.anchor.weaponID
		return
	}
	if !tracker.holding {
		tracker.start(sample)
		return
	}
	if sample.weaponID == tracker.anchor.weaponID && angle <= summonHoldJitter && distance <= summonHoldRadius {
		return
	}
	hold := float64(previous.tick-tracker.startTick) / rate
	if hold >= summonHoldSeconds && distance <= summonHoldRadius {
		tracker.action = &summonAction{before: tracker.anchor, tick: sample.tick, frame: sample.frame,
			holdSeconds: hold, switched: sample.weaponID != tracker.anchor.weaponID}
		tracker.holding = false
		return
	}
	tracker.start(sample)
}

func (tracker *summonTracker) evidence(kill *Kill, rate float64) *SummonEnemyEvidence {
	if tracker == nil || tracker.action == nil || !kill.isEnemyKill() || kill.IsKillerControllingBot || kill.IsVictimControllingBot ||
		rate <= 0 || !summonFinite(rate) {
		return nil
	}
	action := tracker.action
	delay := float64(kill.Tick-action.tick) / rate
	if kill.RoundNumber != action.before.round || action.tick >= kill.Tick || action.frame >= kill.Frame ||
		tracker.last.tick > kill.Tick || tracker.last.frame >= kill.Frame ||
		float64(kill.Tick-tracker.last.tick)/rate > summonMaxGap || delay > summonDeathWindow {
		return nil
	}
	// Event-time yaw can confirm the turn, but never creates a trigger: a live
	// pre-death frame must already contain the action. Ignore absent/dead weapons.
	turn := summonAngle(float64(kill.VictimYaw), action.before.yaw)
	if !summonFinite(turn) || (kill.HasVictimActiveWeaponData && kill.VictimActiveWeaponName == action.before.weapon && turn <= summonHoldJitter) {
		return nil
	}
	killerBearing := math.Atan2(kill.KillerY-action.before.position.Y, kill.KillerX-action.before.position.X) * 180 / math.Pi
	facing := summonAngle(killerBearing, action.before.yaw)
	if !summonFinite(facing) || facing > 30 {
		return nil
	}
	reasons := []string{}
	if turn >= summonTurnAngle && summonAngle(tracker.last.yaw, action.before.yaw) >= summonTurnAngle {
		reasons = append(reasons, "turn_away")
	}
	if action.switched {
		reasons = append(reasons, "weapon_switch")
	}
	if len(reasons) == 0 {
		return nil
	}
	return &SummonEnemyEvidence{TriggerTick: action.tick, DelaySeconds: delay, HoldSeconds: action.holdSeconds,
		Reasons: reasons, WeaponBefore: action.before.weapon, WeaponAfter: tracker.last.weapon,
		TurnAngle: turn, HoldFacingKillerAngle: facing}
}

func (analyzer *Analyzer) resetBehaviorTracking() {
	clear(analyzer.summonTrackers)
	analyzer.armorCheckRound = 0
}

func (analyzer *Analyzer) registerBehaviorHandlers() {
	analyzer.parser.RegisterEventHandler(func(event events.RoundFreezetimeChanged) {
		analyzer.resetBehaviorTracking()
		if !event.NewIsFreezetime && analyzer.matchStarted() {
			analyzer.armorCheckRound, analyzer.armorCheckTick = analyzer.currentRound.Number, analyzer.currentTick()
		}
	})
	analyzer.parser.RegisterEventHandler(func(events.RoundStart) { analyzer.resetBehaviorTracking() })
}

func (analyzer *Analyzer) sampleSummonEnemy() {
	if analyzer.parser.GameState().IsFreezetimePeriod() || analyzer.currentRound.EndTick > 0 {
		clear(analyzer.summonTrackers)
		return
	}
	if analyzer.summonTrackers == nil {
		analyzer.summonTrackers = map[uint64]*summonTracker{}
	}
	for _, player := range analyzer.parser.GameState().Participants().Playing() {
		if player == nil || player.SteamID64 == 0 {
			continue
		}
		id := player.SteamID64
		if player.IsBot || player.IsControllingBot() || !player.IsAlive() || player.IsAirborne() || player.IsBlinded() {
			delete(analyzer.summonTrackers, id)
			continue
		}
		weapon := player.ActiveWeapon()
		if weapon == nil || weapon.UniqueID() == 0 {
			delete(analyzer.summonTrackers, id)
			continue
		}
		name, known := equipmentToWeaponName[weapon.Type]
		velocity := getPlayerVelocity(player, analyzer)
		// v6 increments CurrentFrame before dispatching FrameDone. Normalize to
		// the completed frame so the next frame's Kill can use this live sample.
		sample := summonSample{tick: analyzer.currentTick(), frame: analyzer.parser.CurrentFrame() - 1, round: analyzer.currentRound.Number,
			position: player.Position(), yaw: float64(player.ViewDirectionX()), speed: math.Hypot(velocity.X, velocity.Y),
			weapon: name, weaponID: weapon.UniqueID().String(), gun: isFirearmType(getEquipmentWeaponType(*weapon)),
			valid: known && name != constants.WeaponUnknown}
		tracker := analyzer.summonTrackers[id]
		if tracker == nil {
			tracker = &summonTracker{}
			analyzer.summonTrackers[id] = tracker
		}
		tracker.update(sample, analyzer.parser.TickRate())
	}
}

func (analyzer *Analyzer) applySummonEnemyDeath(kill *Kill) {
	defer delete(analyzer.summonTrackers, kill.VictimSteamID64)
	tracker := analyzer.summonTrackers[kill.VictimSteamID64]
	kill.SummonEnemy = tracker.evidence(kill, analyzer.match.TickRate)
	kill.IsSummonEnemyDeath = kill.SummonEnemy != nil
}

func (analyzer *Analyzer) interruptSummonHold(id uint64) {
	// A first hit after the action is part of the ambush. A hit before the action
	// invalidates the earlier peaceful hold; do not infer reaction as distraction.
	if tracker := analyzer.summonTrackers[id]; tracker != nil && tracker.action == nil {
		delete(analyzer.summonTrackers, id)
	}
}

func (player *Player) SummonEnemyDeathCount() int {
	count := 0
	for _, kill := range player.match.Kills {
		if kill.isEnemyKill() && kill.VictimSteamID64 == player.SteamID64 && kill.IsSummonEnemyDeath {
			count++
		}
	}
	return count
}
