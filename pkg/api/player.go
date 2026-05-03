package api

import (
	"encoding/json"
	"math"
	"sort"

	"github.com/akiver/cs-demo-analyzer/internal/strings"
	"github.com/akiver/cs-demo-analyzer/pkg/api/constants"
	"github.com/akiver/cs-demo-analyzer/pkg/api/funData"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	st "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/sendtables"
)

type weaponInspectionData struct {
	tick          int
	cancelledTick int
}

type Player struct {
	match                          *Match
	SteamID64                      uint64       `json:"steamId"`
	UserID                         int          `json:"userId"` // +1 to get the player's slot
	Name                           string       `json:"name"`
	Score                          int          `json:"score"`
	Team                           *Team        `json:"team"`
	MvpCount                       int          `json:"mvpCount"`
	RankType                       int          `json:"rankType"`
	Rank                           int          `json:"rank"`
	OldRank                        int          `json:"oldRank"`
	WinCount                       int          `json:"winCount"`
	CrosshairShareCode             string       `json:"crosshairShareCode"`
	Color                          common.Color `json:"color"`
	InspectWeaponCount             int          `json:"inspectWeaponCount"`
	LeechValue                     int          `json:"leechValue"`
	FeedValue                      int          `json:"feedValue"`
	LeechCount                     int          `json:"leechCount"`
	FeedCount                      int          `json:"feedCount"`
	WastedUtilityValue             int          `json:"wastedUtilityValue"`
	lastWeaponInspection           weaponInspectionData
	hasCounterStrafeSampleCaches   bool
	counterStrafeSamplesCache      []counterStrafeSample
	counterStrafeComboSamplesCache []counterStrafeComboSample
	hasCounterStrafeSummaryCaches  bool
	counterStrafeSummariesByDirection map[counterStrafeDirection]counterStrafeSummary
	counterStrafeComboSummary        counterStrafeComboSummary
}

type PlayerAlias Player

type PlayerJSON struct {
	*PlayerAlias
	KillCount                   int     `json:"killCount"`
	DeathCount                  int     `json:"deathCount"`
	AssistCount                 int     `json:"assistCount"`
	KillDeathRatio              float32 `json:"killDeathRatio"`
	KAST                        float32 `json:"kast"`
	BombDefusedCount            int     `json:"bombDefusedCount"`
	BombPlantedCount            int     `json:"bombPlantedCount"`
	HealthDamage                int     `json:"healthDamage"`
	ArmorDamage                 int     `json:"armorDamage"`
	UtilityDamage               int     `json:"utilityDamage"`
	HeadshotCount               int     `json:"headshotCount"`
	HeadshotPercent             int     `json:"headshotPercent"`
	OneVsOneCount               int     `json:"oneVsOneCount"`
	OneVsOneWonCount            int     `json:"oneVsOneWonCount"`
	OneVsOneLostCount           int     `json:"oneVsOneLostCount"`
	OneVsTwoCount               int     `json:"oneVsTwoCount"`
	OneVsTwoWonCount            int     `json:"oneVsTwoWonCount"`
	OneVsTwoLostCount           int     `json:"oneVsTwoLostCount"`
	OneVsThreeCount             int     `json:"oneVsThreeCount"`
	OneVsThreeWonCount          int     `json:"oneVsThreeWonCount"`
	OneVsThreeLostCount         int     `json:"oneVsThreeLostCount"`
	OneVsFourCount              int     `json:"oneVsFourCount"`
	OneVsFourWonCount           int     `json:"oneVsFourWonCount"`
	OneVsFourLostCount          int     `json:"oneVsFourLostCount"`
	OneVsFiveCount              int     `json:"oneVsFiveCount"`
	OneVsFiveWonCount           int     `json:"oneVsFiveWonCount"`
	OneVsFiveLostCount          int     `json:"oneVsFiveLostCount"`
	HostageRescuedCount         int     `json:"hostageRescuedCount"`
	AverageKillPerRound         float32 `json:"averageKillPerRound"`
	AverageDeathPerRound        float32 `json:"averageDeathPerRound"`
	AverageDamagePerRound       float32 `json:"averageDamagePerRound"`
	UtilityDamagePerRound       float32 `json:"utilityDamagePerRound"`
	FirstKillCount              int     `json:"firstKillCount"`
	FirstDeathCount             int     `json:"firstDeathCount"`
	FirstTradeDeathCount        int     `json:"firstTradeDeathCount"`
	TradeDeathCount             int     `json:"tradeDeathCount"`
	TradeKillCount              int     `json:"tradeKillCount"`
	FirstTradeKillCount         int     `json:"firstTradeKillCount"`
	OneKillCount                int     `json:"oneKillCount"`
	TwoKillCount                int     `json:"twoKillCount"`
	ThreeKillCount              int     `json:"threeKillCount"`
	FourKillCount               int     `json:"fourKillCount"`
	FiveKillCount               int     `json:"fiveKillCount"`
	HltvRating                  float32 `json:"hltvRating"`
	HltvRating2                 float32 `json:"hltvRating2"`
	LeechValue                  int     `json:"leechValue"`
	FeedValue                   int     `json:"feedValue"`
	LeechCount                  int     `json:"leechCount"`
	FeedCount                   int     `json:"feedCount"`
	WastedUtilityValue          int     `json:"wastedUtilityValue"`
	UtilityDamageTaken          int     `json:"utilityDamageTaken"`
	WallbangDamageDealt         int     `json:"wallbangDamageDealt"`
	WallbangDamageTaken         int     `json:"wallbangDamageTaken"`
	TrueWallbangDamageTaken     int     `json:"trueWallbangDamageTaken"`
	TeamDamageTaken             int     `json:"teamDamageTaken"`
	FallDamageTaken             int     `json:"fallDamageTaken"`
	AirDamageTaken              int     `json:"airDamageTaken"`
	RunAndGunOrAirKilledByCount int     `json:"runAndGunOrAirKilledByCount"`
	ThroughSmokeKillCount       int     `json:"throughSmokeKillCount"`
	WallbangKillCount           int     `json:"wallbangKillCount"`
	AwpHoldKillCount            int     `json:"awpHoldKillCount"`
	AwpHoldDeathCount           int     `json:"awpHoldDeathCount"`
	TeamAttackDamage            int     `json:"teamAttackDamage"`
	TeamUtilityDamage           int     `json:"teamUtilityDamage"`
	TeamFlashDuration           float32 `json:"teamFlashDuration"`
	FirstShotCount              int     `json:"firstShotCount"`
	FirstShotHitCount           int     `json:"firstShotHitCount"`
	FirstShotAccuracy           float32 `json:"firstShotAccuracy"`
	CounterStrafingSuccessRate  float32 `json:"counterStrafingSuccessRate"`
	CounterStrafingAverageDeltaTick float64 `json:"counterStrafingAverageDeltaTick"`
	CounterStrafingDeltaStdDevTick float64 `json:"counterStrafingDeltaStdDevTick"`
	CounterStrafingPerfectRate float32 `json:"counterStrafingPerfectRate"`
	CounterStrafingAToDAverageDeltaTick float64 `json:"counterStrafingAToDAverageDeltaTick"`
	CounterStrafingAToDPerfectRate float32 `json:"counterStrafingAToDPerfectRate"`
	CounterStrafingDToAAverageDeltaTick float64 `json:"counterStrafingDToAAverageDeltaTick"`
	CounterStrafingDToAPerfectRate float32 `json:"counterStrafingDToAPerfectRate"`
	CounterStrafingWToSAverageDeltaTick float64 `json:"counterStrafingWToSAverageDeltaTick"`
	CounterStrafingWToSPerfectRate float32 `json:"counterStrafingWToSPerfectRate"`
	CounterStrafingSToWAverageDeltaTick float64 `json:"counterStrafingSToWAverageDeltaTick"`
	CounterStrafingSToWPerfectRate float32 `json:"counterStrafingSToWPerfectRate"`
	CounterStrafingComboAverageDeltaTick float64 `json:"counterStrafingComboAverageDeltaTick"`
	CounterStrafingComboDeltaStdDevTick float64 `json:"counterStrafingComboDeltaStdDevTick"`
	CounterStrafingComboPerfectRate float32 `json:"counterStrafingComboPerfectRate"`
}

type counterStrafeDirection int

const (
	counterStrafeDirectionAll counterStrafeDirection = iota
	counterStrafeDirectionAToD
	counterStrafeDirectionDToA
	counterStrafeDirectionWToS
	counterStrafeDirectionSToW
)

type counterStrafeComboDirection int

const (
	counterStrafeComboDirectionAll counterStrafeComboDirection = iota
	counterStrafeComboDirectionWAToSD
	counterStrafeComboDirectionWDToSA
	counterStrafeComboDirectionSAToWD
	counterStrafeComboDirectionSDToWA
)

const perfectCounterStrafeDeltaTickWindow = 3
const perfectCounterStrafeComboDeltaTickWindow = 10

type counterStrafeSample struct {
	deltaTick      int
	direction      counterStrafeDirection
	completionTick int
}

type counterStrafeShotAxisSamples struct {
	hasAToD bool
	aToD    counterStrafeSample
	hasDToA bool
	dToA    counterStrafeSample
	hasWToS bool
	wToS    counterStrafeSample
	hasSToW bool
	sToW    counterStrafeSample
}

type counterStrafeComboSample struct {
	horizontalDeltaTick int
	verticalDeltaTick   int
	direction           counterStrafeComboDirection
	completionTick      int
}

type counterStrafeSummary struct {
	average    float64
	stdDev     float64
	perfectRate float32
}

type counterStrafeComboSummary struct {
	average    float64
	stdDev     float64
	perfectRate float32
}

type counterStrafeSummaryAccumulator struct {
	count        int
	sum          float64
	sumOfSquares float64
	perfectCount int
}

type counterStrafeComboSummaryAccumulator struct {
	count        int
	sum          float64
	sumOfSquares float64
	perfectCount int
}

func (accumulator *counterStrafeSummaryAccumulator) addSample(sample counterStrafeSample) {
	deltaTick := float64(sample.deltaTick)
	accumulator.count++
	accumulator.sum += deltaTick
	accumulator.sumOfSquares += deltaTick * deltaTick
	if math.Abs(deltaTick) <= perfectCounterStrafeDeltaTickWindow {
		accumulator.perfectCount++
	}
}

func (accumulator counterStrafeSummaryAccumulator) summary() counterStrafeSummary {
	if accumulator.count == 0 {
		return counterStrafeSummary{}
	}

	average := accumulator.sum / float64(accumulator.count)
	variance := accumulator.sumOfSquares/float64(accumulator.count) - average*average
	if variance < 0 {
		variance = 0
	}

	return counterStrafeSummary{
		average: average,
		stdDev: math.Sqrt(variance),
		perfectRate: float32(accumulator.perfectCount) / float32(accumulator.count) * 100,
	}
}

func (accumulator *counterStrafeComboSummaryAccumulator) addSample(sample counterStrafeComboSample) {
	comboDeltaTick := float64(max(counterStrafeAbsoluteDeltaTick(sample.horizontalDeltaTick), counterStrafeAbsoluteDeltaTick(sample.verticalDeltaTick)))
	accumulator.count++
	accumulator.sum += comboDeltaTick
	accumulator.sumOfSquares += comboDeltaTick * comboDeltaTick
	if counterStrafeAbsoluteDeltaTick(sample.horizontalDeltaTick) <= perfectCounterStrafeComboDeltaTickWindow && counterStrafeAbsoluteDeltaTick(sample.verticalDeltaTick) <= perfectCounterStrafeComboDeltaTickWindow {
		accumulator.perfectCount++
	}
}

func (accumulator counterStrafeComboSummaryAccumulator) summary() counterStrafeComboSummary {
	if accumulator.count == 0 {
		return counterStrafeComboSummary{}
	}

	average := accumulator.sum / float64(accumulator.count)
	variance := accumulator.sumOfSquares/float64(accumulator.count) - average*average
	if variance < 0 {
		variance = 0
	}

	return counterStrafeComboSummary{
		average: average,
		stdDev: math.Sqrt(variance),
		perfectRate: float32(accumulator.perfectCount) / float32(accumulator.count) * 100,
	}
}

type counterStrafeTransitionTracker struct {
	direction           counterStrafeDirection
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
	latestSample        counterStrafeSample
}

type counterStrafeComboTransitionTracker struct {
	direction                  counterStrafeComboDirection
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
	latestSample               counterStrafeComboSample
}

func isFirstShotOfFiringSequence(shot *Shot) bool {
	return shot.RecoilIndex == 1
}

func newCounterStrafeTransitionTracker(direction counterStrafeDirection, originalButton common.ButtonBitMask, reverseButton common.ButtonBitMask) counterStrafeTransitionTracker {
	return counterStrafeTransitionTracker{
		direction:      direction,
		originalButton: originalButton,
		reverseButton:  reverseButton,
	}
}

func newCounterStrafeComboTransitionTracker(direction counterStrafeComboDirection, originalHorizontalButton common.ButtonBitMask, reverseHorizontalButton common.ButtonBitMask, originalVerticalButton common.ButtonBitMask, reverseVerticalButton common.ButtonBitMask) counterStrafeComboTransitionTracker {
	return counterStrafeComboTransitionTracker{
		direction:                direction,
		originalHorizontalButton: originalHorizontalButton,
		reverseHorizontalButton:  reverseHorizontalButton,
		originalVerticalButton:   originalVerticalButton,
		reverseVerticalButton:    reverseVerticalButton,
	}
}

func isButtonPressed(mask uint64, button common.ButtonBitMask) bool {
	return mask&uint64(button) != 0
}

func (tracker *counterStrafeTransitionTracker) initialize(mask uint64) {
	tracker.originalPressed = isButtonPressed(mask, tracker.originalButton)
	tracker.reversePressed = isButtonPressed(mask, tracker.reverseButton)
	tracker.hasObservedOriginal = tracker.originalPressed
	tracker.hasRelease = false
	tracker.hasReversePress = false
}

func (tracker *counterStrafeTransitionTracker) completeSample() {
	tracker.latestSample = counterStrafeSample{
		deltaTick:      tracker.reversePressTick - tracker.releaseTick,
		direction:      tracker.direction,
		completionTick: max(tracker.reversePressTick, tracker.releaseTick),
	}
	tracker.hasLatestSample = true
	tracker.hasObservedOriginal = false
	tracker.hasRelease = false
	tracker.hasReversePress = false
}

func (tracker *counterStrafeTransitionTracker) apply(mask uint64, tick int) {
	currentOriginalPressed := isButtonPressed(mask, tracker.originalButton)
	currentReversePressed := isButtonPressed(mask, tracker.reverseButton)

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

func (tracker *counterStrafeComboTransitionTracker) resetObservedOriginalCombo() {
	tracker.hasObservedOriginalCombo = false
	tracker.hasHorizontalRelease = false
	tracker.hasHorizontalReversePress = false
	tracker.hasVerticalRelease = false
	tracker.hasVerticalReversePress = false
}

func (tracker *counterStrafeComboTransitionTracker) observeOriginalCombo() {
	tracker.hasObservedOriginalCombo = true
	tracker.hasHorizontalRelease = false
	tracker.hasHorizontalReversePress = false
	tracker.hasVerticalRelease = false
	tracker.hasVerticalReversePress = false
}

func (tracker *counterStrafeComboTransitionTracker) initialize(mask uint64) {
	tracker.originalHorizontalPressed = isButtonPressed(mask, tracker.originalHorizontalButton)
	tracker.reverseHorizontalPressed = isButtonPressed(mask, tracker.reverseHorizontalButton)
	tracker.originalVerticalPressed = isButtonPressed(mask, tracker.originalVerticalButton)
	tracker.reverseVerticalPressed = isButtonPressed(mask, tracker.reverseVerticalButton)
	tracker.hasObservedOriginalCombo = tracker.originalHorizontalPressed && tracker.originalVerticalPressed
	tracker.hasHorizontalRelease = false
	tracker.hasHorizontalReversePress = false
	tracker.hasVerticalRelease = false
	tracker.hasVerticalReversePress = false
}

func (tracker *counterStrafeComboTransitionTracker) completeSample() {
	tracker.latestSample = counterStrafeComboSample{
		horizontalDeltaTick: tracker.horizontalReversePressTick - tracker.horizontalReleaseTick,
		verticalDeltaTick:   tracker.verticalReversePressTick - tracker.verticalReleaseTick,
		direction:           tracker.direction,
		completionTick:      max(max(tracker.horizontalReversePressTick, tracker.horizontalReleaseTick), max(tracker.verticalReversePressTick, tracker.verticalReleaseTick)),
	}
	tracker.hasLatestSample = true
	tracker.resetObservedOriginalCombo()
}

func (tracker *counterStrafeComboTransitionTracker) apply(mask uint64, tick int) {
	currentOriginalHorizontalPressed := isButtonPressed(mask, tracker.originalHorizontalButton)
	currentReverseHorizontalPressed := isButtonPressed(mask, tracker.reverseHorizontalButton)
	currentOriginalVerticalPressed := isButtonPressed(mask, tracker.originalVerticalButton)
	currentReverseVerticalPressed := isButtonPressed(mask, tracker.reverseVerticalButton)

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

func collectCounterStrafeAxisSamplesForShot(buttons []*funData.PlayerButtons, startExclusiveTick int, shotTick int) counterStrafeShotAxisSamples {
	startIndex := sort.Search(len(buttons), func(index int) bool {
		return buttons[index].Tick > startExclusiveTick
	})

	lastMask := uint64(0)
	if startIndex > 0 {
		lastMask = buttons[startIndex-1].Buttons
	}

	trackers := []counterStrafeTransitionTracker{
		newCounterStrafeTransitionTracker(counterStrafeDirectionAToD, common.ButtonMoveLeft, common.ButtonMoveRight),
		newCounterStrafeTransitionTracker(counterStrafeDirectionDToA, common.ButtonMoveRight, common.ButtonMoveLeft),
		newCounterStrafeTransitionTracker(counterStrafeDirectionWToS, common.ButtonForward, common.ButtonBack),
		newCounterStrafeTransitionTracker(counterStrafeDirectionSToW, common.ButtonBack, common.ButtonForward),
	}
	for index := range trackers {
		trackers[index].initialize(lastMask)
	}

	for index := startIndex; index < len(buttons); index++ {
		buttonState := buttons[index]
		if buttonState.Tick > shotTick {
			break
		}

		for trackerIndex := range trackers {
			trackers[trackerIndex].apply(buttonState.Buttons, buttonState.Tick)
		}
	}

	axisSamples := counterStrafeShotAxisSamples{}
	for _, tracker := range trackers {
		if !tracker.hasLatestSample {
			continue
		}

		sample := tracker.latestSample
		switch sample.direction {
		case counterStrafeDirectionAToD:
			axisSamples.hasAToD = true
			axisSamples.aToD = sample
		case counterStrafeDirectionDToA:
			axisSamples.hasDToA = true
			axisSamples.dToA = sample
		case counterStrafeDirectionWToS:
			axisSamples.hasWToS = true
			axisSamples.wToS = sample
		case counterStrafeDirectionSToW:
			axisSamples.hasSToW = true
			axisSamples.sToW = sample
		}
	}

	return axisSamples
}

func collectCounterStrafeSampleForShot(buttons []*funData.PlayerButtons, startExclusiveTick int, shotTick int) (*counterStrafeSample, bool) {
	axisSamples := collectCounterStrafeAxisSamplesForShot(buttons, startExclusiveTick, shotTick)
	samples := make([]counterStrafeSample, 0, 4)
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

func collectCounterStrafeComboSampleForShot(buttons []*funData.PlayerButtons, startExclusiveTick int, shotTick int) (*counterStrafeComboSample, bool) {
	startIndex := sort.Search(len(buttons), func(index int) bool {
		return buttons[index].Tick > startExclusiveTick
	})

	lastMask := uint64(0)
	if startIndex > 0 {
		lastMask = buttons[startIndex-1].Buttons
	}

	trackers := []counterStrafeComboTransitionTracker{
		newCounterStrafeComboTransitionTracker(counterStrafeComboDirectionWAToSD, common.ButtonMoveLeft, common.ButtonMoveRight, common.ButtonForward, common.ButtonBack),
		newCounterStrafeComboTransitionTracker(counterStrafeComboDirectionWDToSA, common.ButtonMoveRight, common.ButtonMoveLeft, common.ButtonForward, common.ButtonBack),
		newCounterStrafeComboTransitionTracker(counterStrafeComboDirectionSAToWD, common.ButtonMoveLeft, common.ButtonMoveRight, common.ButtonBack, common.ButtonForward),
		newCounterStrafeComboTransitionTracker(counterStrafeComboDirectionSDToWA, common.ButtonMoveRight, common.ButtonMoveLeft, common.ButtonBack, common.ButtonForward),
	}
	for index := range trackers {
		trackers[index].initialize(lastMask)
	}

	for index := startIndex; index < len(buttons); index++ {
		buttonState := buttons[index]
		if buttonState.Tick > shotTick {
			break
		}

		for trackerIndex := range trackers {
			trackers[trackerIndex].apply(buttonState.Buttons, buttonState.Tick)
		}
	}

	comboSamples := make([]counterStrafeComboSample, 0, len(trackers))
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

func counterStrafeAbsoluteDeltaTick(deltaTick int) int {
	if deltaTick < 0 {
		return -deltaTick
	}

	return deltaTick
}

func (player *Player) ensureCounterStrafeSampleCaches() {
	if player.hasCounterStrafeSampleCaches {
		return
	}

	player.match.ensureCounterStrafeRoundIndexes()

	oneDimensionalSamples := []counterStrafeSample{}
	comboSamples := []counterStrafeComboSample{}
	for _, round := range player.match.Rounds {
		key := roundPlayerKey{roundNumber: round.Number, steamID64: player.SteamID64}
		shots := player.match.counterStrafeShotsByRoundPlayer[key]
		if len(shots) == 0 {
			continue
		}

		buttons := player.match.counterStrafeButtonsByRoundPlayer[key]
		if len(buttons) == 0 {
			continue
		}

		previousShotTick := -1
		for _, shot := range shots {
			if sample, ok := collectCounterStrafeSampleForShot(buttons, previousShotTick, shot.Tick); ok {
				oneDimensionalSamples = append(oneDimensionalSamples, *sample)
			}
			if comboSample, ok := collectCounterStrafeComboSampleForShot(buttons, previousShotTick, shot.Tick); ok {
				comboSamples = append(comboSamples, *comboSample)
			}
			previousShotTick = shot.Tick
		}
	}

	player.counterStrafeSamplesCache = oneDimensionalSamples
	player.counterStrafeComboSamplesCache = comboSamples
	player.hasCounterStrafeSampleCaches = true
}

func (player *Player) ensureCounterStrafeSummaryCaches() {
	if player.hasCounterStrafeSummaryCaches {
		return
	}

	player.ensureCounterStrafeSampleCaches()

	accumulatorsByDirection := map[counterStrafeDirection]*counterStrafeSummaryAccumulator{
		counterStrafeDirectionAll:  &counterStrafeSummaryAccumulator{},
		counterStrafeDirectionAToD: &counterStrafeSummaryAccumulator{},
		counterStrafeDirectionDToA: &counterStrafeSummaryAccumulator{},
		counterStrafeDirectionWToS: &counterStrafeSummaryAccumulator{},
		counterStrafeDirectionSToW: &counterStrafeSummaryAccumulator{},
	}

	for _, sample := range player.counterStrafeSamplesCache {
		accumulatorsByDirection[counterStrafeDirectionAll].addSample(sample)
		accumulatorsByDirection[sample.direction].addSample(sample)
	}

	player.counterStrafeSummariesByDirection = make(map[counterStrafeDirection]counterStrafeSummary, len(accumulatorsByDirection))
	for direction, accumulator := range accumulatorsByDirection {
		player.counterStrafeSummariesByDirection[direction] = accumulator.summary()
	}

	comboAccumulator := counterStrafeComboSummaryAccumulator{}
	for _, sample := range player.counterStrafeComboSamplesCache {
		comboAccumulator.addSample(sample)
	}
	player.counterStrafeComboSummary = comboAccumulator.summary()

	player.hasCounterStrafeSummaryCaches = true
}

func (player *Player) counterStrafeSummary(direction counterStrafeDirection) counterStrafeSummary {
	player.ensureCounterStrafeSummaryCaches()
	return player.counterStrafeSummariesByDirection[direction]
}

func (player *Player) MarshalJSON() ([]byte, error) {
	return json.Marshal(PlayerJSON{
		PlayerAlias:                 (*PlayerAlias)(player),
		KillCount:                   player.KillCount(),
		DeathCount:                  player.DeathCount(),
		AssistCount:                 player.AssistCount(),
		KillDeathRatio:              player.KillDeathRatio(),
		KAST:                        player.KAST(),
		BombDefusedCount:            player.BombDefusedCount(),
		BombPlantedCount:            player.BombPlantedCount(),
		HealthDamage:                player.HealthDamage(),
		ArmorDamage:                 player.ArmorDamage(),
		UtilityDamage:               player.UtilityDamage(),
		HeadshotCount:               player.HeadshotCount(),
		HeadshotPercent:             player.HeadshotPercent(),
		OneVsOneCount:               player.OneVsOneCount(),
		OneVsOneWonCount:            player.OneVsOneWonCount(),
		OneVsOneLostCount:           player.OneVsOneLostCount(),
		OneVsTwoCount:               player.OneVsTwoCount(),
		OneVsTwoWonCount:            player.OneVsTwoWonCount(),
		OneVsTwoLostCount:           player.OneVsTwoLostCount(),
		OneVsThreeCount:             player.OneVsThreeCount(),
		OneVsThreeWonCount:          player.OneVsThreeWonCount(),
		OneVsThreeLostCount:         player.OneVsThreeLostCount(),
		OneVsFourCount:              player.OneVsFourCount(),
		OneVsFourWonCount:           player.OneVsFourWonCount(),
		OneVsFourLostCount:          player.OneVsFourLostCount(),
		OneVsFiveCount:              player.OneVsFiveCount(),
		OneVsFiveWonCount:           player.OneVsFiveWonCount(),
		OneVsFiveLostCount:          player.OneVsFiveLostCount(),
		HostageRescuedCount:         player.HostageRescuedCount(),
		AverageKillPerRound:         player.AverageKillPerRound(),
		AverageDeathPerRound:        player.AverageDeathPerRound(),
		AverageDamagePerRound:       player.AverageDamagePerRound(),
		UtilityDamagePerRound:       player.UtilityDamagePerRound(),
		FirstKillCount:              player.FirstKillCount(),
		FirstDeathCount:             player.FirstDeathCount(),
		FirstTradeDeathCount:        player.FirstTradeDeathCount(),
		TradeDeathCount:             player.TradeDeathCount(),
		TradeKillCount:              player.TradeKillCount(),
		FirstTradeKillCount:         player.FirstTradeKillCount(),
		OneKillCount:                player.OneKillCount(),
		TwoKillCount:                player.TwoKillCount(),
		ThreeKillCount:              player.ThreeKillCount(),
		FourKillCount:               player.FourKillCount(),
		FiveKillCount:               player.FiveKillCount(),
		HltvRating2:                 player.HltvRating2(),
		HltvRating:                  player.HltvRating(),
		LeechValue:                  player.LeechValue,
		FeedValue:                   player.FeedValue,
		LeechCount:                  player.LeechCount,
		FeedCount:                   player.FeedCount,
		WastedUtilityValue:          player.WastedUtilityValue,
		UtilityDamageTaken:          player.UtilityDamageTaken(),
		WallbangDamageDealt:         player.WallbangDamageDealt(),
		WallbangDamageTaken:         player.WallbangDamageTaken(),
		TrueWallbangDamageTaken:     player.TrueWallbangDamageTaken(),
		TeamDamageTaken:             player.TeamDamageTaken(),
		FallDamageTaken:             player.FallDamageTaken(),
		AirDamageTaken:              player.AirDamageTaken(),
		TeamAttackDamage:            player.TeamAttackDamage(),
		TeamUtilityDamage:           player.TeamUtilityDamage(),
		TeamFlashDuration:           player.TeamFlashDuration(),
		FirstShotCount:              player.FirstShotCount(),
		FirstShotHitCount:           player.FirstShotHitCount(),
		FirstShotAccuracy:           player.FirstShotAccuracy(),
		RunAndGunOrAirKilledByCount: player.RunAndGunOrAirKilledByCount(),
		ThroughSmokeKillCount:       player.ThroughSmokeKillCount(),
		WallbangKillCount:           player.WallbangKillCount(),
		CounterStrafingSuccessRate:  player.CounterStrafingSuccessRate(),
		CounterStrafingAverageDeltaTick: player.CounterStrafingAverageDeltaTick(),
		CounterStrafingDeltaStdDevTick: player.CounterStrafingDeltaStdDevTick(),
		CounterStrafingPerfectRate: player.CounterStrafingPerfectRate(),
		CounterStrafingAToDAverageDeltaTick: player.CounterStrafingAToDAverageDeltaTick(),
		CounterStrafingAToDPerfectRate: player.CounterStrafingAToDPerfectRate(),
		CounterStrafingDToAAverageDeltaTick: player.CounterStrafingDToAAverageDeltaTick(),
		CounterStrafingDToAPerfectRate: player.CounterStrafingDToAPerfectRate(),
		CounterStrafingWToSAverageDeltaTick: player.CounterStrafingWToSAverageDeltaTick(),
		CounterStrafingWToSPerfectRate: player.CounterStrafingWToSPerfectRate(),
		CounterStrafingSToWAverageDeltaTick: player.CounterStrafingSToWAverageDeltaTick(),
		CounterStrafingSToWPerfectRate: player.CounterStrafingSToWPerfectRate(),
		CounterStrafingComboAverageDeltaTick: player.CounterStrafingComboAverageDeltaTick(),
		CounterStrafingComboDeltaStdDevTick: player.CounterStrafingComboDeltaStdDevTick(),
		CounterStrafingComboPerfectRate: player.CounterStrafingComboPerfectRate(),
		AwpHoldKillCount:            player.AwpHoldKillCount(),
		AwpHoldDeathCount:           player.AwpHoldDeathCount(),
	})
}

func (player *Player) UtilityDamageTaken() int {
	var utilityDamageTaken int
	for _, damage := range player.match.Damages {
		if damage.VictimSteamID64 == player.SteamID64 && damage.IsGrenadeWeapon() {
			utilityDamageTaken += damage.HealthDamage
		}
	}
	return utilityDamageTaken
}

func (player *Player) TrueWallbangDamageTaken() int {
	var trueWallbangDamageTaken int
	for _, damage := range player.match.Damages {
		// True wallbang signal: only when BulletDamage <-> PlayerHurt same-frame correlation reports penetrations.
		if damage.VictimSteamID64 == player.SteamID64 && damage.isWallbang() {
			trueWallbangDamageTaken += damage.HealthDamage
		}
	}

	return trueWallbangDamageTaken
}

func (player *Player) WallbangDamageDealt() int {
	var wallbangDamageDealt int
	for _, damage := range player.match.Damages {
		// Outward wallbang aggregate: includes heuristic approximation because direct non-lethal wallbang extraction is
		// often unavailable from parser signals.
		if damage.AttackerSteamID64 == player.SteamID64 && damage.IsWallbang {
			wallbangDamageDealt += damage.HealthDamage
		}
	}

	return wallbangDamageDealt
}

func (player *Player) WallbangDamageTaken() int {
	var wallbangDamageTaken int
	for _, damage := range player.match.Damages {
		// Outward wallbang aggregate: includes heuristic approximation because direct non-lethal wallbang extraction is
		// often unavailable from parser signals.
		if damage.VictimSteamID64 == player.SteamID64 && damage.IsWallbang {
			wallbangDamageTaken += damage.HealthDamage
		}
	}

	return wallbangDamageTaken
}

func (player *Player) TeamDamageTaken() int {
	var teamDamageTaken int
	for _, damage := range player.match.Damages {
		if damage.VictimSteamID64 == player.SteamID64 && damage.AttackerSteamID64 != 0 && damage.AttackerSide == damage.VictimSide && damage.AttackerSteamID64 != damage.VictimSteamID64 {
			teamDamageTaken += damage.HealthDamage
		}
	}
	return teamDamageTaken
}

func (player *Player) FallDamageTaken() int {
	var fallDamageTaken int
	for _, damage := range player.match.Damages {
		if damage.VictimSteamID64 == player.SteamID64 && (damage.isFallDamage || (!damage.isFallDamage && damage.WeaponName == constants.WeaponWorld)) {
			fallDamageTaken += damage.HealthDamage
		}
	}

	return fallDamageTaken
}

func (player *Player) AirDamageTaken() int {
	var airDamageTaken int
	for _, damage := range player.match.Damages {
		if damage.VictimSteamID64 == player.SteamID64 && damage.IsAttackerAirborne {
			airDamageTaken += damage.HealthDamage
		}
	}
	return airDamageTaken
}

func (player *Player) ThroughSmokeKillCount() int {
	var count int
	for _, kill := range player.Deaths() {
		if kill.KillerSteamID64 != 0 && kill.KillerSteamID64 != player.SteamID64 && kill.IsThroughSmoke {
			count++
		}
	}
	return count
}

func (player *Player) WallbangKillCount() int {
	var count int
	for _, kill := range player.Deaths() {
		if kill.KillerSteamID64 != 0 && kill.KillerSteamID64 != player.SteamID64 && kill.PenetratedObjects > 0 {
			count++
		}
	}
	return count
}

func (player *Player) RunAndGunOrAirKilledByCount() int {
	var count int
	for _, kill := range player.Deaths() {
		if kill.KillerSteamID64 != 0 && kill.KillerSteamID64 != player.SteamID64 {
			if kill.IsKillerRunning || kill.IsKillerAirborne {
				count++
			}
		}
	}
	return count
}

func (player *Player) AwpHoldKillCount() int {
	var count int
	for _, awpHoldDeath := range player.match.AwpHoldDeaths {
		if awpHoldDeath.KillerSteamID64 == player.SteamID64 {
			count++
		}
	}
	return count
}

func (player *Player) AwpHoldDeathCount() int {
	var count int
	for _, awpHoldDeath := range player.match.AwpHoldDeaths {
		if awpHoldDeath.VictimSteamID64 == player.SteamID64 {
			count++
		}
	}
	return count
}

func (player *Player) TeamName() string {
	return player.Team.Name
}

func (player *Player) String() string {
	return player.Name
}

// This returns the percentage of rounds in which the player either had a kill, assist, survived or was traded.
func (player *Player) KAST() float32 {
	kastPerRound := make(map[int]bool)
	for _, round := range player.match.Rounds {
		kastPerRound[round.Number] = false
		playerSurvived := true

		for _, kill := range player.match.Kills {
			if round.Number != kill.RoundNumber {
				continue
			}

			isTeamKill := kill.KillerSide == kill.VictimSide
			if isTeamKill {
				continue
			}

			if kill.AssisterSteamID64 == player.SteamID64 {
				kastPerRound[round.Number] = true
				continue
			}

			if kill.KillerSteamID64 == player.SteamID64 && kill.VictimSteamID64 != player.SteamID64 {
				kastPerRound[round.Number] = true
				continue
			}

			if kill.VictimSteamID64 == player.SteamID64 {
				playerSurvived = false
				if kill.IsTradeDeath {
					kastPerRound[round.Number] = true
				}
			}
		}
		if playerSurvived {
			kastPerRound[round.Number] = true
		}
	}

	kastEventCount := 0
	for _, hasKASTEvent := range kastPerRound {
		if hasKASTEvent {
			kastEventCount++
		}
	}

	if len(kastPerRound) > 0 {
		return float32(kastEventCount) / float32(len(kastPerRound)) * 100
	}

	return 0
}

func (player *Player) BombPlantedCount() int {
	var bombPlantedCount int
	for _, bombPlanted := range player.match.BombsPlanted {
		if bombPlanted.PlanterSteamID64 == player.SteamID64 && !bombPlanted.IsPlayerControllingBot {
			bombPlantedCount++
		}
	}

	return bombPlantedCount
}

func (player *Player) BombDefusedCount() int {
	var bombDefusedCount int
	for _, bombDefused := range player.match.BombsDefused {
		if bombDefused.DefuserSteamID64 == player.SteamID64 && !bombDefused.IsPlayerControllingBot {
			bombDefusedCount++
		}
	}

	return bombDefusedCount
}

func (player *Player) Clutches() []*Clutch {
	clutches := []*Clutch{}
	for _, clutch := range player.match.Clutches {
		if clutch.ClutcherSteamID64 == player.SteamID64 {
			clutches = append(clutches, clutch)
		}
	}

	return clutches
}

func (player *Player) OneVsOneWonCount() int {
	return player.oneVsXClutchesWonCount(1)
}

func (player *Player) OneVsOneLostCount() int {
	return player.oneVsXClutchesLostCount(1)
}

func (player *Player) OneVsOneCount() int {
	clutches := player.oneVsXClutches(1)
	return len(clutches)
}

func (player *Player) OneVsTwoWonCount() int {
	return player.oneVsXClutchesWonCount(2)
}

func (player *Player) OneVsTwoLostCount() int {
	return player.oneVsXClutchesLostCount(2)
}

func (player *Player) OneVsTwoCount() int {
	clutches := player.oneVsXClutches(2)
	return len(clutches)
}

func (player *Player) OneVsThreeWonCount() int {
	return player.oneVsXClutchesWonCount(3)
}

func (player *Player) OneVsThreeLostCount() int {
	return player.oneVsXClutchesLostCount(3)
}

func (player *Player) OneVsThreeCount() int {
	clutches := player.oneVsXClutches(3)
	return len(clutches)
}

func (player *Player) OneVsFourWonCount() int {
	return player.oneVsXClutchesWonCount(4)
}

func (player *Player) OneVsFourLostCount() int {
	return player.oneVsXClutchesLostCount(4)
}

func (player *Player) OneVsFourCount() int {
	clutches := player.oneVsXClutches(4)
	return len(clutches)
}

func (player *Player) OneVsFiveWonCount() int {
	return player.oneVsXClutchesWonCount(5)
}

func (player *Player) OneVsFiveLostCount() int {
	return player.oneVsXClutchesLostCount(5)
}

func (player *Player) OneVsFiveCount() int {
	clutches := player.oneVsXClutches(5)
	return len(clutches)
}

func (player *Player) HostageRescuedCount() int {
	var hostageRescuedCount int
	for _, hostageRescued := range player.match.HostageRescued {
		if hostageRescued.PlayerSteamID64 == player.SteamID64 && !hostageRescued.IsPlayerControllingBot {
			hostageRescuedCount++
		}
	}

	return hostageRescuedCount
}

func (player *Player) kills() []*Kill {
	var kills []*Kill = make([]*Kill, 0)
	for _, kill := range player.match.Kills {
		if kill.KillerSteamID64 == player.SteamID64 && !kill.IsKillerControllingBot {
			kills = append(kills, kill)
		}
	}

	return kills
}

func (player *Player) Deaths() []*Kill {
	var deaths []*Kill = make([]*Kill, 0)
	for _, death := range player.match.Kills {
		if death.VictimSteamID64 == player.SteamID64 && !death.IsVictimControllingBot {
			deaths = append(deaths, death)
		}
	}

	return deaths
}

func (player *Player) KillCount() int {
	var killCount int
	for _, kill := range player.match.Kills {
		if kill.KillerSteamID64 == player.SteamID64 {
			if kill.IsKillerControllingBot {
				continue
			}

			if kill.IsSuicide() {
				// CSGO decreases player kill count on disconnection caused by network issue or by a vote kick, we don't
				isClientDisconnection := kill.WeaponName == constants.WeaponWorld
				if !isClientDisconnection {
					killCount--
				}
				continue
			}

			if kill.IsTeamKill() {
				killCount--
				continue
			}

			killCount++
		} else if kill.VictimSteamID64 == player.SteamID64 {
			if kill.IsVictimControllingBot {
				continue
			}

			isSuicide := kill.KillerSteamID64 == 0 && kill.WeaponName == constants.WeaponWorld
			if isSuicide {
				killCount--
			}
		}
	}

	return killCount
}

func (player *Player) DeathCount() int {
	var deathCount int
	for _, kill := range player.Deaths() {
		if kill.IsSuicide() {
			isClientDisconnection := kill.WeaponName == constants.WeaponWorld
			if isClientDisconnection {
				continue
			}
		}

		deathCount++
	}

	return deathCount
}

func (player *Player) AssistCount() int {
	var assistCount int
	for _, kill := range player.match.Kills {
		if kill.AssisterSteamID64 == player.SteamID64 && !kill.IsAssisterControllingBot && kill.AssisterSide != kill.VictimSide {
			assistCount++
		}
	}

	return assistCount
}

func (player *Player) HeadshotCount() int {
	var headshotCount int
	for _, kill := range player.kills() {
		if !kill.IsHeadshot || kill.IsSuicide() || kill.IsTeamKill() {
			continue
		}

		headshotCount++
	}

	return headshotCount
}

func (player *Player) FirstKillCount() int {
	var firstKillCount int
	for _, round := range player.match.Rounds {
		var killsInRound []*Kill = make([]*Kill, 0)
		for _, kill := range player.match.Kills {
			if kill.RoundNumber != round.Number {
				continue
			}
			killsInRound = append(killsInRound, kill)
		}

		for _, kill := range killsInRound {
			if kill.IsKillerControllingBot {
				continue
			}

			isSuicide := kill.KillerSteamID64 == kill.VictimSteamID64
			if isSuicide {
				continue
			}

			isTeamKill := kill.KillerSide == kill.VictimSide
			if isTeamKill {
				continue
			}

			if kill.KillerSteamID64 == player.SteamID64 {
				firstKillCount++
			}
			break
		}
	}

	return firstKillCount
}

func (player *Player) FirstDeathCount() int {
	var firstDeathCount int
	for _, round := range player.match.Rounds {
		var killsInRound []*Kill = make([]*Kill, 0)
		for _, kill := range player.match.Kills {
			if kill.RoundNumber != round.Number {
				continue
			}
			killsInRound = append(killsInRound, kill)
		}

		for _, kill := range killsInRound {
			if kill.IsKillerControllingBot || kill.IsSuicide() || kill.IsTeamKill() {
				continue
			}

			if kill.VictimSteamID64 == player.SteamID64 {
				firstDeathCount++
			}
			break
		}
	}

	return firstDeathCount
}

func (player *Player) FirstTradeDeathCount() int {
	var firstTradeDeathCount int
	for _, kills := range player.match.KillsByRound() {
		for _, kill := range kills {
			if kill.IsVictimControllingBot || kill.IsSuicide() || kill.IsTeamKill() {
				continue
			}

			if kill.VictimSteamID64 == player.SteamID64 && kill.IsTradeDeath {
				firstTradeDeathCount++
			}

			break
		}
	}

	return firstTradeDeathCount
}

func (player *Player) FirstTradeKillCount() int {
	var firstTradeKillCount int
	for _, kills := range player.match.KillsByRound() {
		for _, kill := range kills {
			if kill.IsKillerControllingBot || kill.IsSuicide() || kill.IsTeamKill() {
				continue
			}

			if kill.KillerSteamID64 == player.SteamID64 && kill.IsTradeKill {
				firstTradeKillCount++
			}

			break
		}
	}

	return firstTradeKillCount
}

func (player *Player) TradeDeathCount() int {
	var tradeDeathCount int
	for _, kill := range player.Deaths() {
		if !kill.IsTradeDeath || kill.IsSuicide() || kill.IsTeamKill() {
			continue
		}

		tradeDeathCount++
	}

	return tradeDeathCount
}

func (player *Player) TradeKillCount() int {
	var tradeKillCount int
	for _, kill := range player.kills() {
		if !kill.IsTradeKill || kill.IsSuicide() || kill.IsTeamKill() {
			continue
		}

		tradeKillCount++
	}

	return tradeKillCount
}

func (player *Player) HealthDamage() int {
	var healthDamage int
	for _, damage := range player.match.Damages {
		if damage.isValidPlayerDamageEvent(player) {
			healthDamage += damage.HealthDamage
		}
	}

	return healthDamage
}

func (player *Player) ArmorDamage() int {
	var armorDamage int
	for _, damage := range player.match.Damages {
		if damage.isValidPlayerDamageEvent(player) {
			armorDamage += damage.ArmorDamage
		}
	}

	return armorDamage
}

func (player *Player) UtilityDamage() int {
	var utilityDamage int
	for _, damage := range player.match.Damages {
		if damage.isValidPlayerDamageEvent(player) && damage.IsGrenadeWeapon() {
			utilityDamage += damage.HealthDamage
		}
	}

	return utilityDamage
}

func (player *Player) HeadshotPercent() int {
	killCount := player.KillCount()
	if killCount > 0 {
		return 100 * player.HeadshotCount() / killCount
	}

	return 0
}

func (player *Player) KillDeathRatio() float32 {
	killCount := player.KillCount()
	if killCount <= 0 {
		return 0
	}

	deathCount := player.DeathCount()
	if deathCount > 0 {
		return float32(killCount) / float32(deathCount)
	}

	return float32(killCount)
}

func (player *Player) AverageKillPerRound() float32 {
	killCount := player.KillCount()
	roundCount := player.roundCount()
	if killCount <= 0 || roundCount <= 0 {
		return 0
	}

	return float32(killCount) / float32(roundCount)
}

func (player *Player) AverageAssistPerRound() float32 {
	assistCount := player.AssistCount()
	roundCount := player.roundCount()
	if assistCount <= 0 || roundCount <= 0 {
		return 0
	}

	return float32(assistCount) / float32(roundCount)
}

func (player *Player) AverageDeathPerRound() float32 {
	deathCount := player.DeathCount()
	roundCount := player.roundCount()
	if deathCount <= 0 || roundCount <= 0 {
		return 0
	}

	return float32(deathCount) / float32(roundCount)
}

func (player *Player) AverageDamagePerRound() float32 {
	roundCount := player.roundCount()
	if roundCount > 0 {
		return float32(player.HealthDamage()) / float32(roundCount)
	}

	return 0
}

func (player *Player) UtilityDamagePerRound() float32 {
	roundCount := player.roundCount()
	if roundCount > 0 {
		return float32(player.UtilityDamage()) / float32(roundCount)
	}

	return 0
}

func (player *Player) FirstShotCount() int {
	count := 0
	for _, shot := range player.match.Shots {
		if shot.PlayerSteamID64 != player.SteamID64 || shot.IsPlayerControllingBot || !isFirstShotOfFiringSequence(shot) {
			continue
		}

		count++
	}

	return count
}

func (player *Player) FirstShotHitCount() int {
	shots := player.shotsByWeaponID()
	if len(shots) == 0 {
		return 0
	}

	hitShots := make(map[*Shot]struct{})
	for _, damage := range player.match.Damages {
		if !damage.isValidPlayerDamageEvent(player) {
			continue
		}

		matchedShot := nearestPriorShotForDamage(damage, shots)
		if matchedShot == nil || !isFirstShotOfFiringSequence(matchedShot) {
			continue
		}

		hitShots[matchedShot] = struct{}{}
	}

	return len(hitShots)
}

func (player *Player) FirstShotAccuracy() float32 {
	firstShotCount := player.FirstShotCount()
	if firstShotCount == 0 {
		return 0
	}

	return float32(player.FirstShotHitCount()) / float32(firstShotCount) * 100
}

func (player *Player) CounterStrafingSuccessRate() float32 {
	firstShotCount := 0
	counterStrafingSuccessCount := 0

	for _, shot := range player.match.Shots {
		if shot.PlayerSteamID64 != player.SteamID64 || shot.IsPlayerControllingBot || !isFirstShotOfFiringSequence(shot) {
			continue
		}

		firstShotCount++
		if !shot.IsPlayerRunning {
			counterStrafingSuccessCount++
		}
	}

	if firstShotCount == 0 {
		return 0
	}

	return float32(counterStrafingSuccessCount) / float32(firstShotCount) * 100
}

func (player *Player) shotsByWeaponID() map[string][]shotIndexEntry {
	shotsByWeaponID := make(map[string][]shotIndexEntry)
	for _, shot := range player.match.Shots {
		if shot == nil || shot.PlayerSteamID64 != player.SteamID64 || shot.IsPlayerControllingBot || shot.WeaponID == "" {
			continue
		}

		shotsByWeaponID[shot.WeaponID] = append(shotsByWeaponID[shot.WeaponID], shotIndexEntry{
			frame: shot.Frame,
			tick:  shot.Tick,
			shot:  shot,
		})
	}

	for weaponID := range shotsByWeaponID {
		sort.Slice(shotsByWeaponID[weaponID], func(i int, j int) bool {
			if shotsByWeaponID[weaponID][i].frame == shotsByWeaponID[weaponID][j].frame {
				return shotsByWeaponID[weaponID][i].tick < shotsByWeaponID[weaponID][j].tick
			}

			return shotsByWeaponID[weaponID][i].frame < shotsByWeaponID[weaponID][j].frame
		})
	}

	return shotsByWeaponID
}

func nearestPriorShotForDamage(damage *Damage, shotsByWeaponID map[string][]shotIndexEntry) *Shot {
	entries, exists := shotsByWeaponID[damage.WeaponUniqueID]
	if !exists || len(entries) == 0 {
		return nil
	}

	var best *Shot
	bestFrameDelta := math.MaxInt
	bestTickDelta := math.MaxInt

	for _, entry := range entries {
		if entry.shot == nil || entry.shot.RoundNumber != damage.RoundNumber {
			continue
		}
		if entry.frame > damage.Frame {
			continue
		}
		if entry.frame == damage.Frame && entry.tick > damage.Tick {
			continue
		}

		frameDelta := absInt(entry.frame - damage.Frame)
		if frameDelta > constants.HeuristicDamageAttributionMaxShotFrameDistance {
			continue
		}

		tickDelta := absInt(entry.tick - damage.Tick)
		if frameDelta < bestFrameDelta || (frameDelta == bestFrameDelta && tickDelta < bestTickDelta) {
			best = entry.shot
			bestFrameDelta = frameDelta
			bestTickDelta = tickDelta
		}
	}

	return best
}

func (player *Player) CounterStrafingAverageDeltaTick() float64 {
	return player.counterStrafeSummary(counterStrafeDirectionAll).average
}

func (player *Player) CounterStrafingDeltaStdDevTick() float64 {
	return player.counterStrafeSummary(counterStrafeDirectionAll).stdDev
}

func (player *Player) CounterStrafingPerfectRate() float32 {
	return player.counterStrafeSummary(counterStrafeDirectionAll).perfectRate
}

func (player *Player) CounterStrafingAToDAverageDeltaTick() float64 {
	return player.counterStrafeSummary(counterStrafeDirectionAToD).average
}

func (player *Player) CounterStrafingAToDPerfectRate() float32 {
	return player.counterStrafeSummary(counterStrafeDirectionAToD).perfectRate
}

func (player *Player) CounterStrafingDToAAverageDeltaTick() float64 {
	return player.counterStrafeSummary(counterStrafeDirectionDToA).average
}

func (player *Player) CounterStrafingDToAPerfectRate() float32 {
	return player.counterStrafeSummary(counterStrafeDirectionDToA).perfectRate
}

func (player *Player) CounterStrafingWToSAverageDeltaTick() float64 {
	return player.counterStrafeSummary(counterStrafeDirectionWToS).average
}

func (player *Player) CounterStrafingWToSPerfectRate() float32 {
	return player.counterStrafeSummary(counterStrafeDirectionWToS).perfectRate
}

func (player *Player) CounterStrafingSToWAverageDeltaTick() float64 {
	return player.counterStrafeSummary(counterStrafeDirectionSToW).average
}

func (player *Player) CounterStrafingSToWPerfectRate() float32 {
	return player.counterStrafeSummary(counterStrafeDirectionSToW).perfectRate
}

func (player *Player) CounterStrafingComboAverageDeltaTick() float64 {
	player.ensureCounterStrafeSummaryCaches()
	return player.counterStrafeComboSummary.average
}

func (player *Player) CounterStrafingComboDeltaStdDevTick() float64 {
	player.ensureCounterStrafeSummaryCaches()
	return player.counterStrafeComboSummary.stdDev
}

func (player *Player) CounterStrafingComboPerfectRate() float32 {
	player.ensureCounterStrafeSummaryCaches()
	return player.counterStrafeComboSummary.perfectRate
}

func (player *Player) OneKillCount() int {
	return player.getXKillCount(1)
}

func (player *Player) TwoKillCount() int {
	return player.getXKillCount(2)
}

func (player *Player) ThreeKillCount() int {
	return player.getXKillCount(3)
}

func (player *Player) FourKillCount() int {
	return player.getXKillCount(4)
}

func (player *Player) FiveKillCount() int {
	return player.getXKillCount(5)
}

// This returns the "impact" as described in the following blog post.
// https://flashed.gg/posts/reverse-engineering-hltv-rating/
// 2.13*KPR + 0.42*Assist per Round -0.41 ≈ impact
func (player *Player) impact() float32 {
	return 2.13*player.AverageKillPerRound() + 0.42*player.AverageAssistPerRound() + -0.41
}

// This returns the player's HLTV rating 2.0.
// https://flashed.gg/posts/reverse-engineering-hltv-rating/
// 0.0073*KAST + 0.3591*KPR + -0.5329*DPR + 0.2372*Impact + 0.0032*ADR + 0.1587 ≈ Rating 2.0
func (player *Player) HltvRating2() float32 {
	rating := 0.0073*player.KAST() + 0.3591*player.AverageKillPerRound() + -0.5329*player.AverageDeathPerRound() + 0.2372*player.impact() + 0.0032*float32(player.AverageDamagePerRound()) + 0.1587

	if rating < 0 {
		return 0
	}

	return rating
}

// This returns the player's HLTV rating 1.0.
// Formula: https://web.archive.org/web/20170427062206/http://www.hltv.org/?pageid=242&eventid=0
func (player *Player) HltvRating() float32 {
	roundCount := float32(player.roundCount())
	if roundCount == 0 {
		return 0
	}

	killRating := player.AverageKillPerRound() / 0.679
	survivalRating := (roundCount - float32(player.DeathCount())) / roundCount / 0.317
	roundsWithMultipleKillsRating := (float32(player.OneKillCount()) + 4*float32(player.TwoKillCount()) + 9*float32(player.ThreeKillCount()) + 16*float32(player.FourKillCount()) + 25*float32(player.FiveKillCount())) / roundCount / 1.277
	rating := (killRating + 0.7*survivalRating + roundsWithMultipleKillsRating) / 2.7

	return rating
}

func (player *Player) roundCount() int {
	return len(player.match.Rounds)
}

func (player *Player) oneVsXClutches(opponentCount int) []*Clutch {
	clutches := player.Clutches()
	for _, clutch := range player.match.Clutches {
		if clutch.OpponentCount == opponentCount {
			clutches = append(clutches, clutch)
		}
	}

	return clutches
}

func (player *Player) oneVsXClutchesWonCount(opponentCount int) int {
	clutches := player.oneVsXClutches(opponentCount)
	var count int
	for _, clutch := range clutches {
		if clutch.HasWon {
			count++
		}
	}

	return count
}

func (player *Player) oneVsXClutchesLostCount(opponentCount int) int {
	clutches := player.oneVsXClutches(opponentCount)
	var count int
	for _, clutch := range clutches {
		if !clutch.HasWon {
			count++
		}
	}

	return count
}

func (player *Player) getXKillCount(count int) int {
	var xKillCount int
	for _, kills := range player.match.KillsByRound() {
		playerKillInRoundCount := 0
		for _, kill := range kills {
			if kill.KillerSteamID64 != player.SteamID64 || kill.IsKillerControllingBot || kill.IsSuicide() || kill.IsTeamKill() {
				continue
			}

			playerKillInRoundCount++
		}

		if playerKillInRoundCount == count {
			xKillCount++
		}
	}

	return xKillCount
}

func (player *Player) IsInspectingWeapon(analyzer *Analyzer) bool {
	// weapon inspection animation lasts approx. 5 seconds
	return player.lastWeaponInspection.tick > -1 && !analyzer.secondsHasPassedSinceTick(5, player.lastWeaponInspection.tick)
}

func (player *Player) startWeaponInspection(tick int) {
	player.InspectWeaponCount++
	player.lastWeaponInspection = weaponInspectionData{
		tick:          tick,
		cancelledTick: -1,
	}
}

func (player *Player) stopWeaponInspection(tick int) {
	// update the cancellation tick only if we didn't already detected a cancellation otherwise we may have multiple
	// cancellations for a single inspection which would mess up the time-based logic in IsInspectingWeapon.
	if player.lastWeaponInspection.cancelledTick == -1 {
		player.lastWeaponInspection.cancelledTick = tick
	}
}

func (player *Player) reset() {
	player.Score = 0
	player.MvpCount = 0
	player.InspectWeaponCount = 0
	player.LeechValue = 0
	player.FeedValue = 0
	player.LeechCount = 0
	player.FeedCount = 0
	player.WastedUtilityValue = 0
	player.lastWeaponInspection = weaponInspectionData{
		tick:          -1,
		cancelledTick: -1,
	}
	player.hasCounterStrafeSampleCaches = false
	player.counterStrafeSamplesCache = nil
	player.counterStrafeComboSamplesCache = nil
	player.hasCounterStrafeSummaryCaches = false
	player.counterStrafeSummariesByDirection = nil
	player.counterStrafeComboSummary = counterStrafeComboSummary{}
}

func NewPlayer(analyzer *Analyzer, currentTeam common.Team, player common.Player) *Player {
	var team *Team
	if *analyzer.match.TeamA.CurrentSide == currentTeam {
		team = analyzer.match.TeamA
	} else {
		team = analyzer.match.TeamB
	}

	color, _ := player.ColorOrErr()
	rank := player.Rank()
	userID := 0
	if player.UserID <= math.MaxUint16 {
		userID = player.UserID & 0xff
	}

	newPlayer := &Player{
		match:              analyzer.match,
		SteamID64:          player.SteamID64,
		UserID:             userID,
		Name:               strings.ReplaceUTF8ByteSequences(player.Name),
		Team:               team,
		CrosshairShareCode: player.CrosshairCode(),
		Color:              color,
		RankType:           player.RankType(),
		Rank:               rank,
		OldRank:            rank,
		WinCount:           player.CompetitiveWins(),
		lastWeaponInspection: weaponInspectionData{
			tick:          -1,
			cancelledTick: -1,
		},
	}

	var activeWeaponProp st.Property
	if analyzer.isSource2 {
		if pawnEntity := player.PlayerPawnEntity(); pawnEntity != nil {
			activeWeaponProp = pawnEntity.Property("m_pWeaponServices.m_hActiveWeapon")
		}
	} else {
		activeWeaponProp = player.Entity.Property("m_hActiveWeapon")
	}
	if activeWeaponProp != nil {
		activeWeaponProp.OnUpdate(func(pv st.PropertyValue) {
			// switching weapon stops the weapon inspection animation
			newPlayer.stopWeaponInspection(analyzer.currentTick())
		})
	}

	return newPlayer
}

func (player *Player) TeamAttackDamage() int {
	var teamAttackDamage int
	for _, damage := range player.match.Damages {
		if damage.AttackerSteamID64 == player.SteamID64 && damage.VictimSteamID64 != 0 && damage.AttackerSide == damage.VictimSide && damage.AttackerSteamID64 != damage.VictimSteamID64 && !damage.IsGrenadeWeapon() {
			teamAttackDamage += damage.HealthDamage
		}
	}
	return teamAttackDamage
}

func (player *Player) TeamUtilityDamage() int {
	var teamUtilityDamage int
	for _, damage := range player.match.Damages {
		if damage.AttackerSteamID64 == player.SteamID64 && damage.VictimSteamID64 != 0 && damage.AttackerSide == damage.VictimSide && damage.AttackerSteamID64 != damage.VictimSteamID64 && damage.IsGrenadeWeapon() {
			teamUtilityDamage += damage.HealthDamage
		}
	}
	return teamUtilityDamage
}

func (player *Player) TeamFlashDuration() float32 {
	var teamFlashDuration float32
	for _, flashed := range player.match.PlayersFlashed {
		if flashed.FlasherSteamID64 == player.SteamID64 && flashed.FlashedSteamID64 != 0 && flashed.FlasherSide == flashed.FlashedSide && flashed.FlasherSteamID64 != flashed.FlashedSteamID64 {
			teamFlashDuration += flashed.Duration
		}
	}
	return teamFlashDuration
}
