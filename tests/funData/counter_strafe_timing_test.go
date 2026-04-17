package funData

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/akiver/cs-demo-analyzer/pkg/api"
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	demoinfocs "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	events "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/events"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type standaloneCounterStrafeDirection int

const (
	standaloneDirectionAll standaloneCounterStrafeDirection = iota
	standaloneDirectionAToD
	standaloneDirectionDToA
	standaloneDirectionWToS
	standaloneDirectionSToW
)

type standaloneCounterStrafeComboDirection int

const (
	standaloneComboDirectionAll standaloneCounterStrafeComboDirection = iota
	standaloneComboDirectionWAToSD
	standaloneComboDirectionWDToSA
	standaloneComboDirectionSAToWD
	standaloneComboDirectionSDToWA
)

const standalonePerfectCounterStrafeDeltaWindow = 3
const standalonePerfectCounterStrafeComboDeltaWindow = 10

type standaloneButtonSnapshot struct {
	tick    int
	buttons uint64
}

type standaloneShot struct {
	roundNumber int
	tick        int
	steamID64   uint64
	recoilIndex float32
	isBot       bool
}

type standaloneCounterStrafeSample struct {
	deltaTick      int
	direction      standaloneCounterStrafeDirection
	completionTick int
}

type standaloneShotAxisSamples struct {
	hasAToD bool
	aToD    standaloneCounterStrafeSample
	hasDToA bool
	dToA    standaloneCounterStrafeSample
	hasWToS bool
	wToS    standaloneCounterStrafeSample
	hasSToW bool
	sToW    standaloneCounterStrafeSample
}

type standaloneCounterStrafeComboSample struct {
	horizontalDeltaTick int
	verticalDeltaTick   int
	direction           standaloneCounterStrafeComboDirection
	completionTick      int
}

type standaloneTransitionTracker struct {
	direction           standaloneCounterStrafeDirection
	originalButton      common.ButtonBitMask
	reverseButton       common.ButtonBitMask
	originalPressed     bool
	reversePressed      bool
	hasObservedOriginal bool
	hasRelease          bool
	releaseTick         int
	hasReversePress     bool
	reversePressTick    int
	hasLatestSample     bool
	latestSample        standaloneCounterStrafeSample
}

type standaloneComboTransitionTracker struct {
	direction                  standaloneCounterStrafeComboDirection
	originalHorizontalButton   common.ButtonBitMask
	reverseHorizontalButton    common.ButtonBitMask
	originalVerticalButton     common.ButtonBitMask
	reverseVerticalButton      common.ButtonBitMask
	originalHorizontalPressed  bool
	reverseHorizontalPressed   bool
	originalVerticalPressed    bool
	reverseVerticalPressed     bool
	hasObservedOriginalCombo   bool
	hasHorizontalRelease       bool
	horizontalReleaseTick      int
	hasHorizontalReversePress  bool
	horizontalReversePressTick int
	hasVerticalRelease         bool
	verticalReleaseTick        int
	hasVerticalReversePress    bool
	verticalReversePressTick   int
	hasLatestSample            bool
	latestSample               standaloneCounterStrafeComboSample
}

type standaloneCounterStrafeStats struct {
	averageDeltaTick float64
	stdDevTick       float64
	perfectRate      float32
}

func TestCounterStrafeTiming(t *testing.T) {
	demoName := "renown_match_8_2025_mirage"
	demoPath := filepath.Join("..", "..", "cs-demos", "cs2", demoName+".dem")

	match, err := api.AnalyzeDemo(demoPath, api.AnalyzeDemoOptions{
		Source: constants.DemoSourceRenown,
	})
	if err != nil {
		t.Fatalf("failed to analyze demo through API: %v", err)
	}

	standaloneStatsBySteamID, standaloneComboStatsBySteamID, err := computeStandaloneCounterStrafeStats(demoPath)
	if err != nil {
		t.Fatalf("failed to compute standalone counter-strafe stats: %v", err)
	}

	assertCounterStrafeTimingPlayer(t, match, standaloneStatsBySteamID, standaloneComboStatsBySteamID, 76561198029485456, "whatsnxt")
	assertCounterStrafeTimingPlayer(t, match, standaloneStatsBySteamID, standaloneComboStatsBySteamID, 76561198225661896, "Vit0-1337")
}

func computeStandaloneCounterStrafeStats(demoPath string) (map[uint64]map[standaloneCounterStrafeDirection]standaloneCounterStrafeStats, map[uint64]map[standaloneCounterStrafeComboDirection]standaloneCounterStrafeStats, error) {
	file, err := os.Open(demoPath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	parser := demoinfocs.NewParser(file)
	defer parser.Close()

	matchStarted := false
	currentRoundNumber := 1
	hasSeenRoundStart := false
	buttonsByRoundPlayer := make(map[int]map[uint64][]standaloneButtonSnapshot)
	shotsByRoundPlayer := make(map[int]map[uint64][]standaloneShot)

	parser.RegisterEventHandler(func(event events.MatchStartedChanged) {
		matchStarted = event.NewIsStarted
	})

	parser.RegisterEventHandler(func(event events.RoundStart) {
		if !matchStarted {
			return
		}
		if hasSeenRoundStart {
			currentRoundNumber++
		} else {
			hasSeenRoundStart = true
		}
	})

	parser.RegisterEventHandler(func(event events.PlayerButtonsStateUpdate) {
		if !matchStarted || event.Player == nil {
			return
		}
		if _, ok := buttonsByRoundPlayer[currentRoundNumber]; !ok {
			buttonsByRoundPlayer[currentRoundNumber] = make(map[uint64][]standaloneButtonSnapshot)
		}
		buttonsByRoundPlayer[currentRoundNumber][event.Player.SteamID64] = append(
			buttonsByRoundPlayer[currentRoundNumber][event.Player.SteamID64],
			standaloneButtonSnapshot{tick: parser.GameState().IngameTick(), buttons: event.ButtonsState},
		)
	})

	parser.RegisterEventHandler(func(event events.WeaponFire) {
		if !matchStarted || event.Shooter == nil {
			return
		}

		var weaponEntity st.Entity
		activeWeapon := event.Shooter.ActiveWeapon()
		if activeWeapon != nil {
			weaponEntity = activeWeapon.Entity
		} else if event.Weapon.Entity != nil {
			weaponEntity = event.Weapon.Entity
		}

		var recoilIndex float32
		if weaponEntity != nil {
			if prop, exists := weaponEntity.PropertyValue("m_flRecoilIndex"); exists {
				recoilIndex = prop.Float()
			}
		}

		if _, ok := shotsByRoundPlayer[currentRoundNumber]; !ok {
			shotsByRoundPlayer[currentRoundNumber] = make(map[uint64][]standaloneShot)
		}
		shotsByRoundPlayer[currentRoundNumber][event.Shooter.SteamID64] = append(
			shotsByRoundPlayer[currentRoundNumber][event.Shooter.SteamID64],
			standaloneShot{
				roundNumber: currentRoundNumber,
				tick:        parser.GameState().IngameTick(),
				steamID64:   event.Shooter.SteamID64,
				recoilIndex: recoilIndex,
				isBot:       event.Shooter.IsControllingBot(),
			},
		)
	})

	if err := parser.ParseToEnd(); err != nil {
		return nil, nil, err
	}

	oneDimensionalSamplesBySteamID := make(map[uint64][]standaloneCounterStrafeSample)
	comboSamplesBySteamID := make(map[uint64][]standaloneCounterStrafeComboSample)
	for roundNumber, shotsByPlayer := range shotsByRoundPlayer {
		for steamID64, shots := range shotsByPlayer {
			buttons := buttonsByRoundPlayer[roundNumber][steamID64]
			if len(buttons) == 0 {
				continue
			}

			var oneDimensionalSamples []standaloneCounterStrafeSample
			var comboSamples []standaloneCounterStrafeComboSample
			previousShotTick := -1
			for _, shot := range shots {
				if shot.isBot || shot.recoilIndex != 1 {
					continue
				}
				if sample, ok := collectStandaloneCounterStrafeSampleForShot(buttons, previousShotTick, shot.tick); ok {
					oneDimensionalSamples = append(oneDimensionalSamples, *sample)
				}
				if comboSample, ok := collectStandaloneCounterStrafeComboSampleForShot(buttons, previousShotTick, shot.tick); ok {
					comboSamples = append(comboSamples, *comboSample)
				}
				previousShotTick = shot.tick
			}

			if len(oneDimensionalSamples) > 0 {
				oneDimensionalSamplesBySteamID[steamID64] = append(oneDimensionalSamplesBySteamID[steamID64], oneDimensionalSamples...)
			}
			if len(comboSamples) > 0 {
				comboSamplesBySteamID[steamID64] = append(comboSamplesBySteamID[steamID64], comboSamples...)
			}
		}
	}

	oneDimensionalStatsBySteamID := make(map[uint64]map[standaloneCounterStrafeDirection]standaloneCounterStrafeStats)
	for steamID64, samples := range oneDimensionalSamplesBySteamID {
		oneDimensionalStatsBySteamID[steamID64] = map[standaloneCounterStrafeDirection]standaloneCounterStrafeStats{
			standaloneDirectionAll:  summarizeStandaloneCounterStrafeSamples(samples, standaloneDirectionAll),
			standaloneDirectionAToD: summarizeStandaloneCounterStrafeSamples(samples, standaloneDirectionAToD),
			standaloneDirectionDToA: summarizeStandaloneCounterStrafeSamples(samples, standaloneDirectionDToA),
			standaloneDirectionWToS: summarizeStandaloneCounterStrafeSamples(samples, standaloneDirectionWToS),
			standaloneDirectionSToW: summarizeStandaloneCounterStrafeSamples(samples, standaloneDirectionSToW),
		}
	}

	comboStatsBySteamID := make(map[uint64]map[standaloneCounterStrafeComboDirection]standaloneCounterStrafeStats)
	for steamID64, samples := range comboSamplesBySteamID {
		comboStatsBySteamID[steamID64] = map[standaloneCounterStrafeComboDirection]standaloneCounterStrafeStats{
			standaloneComboDirectionAll: summarizeStandaloneCounterStrafeComboSamples(samples, standaloneComboDirectionAll),
		}
	}

	return oneDimensionalStatsBySteamID, comboStatsBySteamID, nil
}

func assertCounterStrafeTimingPlayer(t *testing.T, match *api.Match, standaloneStatsBySteamID map[uint64]map[standaloneCounterStrafeDirection]standaloneCounterStrafeStats, standaloneComboStatsBySteamID map[uint64]map[standaloneCounterStrafeComboDirection]standaloneCounterStrafeStats, steamID64 uint64, name string) {
	t.Helper()

	player := match.PlayersBySteamID[steamID64]
	if player == nil {
		t.Fatalf("expected player %s (%d) to exist in analyzed match", name, steamID64)
	}

	statsByDirection, ok := standaloneStatsBySteamID[steamID64]
	if !ok {
		t.Fatalf("expected standalone stats for %s (%d)", name, steamID64)
	}
	comboStatsByDirection, ok := standaloneComboStatsBySteamID[steamID64]
	if !ok {
		t.Fatalf("expected standalone combo stats for %s (%d)", name, steamID64)
	}

	allStats := statsByDirection[standaloneDirectionAll]
	aToDStats := statsByDirection[standaloneDirectionAToD]
	dToAStats := statsByDirection[standaloneDirectionDToA]
	wToSStats := statsByDirection[standaloneDirectionWToS]
	sToWStats := statsByDirection[standaloneDirectionSToW]
	comboAllStats := comboStatsByDirection[standaloneComboDirectionAll]

	fmt.Printf("standalone counter-strafe timing %s: avg=%.3f stddev=%.3f A->D=%.3f/%.2f%% D->A=%.3f/%.2f%% W->S=%.3f/%.2f%% S->W=%.3f/%.2f%% combo=%.3f/%.2f%% perfect=%.2f%%\n",
		name,
		allStats.averageDeltaTick,
		allStats.stdDevTick,
		aToDStats.averageDeltaTick,
		aToDStats.perfectRate,
		dToAStats.averageDeltaTick,
		dToAStats.perfectRate,
		wToSStats.averageDeltaTick,
		wToSStats.perfectRate,
		sToWStats.averageDeltaTick,
		sToWStats.perfectRate,
		comboAllStats.averageDeltaTick,
		comboAllStats.perfectRate,
		allStats.perfectRate,
	)

	assertFloat64Near(t, name, "avg delta tick", player.CounterStrafingAverageDeltaTick(), allStats.averageDeltaTick)
	assertFloat64Near(t, name, "delta stddev tick", player.CounterStrafingDeltaStdDevTick(), allStats.stdDevTick)
	assertFloat32Near(t, name, "perfect rate", player.CounterStrafingPerfectRate(), allStats.perfectRate)
	assertFloat64Near(t, name, "A->D avg delta tick", player.CounterStrafingAToDAverageDeltaTick(), aToDStats.averageDeltaTick)
	assertFloat32Near(t, name, "A->D perfect rate", player.CounterStrafingAToDPerfectRate(), aToDStats.perfectRate)
	assertFloat64Near(t, name, "D->A avg delta tick", player.CounterStrafingDToAAverageDeltaTick(), dToAStats.averageDeltaTick)
	assertFloat32Near(t, name, "D->A perfect rate", player.CounterStrafingDToAPerfectRate(), dToAStats.perfectRate)
	assertFloat64Near(t, name, "W->S avg delta tick", player.CounterStrafingWToSAverageDeltaTick(), wToSStats.averageDeltaTick)
	assertFloat32Near(t, name, "W->S perfect rate", player.CounterStrafingWToSPerfectRate(), wToSStats.perfectRate)
	assertFloat64Near(t, name, "S->W avg delta tick", player.CounterStrafingSToWAverageDeltaTick(), sToWStats.averageDeltaTick)
	assertFloat32Near(t, name, "S->W perfect rate", player.CounterStrafingSToWPerfectRate(), sToWStats.perfectRate)
	assertFloat64Near(t, name, "combo avg delta tick", player.CounterStrafingComboAverageDeltaTick(), comboAllStats.averageDeltaTick)
	assertFloat64Near(t, name, "combo delta stddev tick", player.CounterStrafingComboDeltaStdDevTick(), comboAllStats.stdDevTick)
	assertFloat32Near(t, name, "combo perfect rate", player.CounterStrafingComboPerfectRate(), comboAllStats.perfectRate)
}

func collectStandaloneCounterStrafeAxisSamplesForShot(buttons []standaloneButtonSnapshot, startExclusiveTick int, shotTick int) standaloneShotAxisSamples {
	lastMask := uint64(0)
	startIndex := 0
	for startIndex < len(buttons) && buttons[startIndex].tick <= startExclusiveTick {
		lastMask = buttons[startIndex].buttons
		startIndex++
	}

	trackers := []standaloneTransitionTracker{
		newStandaloneTransitionTracker(standaloneDirectionAToD, common.ButtonMoveLeft, common.ButtonMoveRight),
		newStandaloneTransitionTracker(standaloneDirectionDToA, common.ButtonMoveRight, common.ButtonMoveLeft),
		newStandaloneTransitionTracker(standaloneDirectionWToS, common.ButtonForward, common.ButtonBack),
		newStandaloneTransitionTracker(standaloneDirectionSToW, common.ButtonBack, common.ButtonForward),
	}
	for index := range trackers {
		trackers[index].initialize(lastMask)
	}

	for index := startIndex; index < len(buttons); index++ {
		buttonState := buttons[index]
		if buttonState.tick > shotTick {
			break
		}

		for trackerIndex := range trackers {
			trackers[trackerIndex].apply(buttonState.buttons, buttonState.tick)
		}
	}

	axisSamples := standaloneShotAxisSamples{}
	for _, tracker := range trackers {
		if !tracker.hasLatestSample {
			continue
		}

		sample := tracker.latestSample
		switch sample.direction {
		case standaloneDirectionAToD:
			axisSamples.hasAToD = true
			axisSamples.aToD = sample
		case standaloneDirectionDToA:
			axisSamples.hasDToA = true
			axisSamples.dToA = sample
		case standaloneDirectionWToS:
			axisSamples.hasWToS = true
			axisSamples.wToS = sample
		case standaloneDirectionSToW:
			axisSamples.hasSToW = true
			axisSamples.sToW = sample
		}
	}

	return axisSamples
}

func collectStandaloneCounterStrafeSampleForShot(buttons []standaloneButtonSnapshot, startExclusiveTick int, shotTick int) (*standaloneCounterStrafeSample, bool) {
	axisSamples := collectStandaloneCounterStrafeAxisSamplesForShot(buttons, startExclusiveTick, shotTick)
	samples := make([]standaloneCounterStrafeSample, 0, 4)
	if axisSamples.hasAToD {
		samples = append(samples, axisSamples.aToD)
	}
	if axisSamples.hasDToA {
		samples = append(samples, axisSamples.dToA)
	}
	if axisSamples.hasWToS {
		samples = append(samples, axisSamples.wToS)
	}
	if axisSamples.hasSToW {
		samples = append(samples, axisSamples.sToW)
	}

	if len(samples) == 0 {
		return nil, false
	}

	latestSample := samples[0]
	for _, sample := range samples[1:] {
		if sample.completionTick > latestSample.completionTick {
			latestSample = sample
		}
	}

	return &latestSample, true
}

func collectStandaloneCounterStrafeComboSampleForShot(buttons []standaloneButtonSnapshot, startExclusiveTick int, shotTick int) (*standaloneCounterStrafeComboSample, bool) {
	lastMask := uint64(0)
	startIndex := 0
	for startIndex < len(buttons) && buttons[startIndex].tick <= startExclusiveTick {
		lastMask = buttons[startIndex].buttons
		startIndex++
	}

	trackers := []standaloneComboTransitionTracker{
		newStandaloneComboTransitionTracker(standaloneComboDirectionWAToSD, common.ButtonMoveLeft, common.ButtonMoveRight, common.ButtonForward, common.ButtonBack),
		newStandaloneComboTransitionTracker(standaloneComboDirectionWDToSA, common.ButtonMoveRight, common.ButtonMoveLeft, common.ButtonForward, common.ButtonBack),
		newStandaloneComboTransitionTracker(standaloneComboDirectionSAToWD, common.ButtonMoveLeft, common.ButtonMoveRight, common.ButtonBack, common.ButtonForward),
		newStandaloneComboTransitionTracker(standaloneComboDirectionSDToWA, common.ButtonMoveRight, common.ButtonMoveLeft, common.ButtonBack, common.ButtonForward),
	}
	for index := range trackers {
		trackers[index].initialize(lastMask)
	}

	for index := startIndex; index < len(buttons); index++ {
		buttonState := buttons[index]
		if buttonState.tick > shotTick {
			break
		}

		for trackerIndex := range trackers {
			trackers[trackerIndex].apply(buttonState.buttons, buttonState.tick)
		}
	}

	comboSamples := make([]standaloneCounterStrafeComboSample, 0, len(trackers))
	for _, tracker := range trackers {
		if tracker.hasLatestSample {
			comboSamples = append(comboSamples, tracker.latestSample)
		}
	}

	if len(comboSamples) == 0 {
		return nil, false
	}

	latestSample := comboSamples[0]
	for _, sample := range comboSamples[1:] {
		if sample.completionTick > latestSample.completionTick {
			latestSample = sample
		}
	}

	return &latestSample, true
}

func newStandaloneTransitionTracker(direction standaloneCounterStrafeDirection, originalButton common.ButtonBitMask, reverseButton common.ButtonBitMask) standaloneTransitionTracker {
	return standaloneTransitionTracker{
		direction:      direction,
		originalButton: originalButton,
		reverseButton:  reverseButton,
	}
}

func newStandaloneComboTransitionTracker(direction standaloneCounterStrafeComboDirection, originalHorizontalButton common.ButtonBitMask, reverseHorizontalButton common.ButtonBitMask, originalVerticalButton common.ButtonBitMask, reverseVerticalButton common.ButtonBitMask) standaloneComboTransitionTracker {
	return standaloneComboTransitionTracker{
		direction:                direction,
		originalHorizontalButton: originalHorizontalButton,
		reverseHorizontalButton:  reverseHorizontalButton,
		originalVerticalButton:   originalVerticalButton,
		reverseVerticalButton:    reverseVerticalButton,
	}
}

func isStandaloneButtonPressed(mask uint64, button common.ButtonBitMask) bool {
	return mask&uint64(button) != 0
}

func (tracker *standaloneTransitionTracker) initialize(mask uint64) {
	tracker.originalPressed = isStandaloneButtonPressed(mask, tracker.originalButton)
	tracker.reversePressed = isStandaloneButtonPressed(mask, tracker.reverseButton)
	tracker.hasObservedOriginal = tracker.originalPressed
	tracker.hasRelease = false
	tracker.hasReversePress = false
}

func (tracker *standaloneTransitionTracker) completeSample() {
	tracker.latestSample = standaloneCounterStrafeSample{
		deltaTick:      tracker.reversePressTick - tracker.releaseTick,
		direction:      tracker.direction,
		completionTick: max(tracker.reversePressTick, tracker.releaseTick),
	}
	tracker.hasLatestSample = true
	tracker.hasObservedOriginal = false
	tracker.hasRelease = false
	tracker.hasReversePress = false
}

func (tracker *standaloneTransitionTracker) apply(mask uint64, tick int) {
	currentOriginalPressed := isStandaloneButtonPressed(mask, tracker.originalButton)
	currentReversePressed := isStandaloneButtonPressed(mask, tracker.reverseButton)

	if tracker.originalPressed && !currentOriginalPressed {
		if tracker.hasObservedOriginal {
			tracker.releaseTick = tick
			tracker.hasRelease = true
			if tracker.hasReversePress {
				tracker.completeSample()
			}
		}
	}

	if !tracker.reversePressed && currentReversePressed {
		if tracker.hasObservedOriginal {
			tracker.reversePressTick = tick
			tracker.hasReversePress = true
			if tracker.hasRelease {
				tracker.completeSample()
			}
		}
	}

	if tracker.reversePressed && !currentReversePressed {
		if tracker.hasReversePress && !tracker.hasRelease {
			tracker.hasReversePress = false
		}
	}

	if !tracker.originalPressed && currentOriginalPressed {
		tracker.hasObservedOriginal = true
		tracker.hasRelease = false
		tracker.hasReversePress = false
	}

	tracker.originalPressed = currentOriginalPressed
	tracker.reversePressed = currentReversePressed
}

func (tracker *standaloneComboTransitionTracker) resetObservedOriginalCombo() {
	tracker.hasObservedOriginalCombo = false
	tracker.hasHorizontalRelease = false
	tracker.hasHorizontalReversePress = false
	tracker.hasVerticalRelease = false
	tracker.hasVerticalReversePress = false
}

func (tracker *standaloneComboTransitionTracker) observeOriginalCombo() {
	tracker.hasObservedOriginalCombo = true
	tracker.hasHorizontalRelease = false
	tracker.hasHorizontalReversePress = false
	tracker.hasVerticalRelease = false
	tracker.hasVerticalReversePress = false
}

func (tracker *standaloneComboTransitionTracker) initialize(mask uint64) {
	tracker.originalHorizontalPressed = isStandaloneButtonPressed(mask, tracker.originalHorizontalButton)
	tracker.reverseHorizontalPressed = isStandaloneButtonPressed(mask, tracker.reverseHorizontalButton)
	tracker.originalVerticalPressed = isStandaloneButtonPressed(mask, tracker.originalVerticalButton)
	tracker.reverseVerticalPressed = isStandaloneButtonPressed(mask, tracker.reverseVerticalButton)
	tracker.hasObservedOriginalCombo = tracker.originalHorizontalPressed && tracker.originalVerticalPressed
	tracker.hasHorizontalRelease = false
	tracker.hasHorizontalReversePress = false
	tracker.hasVerticalRelease = false
	tracker.hasVerticalReversePress = false
}

func (tracker *standaloneComboTransitionTracker) completeSample() {
	tracker.latestSample = standaloneCounterStrafeComboSample{
		horizontalDeltaTick: tracker.horizontalReversePressTick - tracker.horizontalReleaseTick,
		verticalDeltaTick:   tracker.verticalReversePressTick - tracker.verticalReleaseTick,
		direction:           tracker.direction,
		completionTick:      max(max(tracker.horizontalReversePressTick, tracker.horizontalReleaseTick), max(tracker.verticalReversePressTick, tracker.verticalReleaseTick)),
	}
	tracker.hasLatestSample = true
	tracker.resetObservedOriginalCombo()
}

func (tracker *standaloneComboTransitionTracker) apply(mask uint64, tick int) {
	currentOriginalHorizontalPressed := isStandaloneButtonPressed(mask, tracker.originalHorizontalButton)
	currentReverseHorizontalPressed := isStandaloneButtonPressed(mask, tracker.reverseHorizontalButton)
	currentOriginalVerticalPressed := isStandaloneButtonPressed(mask, tracker.originalVerticalButton)
	currentReverseVerticalPressed := isStandaloneButtonPressed(mask, tracker.reverseVerticalButton)

	if tracker.hasObservedOriginalCombo {
		if tracker.originalHorizontalPressed && !currentOriginalHorizontalPressed {
			tracker.horizontalReleaseTick = tick
			tracker.hasHorizontalRelease = true
		}
		if !tracker.reverseHorizontalPressed && currentReverseHorizontalPressed {
			tracker.horizontalReversePressTick = tick
			tracker.hasHorizontalReversePress = true
		}
		if tracker.reverseHorizontalPressed && !currentReverseHorizontalPressed {
			if tracker.hasHorizontalReversePress && !tracker.hasHorizontalRelease {
				tracker.hasHorizontalReversePress = false
			}
		}

		if tracker.originalVerticalPressed && !currentOriginalVerticalPressed {
			tracker.verticalReleaseTick = tick
			tracker.hasVerticalRelease = true
		}
		if !tracker.reverseVerticalPressed && currentReverseVerticalPressed {
			tracker.verticalReversePressTick = tick
			tracker.hasVerticalReversePress = true
		}
		if tracker.reverseVerticalPressed && !currentReverseVerticalPressed {
			if tracker.hasVerticalReversePress && !tracker.hasVerticalRelease {
				tracker.hasVerticalReversePress = false
			}
		}
	}

	if currentOriginalHorizontalPressed && currentOriginalVerticalPressed && (!tracker.originalHorizontalPressed || !tracker.originalVerticalPressed) {
		tracker.observeOriginalCombo()
	}

	if tracker.hasObservedOriginalCombo && tracker.hasHorizontalRelease && tracker.hasHorizontalReversePress && tracker.hasVerticalRelease && tracker.hasVerticalReversePress {
		tracker.completeSample()
	}

	tracker.originalHorizontalPressed = currentOriginalHorizontalPressed
	tracker.reverseHorizontalPressed = currentReverseHorizontalPressed
	tracker.originalVerticalPressed = currentOriginalVerticalPressed
	tracker.reverseVerticalPressed = currentReverseVerticalPressed
}

func summarizeStandaloneCounterStrafeSamples(samples []standaloneCounterStrafeSample, direction standaloneCounterStrafeDirection) standaloneCounterStrafeStats {
	filteredSamples := make([]standaloneCounterStrafeSample, 0, len(samples))
	for _, sample := range samples {
		if direction != standaloneDirectionAll && sample.direction != direction {
			continue
		}
		filteredSamples = append(filteredSamples, sample)
	}

	if len(filteredSamples) == 0 {
		return standaloneCounterStrafeStats{}
	}

	var sum float64
	perfectCount := 0
	for _, sample := range filteredSamples {
		sum += float64(sample.deltaTick)
		if math.Abs(float64(sample.deltaTick)) <= standalonePerfectCounterStrafeDeltaWindow {
			perfectCount++
		}
	}

	average := sum / float64(len(filteredSamples))
	variance := 0.0
	for _, sample := range filteredSamples {
		delta := float64(sample.deltaTick) - average
		variance += delta * delta
	}
	variance /= float64(len(filteredSamples))

	return standaloneCounterStrafeStats{
		averageDeltaTick: average,
		stdDevTick:       math.Sqrt(variance),
		perfectRate:      float32(perfectCount) / float32(len(filteredSamples)) * 100,
	}
}

func standaloneAbsoluteDeltaTick(deltaTick int) int {
	if deltaTick < 0 {
		return -deltaTick
	}

	return deltaTick
}

func summarizeStandaloneCounterStrafeComboSamples(samples []standaloneCounterStrafeComboSample, direction standaloneCounterStrafeComboDirection) standaloneCounterStrafeStats {
	filteredSamples := make([]standaloneCounterStrafeComboSample, 0, len(samples))
	for _, sample := range samples {
		if direction != standaloneComboDirectionAll && sample.direction != direction {
			continue
		}
		filteredSamples = append(filteredSamples, sample)
	}

	if len(filteredSamples) == 0 {
		return standaloneCounterStrafeStats{}
	}

	var sum float64
	perfectCount := 0
	for _, sample := range filteredSamples {
		comboDeltaTick := max(standaloneAbsoluteDeltaTick(sample.horizontalDeltaTick), standaloneAbsoluteDeltaTick(sample.verticalDeltaTick))
		sum += float64(comboDeltaTick)
		if standaloneAbsoluteDeltaTick(sample.horizontalDeltaTick) <= standalonePerfectCounterStrafeComboDeltaWindow && standaloneAbsoluteDeltaTick(sample.verticalDeltaTick) <= standalonePerfectCounterStrafeComboDeltaWindow {
			perfectCount++
		}
	}

	average := sum / float64(len(filteredSamples))
	variance := 0.0
	for _, sample := range filteredSamples {
		comboDeltaTick := max(standaloneAbsoluteDeltaTick(sample.horizontalDeltaTick), standaloneAbsoluteDeltaTick(sample.verticalDeltaTick))
		delta := float64(comboDeltaTick) - average
		variance += delta * delta
	}
	variance /= float64(len(filteredSamples))

	return standaloneCounterStrafeStats{
		averageDeltaTick: average,
		stdDevTick:       math.Sqrt(variance),
		perfectRate:      float32(perfectCount) / float32(len(filteredSamples)) * 100,
	}
}

func assertFloat64Near(t *testing.T, playerName string, label string, got float64, want float64) {
	t.Helper()

	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("expected %s %s to be %v but got %v", playerName, label, want, got)
	}
}

func assertFloat32Near(t *testing.T, playerName string, label string, got float32, want float32) {
	t.Helper()

	if math.Abs(float64(got-want)) > 0.0001 {
		t.Fatalf("expected %s %s to be %v but got %v", playerName, label, want, got)
	}
}
