package api

import (
	"math"
	"sort"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
)

const (
	smokeStatsModel           = "smoke_sphere_v1"
	smokeStatsRadius          = 144.0 // Explicit analysis model, not a decoded engine volume.
	smokeStatsBloomSeconds    = 1.0
	smokeStatsMaxGap          = 0.125
	smokeStatsMinPassDistance = 128.0
	smokeStatsCoreRadius      = 96.0
	smokeAssistRadius         = 600.0
)

type SmokeCloud struct {
	ID          string  `json:"id"`
	RoundNumber int     `json:"roundNumber"`
	StartTick   int     `json:"startTick"`
	ReadyTick   float64 `json:"readyTick"`
	EndTick     int     `json:"endTick"`
	EndReason   string  `json:"endReason"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Z           float64 `json:"z"`
}

func (cloud *SmokeCloud) center() r3.Vector { return r3.Vector{X: cloud.X, Y: cloud.Y, Z: cloud.Z} }

type SmokeAssistance struct {
	Kind     string      `json:"kind"`
	Tick     int         `json:"tick"`
	EndTick  int         `json:"endTick"`
	Side     common.Team `json:"side"`
	Position r3.Vector   `json:"position"`
}

type SmokeVisit struct {
	Model             string            `json:"model"`
	RoundNumber       int               `json:"roundNumber"`
	PlayerSteamID64   uint64            `json:"playerSteamId"`
	PlayerName        string            `json:"playerName"`
	Side              common.Team       `json:"side"`
	SmokeID           string            `json:"smokeId"`
	EntryTick         float64           `json:"entryTick"`
	ExitTick          float64           `json:"exitTick"`
	EntryPosition     r3.Vector         `json:"entryPosition"`
	ExitPosition      r3.Vector         `json:"exitPosition"`
	EntryReason       string            `json:"entryReason"`
	EndReason         string            `json:"endReason"`
	Duration          float64           `json:"duration"`
	MinCenterDistance float64           `json:"minCenterDistance"`
	IsFullPass        bool              `json:"isFullPass"`
	HasAssistanceData bool              `json:"hasAssistanceData"`
	IsDryPass         bool              `json:"isDryPass"`
	Reason            string            `json:"reason"`
	Assistance        []SmokeAssistance `json:"assistance,omitempty"`
	cloud             *SmokeCloud
}

type SmokeStats struct {
	DrySmokePassCount    int     `json:"drySmokePassCount"`
	SmokeStayDuration    float64 `json:"smokeStayDuration"`
	SmokeObservedSeconds float64 `json:"smokeObservedSeconds"`
	HasSmokeTrackingData bool    `json:"hasSmokeTrackingData"`
	SmokeModel           string  `json:"smokeModel"`
}

type smokePlayerSample struct {
	tick, frame, round int
	id                 uint64
	name               string
	side               common.Team
	eye                r3.Vector
}

type smokePlayerTrack struct {
	last  smokePlayerSample
	visit *SmokeVisit
}
type smokeTracker struct {
	clouds   map[string]*SmokeCloud
	players  map[uint64]*smokePlayerTrack
	supports []*smokeSupport
	fires    map[string]*smokeSupport
}
type smokeSupport struct {
	SmokeAssistance
	round int
}
type smokeSpan struct {
	a, b            float64
	entered, exited bool
	cloud           *SmokeCloud
}

func smokeFiniteVector(v r3.Vector) bool {
	return summonFinite(v.X) && summonFinite(v.Y) && summonFinite(v.Z)
}
func smokeLerp(a, b r3.Vector, t float64) r3.Vector { return a.Add(b.Sub(a).Mul(t)) }

// Return the union of line/sphere intersections, clipped to observed smoke
// lifetimes. Overlapping smokes contribute time once, not once per grenade.
func smokeSpans(before, after smokePlayerSample, clouds map[string]*SmokeCloud) []smokeSpan {
	delta := after.eye.Sub(before.eye)
	aa := delta.Dot(delta)
	dt := float64(after.tick - before.tick)
	var spans []smokeSpan
	for _, cloud := range clouds {
		if cloud.RoundNumber != after.round {
			continue
		}
		lo, hi := math.Max(0, (cloud.ReadyTick-float64(before.tick))/dt), 1.0
		if cloud.EndReason != "" {
			hi = math.Min(hi, float64(cloud.EndTick-before.tick)/dt)
		}
		if hi <= lo {
			continue
		}
		offset := before.eye.Sub(cloud.center())
		a, b := math.Inf(-1), math.Inf(1)
		if aa == 0 {
			if offset.Norm() >= smokeStatsRadius {
				continue
			}
		} else {
			bb, cc := 2*offset.Dot(delta), offset.Dot(offset)-smokeStatsRadius*smokeStatsRadius
			discriminant := bb*bb - 4*aa*cc
			if discriminant <= 0 {
				continue
			} // A tangent has zero residence time.
			root := math.Sqrt(discriminant)
			a, b = (-bb-root)/(2*aa), (-bb+root)/(2*aa)
		}
		start, end := math.Max(a, lo), math.Min(b, hi)
		if end <= start {
			continue
		}
		spans = append(spans, smokeSpan{a: start, b: end, entered: a >= lo && a >= 0,
			exited: b <= hi && (cloud.EndReason == "" || float64(before.tick)+b*dt < float64(cloud.EndTick)), cloud: cloud})
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].a == spans[j].a {
			return spans[i].cloud.ID < spans[j].cloud.ID
		}
		return spans[i].a < spans[j].a
	})
	var union []smokeSpan
	for _, span := range spans {
		if len(union) == 0 || span.a > union[len(union)-1].b+1e-12 {
			union = append(union, span)
			continue
		}
		last := &union[len(union)-1]
		if span.a == last.a {
			last.entered = last.entered && span.entered
		}
		if span.b > last.b {
			last.b, last.exited = span.b, span.exited
		} else if span.b == last.b {
			last.exited = last.exited && span.exited
		}
	}
	return union
}

func closestSmokeDistance(a, b, center r3.Vector) float64 {
	delta := b.Sub(a)
	t := 0.0
	if length := delta.Dot(delta); length > 0 {
		t = math.Max(0, math.Min(1, center.Sub(a).Dot(delta)/length))
	}
	return smokeLerp(a, b, t).Sub(center).Norm()
}

func (analyzer *Analyzer) smokeState() *smokeTracker {
	if analyzer.smokeTracker == nil {
		analyzer.smokeTracker = &smokeTracker{clouds: map[string]*SmokeCloud{}, players: map[uint64]*smokePlayerTrack{}, fires: map[string]*smokeSupport{}}
	}
	return analyzer.smokeTracker
}

func (analyzer *Analyzer) closeSmokeVisit(track *smokePlayerTrack, reason string) {
	if track.visit == nil {
		return
	}
	track.visit.EndReason = reason
	if track.visit.Duration > 0 {
		analyzer.match.SmokeVisits = append(analyzer.match.SmokeVisits, track.visit)
	}
	track.visit = nil
}

func (analyzer *Analyzer) breakSmokePlayer(id uint64, reason string) {
	state := analyzer.smokeState()
	if track := state.players[id]; track != nil {
		analyzer.closeSmokeVisit(track, reason)
	}
	delete(state.players, id)
}

func (analyzer *Analyzer) updateSmokePlayer(sample smokePlayerSample, rate float64) {
	if sample.id == 0 || !smokeFiniteVector(sample.eye) || (sample.side != common.TeamTerrorists && sample.side != common.TeamCounterTerrorists) {
		analyzer.breakSmokePlayer(sample.id, "unavailable")
		return
	}
	state := analyzer.smokeState()
	previous := state.players[sample.id]
	if previous == nil {
		state.players[sample.id] = &smokePlayerTrack{last: sample}
		return
	}
	before := previous.last
	if sample.tick == before.tick && sample.frame > before.frame {
		previous.last = sample
		return
	}
	dt := float64(sample.tick-before.tick) / rate
	if !summonFinite(rate) || rate <= 0 || dt <= 0 || dt > smokeStatsMaxGap || sample.frame <= before.frame || sample.round != before.round || sample.side != before.side || sample.eye.Sub(before.eye).Norm() > 128 {
		analyzer.closeSmokeVisit(previous, "gap")
		previous.last = sample
		return
	}
	if analyzer.match.smokeObservedSeconds == nil {
		analyzer.match.smokeObservedSeconds = map[roundPlayerKey]float64{}
	}
	analyzer.match.smokeObservedSeconds[roundPlayerKey{sample.round, sample.id}] += dt
	spans := smokeSpans(before, sample, state.clouds)
	if previous.visit != nil && (len(spans) == 0 || spans[0].a > 1e-12) {
		reason := "movement"
		if cloud := previous.visit.cloud; cloud.EndReason != "" && cloud.EndTick <= before.tick {
			reason = "smoke_expired"
		}
		analyzer.closeSmokeVisit(previous, reason)
	}
	for _, span := range spans {
		start, end := smokeLerp(before.eye, sample.eye, span.a), smokeLerp(before.eye, sample.eye, span.b)
		if previous.visit == nil {
			reason := "observed_inside"
			if span.entered {
				reason = "movement"
			}
			previous.visit = &SmokeVisit{Model: smokeStatsModel, RoundNumber: sample.round, PlayerSteamID64: sample.id, PlayerName: sample.name,
				Side: sample.side, SmokeID: span.cloud.ID, EntryTick: float64(before.tick) + span.a*float64(sample.tick-before.tick),
				EntryPosition: start, EntryReason: reason, MinCenterDistance: math.Inf(1), cloud: span.cloud}
		}
		visit := previous.visit
		visit.Duration += (span.b - span.a) * dt
		visit.ExitTick = float64(before.tick) + span.b*float64(sample.tick-before.tick)
		visit.ExitPosition = end
		visit.MinCenterDistance = math.Min(visit.MinCenterDistance, closestSmokeDistance(start, end, visit.cloud.center()))
		if span.b < 1-1e-12 {
			reason := "smoke_expired"
			if span.exited {
				reason = "movement"
			}
			analyzer.closeSmokeVisit(previous, reason)
		}
	}
	previous.last = sample
}

func (analyzer *Analyzer) finishSmokeTracking(reason string) {
	state := analyzer.smokeState()
	for _, track := range state.players {
		analyzer.closeSmokeVisit(track, reason)
	}
	clear(state.players)
	for _, cloud := range state.clouds {
		if cloud.EndReason == "" {
			cloud.EndTick, cloud.EndReason = analyzer.currentTick(), reason
		}
	}
	for _, fire := range state.fires {
		if fire.EndTick == 0 {
			fire.EndTick = analyzer.currentTick()
		}
	}
	clear(state.clouds)
	clear(state.fires)
}

// Round restarts must discard open visits and auxiliary events as well as the
// exported match rows; otherwise reused round numbers could inherit assistance.
func (analyzer *Analyzer) discardSmokeRound(round int) {
	if analyzer.smokeTracker == nil {
		return
	}
	state := analyzer.smokeTracker
	for id, track := range state.players {
		if track.last.round == round {
			delete(state.players, id)
		}
	}
	for id, cloud := range state.clouds {
		if cloud.RoundNumber == round {
			delete(state.clouds, id)
		}
	}
	for id, fire := range state.fires {
		if fire.round == round {
			delete(state.fires, id)
		}
	}
	kept := state.supports[:0]
	for _, support := range state.supports {
		if support.round != round {
			kept = append(kept, support)
		}
	}
	state.supports = kept
}

func (analyzer *Analyzer) markSmokePasses() {
	if analyzer.smokeTracker == nil {
		return
	}
	rate := analyzer.match.TickRate
	sort.Slice(analyzer.match.SmokeVisits, func(i, j int) bool {
		a, b := analyzer.match.SmokeVisits[i], analyzer.match.SmokeVisits[j]
		if a.RoundNumber != b.RoundNumber {
			return a.RoundNumber < b.RoundNumber
		}
		if a.EntryTick != b.EntryTick {
			return a.EntryTick < b.EntryTick
		}
		if a.PlayerSteamID64 != b.PlayerSteamID64 {
			return a.PlayerSteamID64 < b.PlayerSteamID64
		}
		return a.SmokeID < b.SmokeID
	})
	for _, visit := range analyzer.match.SmokeVisits {
		visit.IsFullPass, visit.IsDryPass, visit.HasAssistanceData = false, false, false
		visit.Assistance = nil
		visit.Reason = "not_full_pass"
		cloud := visit.cloud
		if cloud == nil || visit.EntryReason != "movement" || visit.EndReason != "movement" || !summonFinite(rate) || rate <= 0 ||
			visit.MinCenterDistance > smokeStatsCoreRadius || visit.ExitPosition.Sub(visit.EntryPosition).Norm() < smokeStatsMinPassDistance ||
			visit.EntryPosition.Sub(cloud.center()).Dot(visit.ExitPosition.Sub(cloud.center())) > 1e-6 ||
			(cloud.EndReason != "" && visit.ExitTick >= float64(cloud.EndTick)) {
			continue
		}
		visit.IsFullPass, visit.HasAssistanceData = true, true
		for _, support := range analyzer.smokeTracker.supports {
			if support.round != visit.RoundNumber || float64(support.Tick) > visit.ExitTick+0.5*rate {
				continue
			}
			end := support.EndTick
			if end == 0 {
				end = support.Tick
				if support.Kind == "inferno" {
					end = int(math.Ceil(visit.ExitTick + 0.5*rate))
				}
			}
			if float64(end) < visit.EntryTick-rate || support.Position.Sub(visit.ExitPosition).Norm() > smokeAssistRadius {
				continue
			}
			if support.Side != common.TeamTerrorists && support.Side != common.TeamCounterTerrorists {
				visit.HasAssistanceData = false
			} else if support.Side != visit.Side {
				continue
			}
			visit.Assistance = append(visit.Assistance, support.SmokeAssistance)
		}
		visit.Reason = "assisted"
		if !visit.HasAssistanceData {
			visit.Reason = "unknown_assistance"
		} else if len(visit.Assistance) == 0 {
			visit.IsDryPass, visit.Reason = true, "dry_pass"
		}
	}
}

func (player *Player) SmokeStats() SmokeStats {
	stats := SmokeStats{}
	for key, seconds := range player.match.smokeObservedSeconds {
		if key.steamID64 == player.SteamID64 {
			stats.SmokeObservedSeconds += seconds
		}
	}
	stats.HasSmokeTrackingData = stats.SmokeObservedSeconds > 0
	if stats.HasSmokeTrackingData {
		stats.SmokeModel = smokeStatsModel
	}
	for _, visit := range player.match.SmokeVisits {
		if visit.PlayerSteamID64 == player.SteamID64 {
			stats.SmokeStayDuration += visit.Duration
			if visit.IsDryPass {
				stats.DrySmokePassCount++
			}
		}
	}
	return stats
}
