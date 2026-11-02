package api

import (
	"bytes"
	"fmt"
	"math"
	"sort"

	"github.com/akiver/cs-demo-analyzer/internal/smokevoxel"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/common"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

const smokeVoxelModel = "smoke_voxel_density_v1"
const smokeVoxelThreshold = float32(5)

type SmokeVoxelVisit struct {
	Model             string            `json:"model"`
	RoundNumber       int               `json:"roundNumber"`
	PlayerSteamID64   uint64            `json:"playerSteamId"`
	PlayerName        string            `json:"playerName"`
	SmokeIDs          []string          `json:"smokeIds"`
	EntryTick         float64           `json:"entryTick"`
	ExitTick          float64           `json:"exitTick"`
	EntryPosition     r3.Vector         `json:"entryPosition"`
	ExitPosition      r3.Vector         `json:"exitPosition"`
	EntryInwardNormal r3.Vector         `json:"entryInwardNormal"`
	ExitOutwardNormal r3.Vector         `json:"exitOutwardNormal"`
	EntryReason       string            `json:"entryReason"`
	EndReason         string            `json:"endReason"`
	Duration          float64           `json:"duration"`
	DensityThreshold  float32           `json:"densityThreshold"`
	Side              common.Team       `json:"side"`
	IsFullPass        bool              `json:"isFullPass"`
	HasAssistanceData bool              `json:"hasAssistanceData"`
	IsDryPass         bool              `json:"isDryPass"`
	Reason            string            `json:"reason"`
	Assistance        []SmokeAssistance `json:"assistance,omitempty"`
}
type voxelObservation struct {
	tick    int
	origin  [3]float32
	density []float32 // nil means unknown, never a known empty grid
}
type voxelSmokeVolume struct {
	cloud                     *SmokeCloud
	replay                    smokevoxel.Replay
	last                      smokevoxel.EntityData
	history                   []voxelObservation
	dirty, visible, destroyed bool
}
type voxelSmokePlayer struct {
	last    smokePlayerSample
	visit   *SmokeVoxelVisit
	outside bool // A known outside interval is required for a boundary entry.
}
type voxelSmokeTracker struct {
	entities       map[*st.Entity]*voxelSmokeVolume
	players        map[uint64]*voxelSmokePlayer
	protocol       int
	classAvailable bool
}
type voxelResidenceSpan struct {
	smokevoxel.Span
	ids []string
}
type voxelResidenceInterval struct {
	a, b  float64
	known bool
	spans []voxelResidenceSpan
}

func (analyzer *Analyzer) registerSmokeVoxelHandlers() {
	if !analyzer.isSource2 {
		return
	}
	state := &voxelSmokeTracker{entities: map[*st.Entity]*voxelSmokeVolume{}, players: map[uint64]*voxelSmokePlayer{}}
	analyzer.smokeVoxels = state
	analyzer.match.smokeResidenceModel = smokeVoxelModel
	analyzer.parser.RegisterNetMessageHandler(func(header *msg.CDemoFileHeader) { state.protocol = int(header.GetPatchVersion()) })
	analyzer.parser.RegisterEventHandler(func(events.DataTablesParsed) {
		class := analyzer.parser.ServerClasses().FindByName("CSmokeGrenadeProjectile")
		if class == nil {
			return
		}
		state.classAvailable = true
		class.OnEntityCreated(func(entity *st.Entity) {
			volume := &voxelSmokeVolume{visible: true, dirty: true}
			state.entities[entity] = volume
			entity.OnUpdateFinished(func() { volume.dirty = true })
			entity.OnLeavePVS(func() { volume.visible = false; volume.dirty = true })
			entity.OnEnterPVS(func() { volume.visible = true; volume.dirty = true })
			entity.OnDestroy(func() { volume.destroyed = true; volume.dirty = true })
		})
	})
}

func (analyzer *Analyzer) attachSmokeVoxelCloud(cloud *SmokeCloud, event events.GrenadeEvent) {
	state := analyzer.smokeVoxels
	if state == nil {
		return
	}
	cloud.VoxelStatus = "missing_projectile_entity"
	var entity *st.Entity
	if event.Projectile != nil {
		entity = event.Projectile.Entity
	} else if event.GrenadeEntityID > 0 {
		entity = analyzer.parser.GameState().Entities()[event.GrenadeEntityID]
	}
	if volume := state.entities[entity]; volume != nil {
		volume.replay.Clear()
		volume.last = smokevoxel.EntityData{}
		volume.history = nil
		volume.cloud = cloud
		volume.dirty = true
		cloud.voxel = volume
		cloud.VoxelStatus = "awaiting_data"
	}
}

func (volume *voxelSmokeVolume) observe(observation voxelObservation) {
	if n := len(volume.history); n > 0 && volume.history[n-1].tick == observation.tick {
		volume.history[n-1] = observation
	} else {
		volume.history = append(volume.history, observation)
	}
}

// Publish only after the packet's entities have finished updating. History is
// timestamped when a grid was observed; new grids never apply to earlier ticks.
func (analyzer *Analyzer) publishSmokeVoxelGrids() {
	state := analyzer.smokeVoxels
	if state == nil {
		return
	}
	tick := analyzer.currentTick()
	for entity, volume := range state.entities {
		cloud := volume.cloud
		if !volume.dirty || cloud == nil || cloud.EndReason != "" {
			continue
		}
		volume.dirty = false
		observation := voxelObservation{tick: tick}
		var data smokevoxel.EntityData
		var err error
		if volume.destroyed || !volume.visible {
			err = fmt.Errorf("entity_not_visible")
		} else {
			data, err = smokevoxel.ReadEntity(entity)
		}
		if err == nil && state.protocol <= 13963 {
			err = fmt.Errorf("unsupported_patch_%d", state.protocol)
		}
		if err == nil && len(data.Journal) == 0 {
			err = fmt.Errorf("missing_voxel_journal")
		}
		if err == nil && volume.replay.Simulation != nil && volume.last.EffectTick == data.EffectTick && volume.last.Origin == data.Origin && bytes.Equal(volume.last.Journal, data.Journal) {
			continue
		}
		if err == nil {
			if data.EffectTick != volume.last.EffectTick {
				volume.replay.Clear()
			}
			frames, _, replayErr := volume.replay.Update(data.Journal, data.Origin)
			err = replayErr
			if err == nil {
				observation.origin = data.Origin
				observation.density = volume.replay.Simulation.Density()
				for _, density := range observation.density {
					if !summonFinite(float64(density)) {
						err = fmt.Errorf("non_finite_simulated_density")
						observation.density = nil
						break
					}
				}
				if err == nil {
					cloud.VoxelFrames += len(frames)
					cloud.VoxelStatus = "available"
				}
			}
		}
		cloud.VoxelUpdates++
		if cloud.VoxelUpdates == 1 {
			cloud.VoxelFirstTick = tick
		}
		cloud.VoxelLastTick = tick
		if err != nil {
			cloud.VoxelUnavailableUpdates++
			cloud.VoxelStatus = err.Error()
			if cloud.VoxelUnavailableReasons == nil {
				cloud.VoxelUnavailableReasons = map[string]int{}
			}
			cloud.VoxelUnavailableReasons[err.Error()]++
			volume.replay.Clear()
		}
		volume.last = data
		volume.observe(observation)
	}
}

func (volume *voxelSmokeVolume) observationAt(tick float64) *voxelObservation {
	for i := len(volume.history) - 1; i >= 0; i-- {
		if float64(volume.history[i].tick) <= tick {
			return &volume.history[i]
		}
	}
	return nil
}

// Partition time at observation and lifetime boundaries, then integrate the
// trajectory through the held density grids. An unknown active cloud makes the
// interval unavailable; overlapping known clouds are integrated as a union.
func voxelResidenceIntervals(before, after smokePlayerSample, clouds map[string]*SmokeCloud) []voxelResidenceInterval {
	dt := float64(after.tick - before.tick)
	cuts := []float64{0, 1}
	addCut := func(tick int) {
		f := float64(tick-before.tick) / dt
		if f > 0 && f < 1 {
			cuts = append(cuts, f)
		}
	}
	for _, cloud := range clouds {
		if cloud.RoundNumber != after.round {
			continue
		}
		addCut(cloud.StartTick)
		if cloud.EndReason != "" {
			addCut(cloud.EndTick)
		}
		if cloud.voxel != nil {
			for _, observation := range cloud.voxel.history {
				addCut(observation.tick)
			}
		}
	}
	sort.Float64s(cuts)
	var intervals []voxelResidenceInterval
	for n := 1; n < len(cuts); n++ {
		a, b := cuts[n-1], cuts[n]
		if b-a <= 1e-12 {
			continue
		}
		interval := voxelResidenceInterval{a: a, b: b, known: true}
		startTick := float64(before.tick) + a*dt
		start, end := smokeLerp(before.eye, after.eye, a), smokeLerp(before.eye, after.eye, b)
		for _, cloud := range clouds {
			if cloud.RoundNumber != after.round || startTick < float64(cloud.StartTick) || (cloud.EndReason != "" && startTick >= float64(cloud.EndTick)) {
				continue
			}
			if cloud.voxel == nil {
				interval.known = false
				break
			}
			observation := cloud.voxel.observationAt(startTick)
			if observation == nil || observation.density == nil {
				interval.known = false
				break
			}
			for _, span := range smokevoxel.SegmentSpans(observation.density, observation.origin, [3]float64{start.X, start.Y, start.Z}, [3]float64{end.X, end.Y, end.Z}, smokeVoxelThreshold) {
				interval.spans = append(interval.spans, voxelResidenceSpan{smokevoxel.Span{Start: a + span.Start*(b-a), End: a + span.End*(b-a)}, []string{cloud.ID}})
			}
		}
		if interval.known {
			sort.Slice(interval.spans, func(i, j int) bool {
				if interval.spans[i].Start == interval.spans[j].Start {
					return interval.spans[i].ids[0] < interval.spans[j].ids[0]
				}
				return interval.spans[i].Start < interval.spans[j].Start
			})
			var union []voxelResidenceSpan
			for _, span := range interval.spans {
				if len(union) == 0 || span.Start > union[len(union)-1].End+1e-12 {
					union = append(union, span)
				} else {
					last := &union[len(union)-1]
					last.End = math.Max(last.End, span.End)
					last.ids = append(last.ids, span.ids...)
				}
			}
			interval.spans = union
		}
		intervals = append(intervals, interval)
	}
	return intervals
}

// Use the same held grids and threshold as residence, including its conservative
// unknown-cloud rule. A point on a cell's upper face belongs to the next cell.
func voxelSmokePointState(tick float64, round int, eye r3.Vector, clouds map[string]*SmokeCloud) (inside, known bool) {
	known = true
	point := [3]float64{eye.X, eye.Y, eye.Z}
	for _, cloud := range clouds {
		if cloud.RoundNumber != round || tick < float64(cloud.StartTick) || (cloud.EndReason != "" && tick >= float64(cloud.EndTick)) {
			continue
		}
		if cloud.voxel == nil {
			return false, false
		}
		observation := cloud.voxel.observationAt(tick)
		if observation == nil || observation.density == nil {
			return false, false
		}
		inside = inside || len(smokevoxel.SegmentSpans(observation.density, observation.origin, point, point, smokeVoxelThreshold)) > 0
	}
	return
}

// At an exact sample/update boundary, require a geometric crossing in both the
// old and new grid. Growth/expiry or a newly available grid must not masquerade
// as player movement. The probes only classify a boundary; they add no time.
func voxelSmokeMovementBoundary(tick float64, round int, eye, delta r3.Vector, entering bool, clouds map[string]*SmokeCloud) bool {
	length := delta.Norm()
	if length == 0 {
		return false
	}
	probe := delta.Mul(1e-6 / length)
	for _, at := range []float64{math.Nextafter(tick, math.Inf(-1)), tick} {
		behind, behindKnown := voxelSmokePointState(at, round, eye.Sub(probe), clouds)
		ahead, aheadKnown := voxelSmokePointState(at, round, eye.Add(probe), clouds)
		if !behindKnown || !aheadKnown || behind == entering || ahead != entering {
			return false
		}
	}
	return true
}

// Determine which exposed voxel faces were crossed, rather than using movement
// direction (which can include a large tangent component). At corners, combine
// the exposed face normals. Internal cell boundaries contribute no normal.
func voxelSmokeBoundaryNormal(tick float64, round int, eye, delta r3.Vector, entering bool, clouds map[string]*SmokeCloud) r3.Vector {
	if delta.Norm() == 0 {
		return r3.Vector{}
	}
	probe := delta.Mul(1e-6 / delta.Norm())
	inside, outside := eye.Add(probe), eye.Sub(probe)
	if !entering {
		inside, outside = outside, inside
	}
	in, inKnown := voxelSmokePointState(tick, round, inside, clouds)
	out, outKnown := voxelSmokePointState(tick, round, outside, clouds)
	if !inKnown || !outKnown || !in || out {
		return r3.Vector{}
	}
	p, q := [3]float64{inside.X, inside.Y, inside.Z}, [3]float64{outside.X, outside.Y, outside.Z}
	d := [3]float64{delta.X, delta.Y, delta.Z}
	var normal [3]float64
	for _, cloud := range clouds {
		if cloud.RoundNumber != round || tick < float64(cloud.StartTick) || (cloud.EndReason != "" && tick >= float64(cloud.EndTick)) || cloud.voxel == nil {
			continue
		}
		observation := cloud.voxel.observationAt(tick)
		if observation == nil || len(smokevoxel.SegmentSpans(observation.density, observation.origin, p, p, smokeVoxelThreshold)) == 0 {
			continue
		}
		for axis := range 3 {
			a := math.Floor((p[axis]-float64(observation.origin[axis]))/smokevoxel.CellSize + 16)
			b := math.Floor((q[axis]-float64(observation.origin[axis]))/smokevoxel.CellSize + 16)
			if a == b {
				continue
			}
			across := p
			across[axis] = q[axis]
			occupied, known := voxelSmokePointState(tick, round, r3.Vector{X: across[0], Y: across[1], Z: across[2]}, clouds)
			if known && !occupied {
				normal[axis] = math.Copysign(1, d[axis])
			}
		}
	}
	result := r3.Vector{X: normal[0], Y: normal[1], Z: normal[2]}
	if length := result.Norm(); length > 0 {
		return result.Mul(1 / length)
	}
	return result
}

// A pass must leave through another side of the occupied volume. Same-face
// returns fail even when tangential movement makes the endpoints far apart.
func voxelSmokeOppositeSides(visit *SmokeVoxelVisit) bool {
	in, out := visit.EntryInwardNormal, visit.ExitOutwardNormal
	displacement := visit.ExitPosition.Sub(visit.EntryPosition)
	return in.Norm() > 0 && out.Norm() > 0 && in.Dot(out) >= -1e-9 &&
		displacement.Dot(in) > 1e-6 && displacement.Dot(out) > 1e-6
}

func (analyzer *Analyzer) closeVoxelSmokeVisit(player *voxelSmokePlayer, reason string) {
	if player.visit == nil {
		return
	}
	player.visit.EndReason = reason
	if player.visit.Duration > 0 {
		sort.Strings(player.visit.SmokeIDs)
		ids := player.visit.SmokeIDs[:0]
		for _, id := range player.visit.SmokeIDs {
			if len(ids) == 0 || ids[len(ids)-1] != id {
				ids = append(ids, id)
			}
		}
		player.visit.SmokeIDs = ids
		analyzer.match.SmokeVoxelVisits = append(analyzer.match.SmokeVoxelVisits, player.visit)
	}
	player.visit = nil
}
func (analyzer *Analyzer) breakVoxelSmokePlayer(id uint64, reason string) {
	if state := analyzer.smokeVoxels; state != nil {
		if player := state.players[id]; player != nil {
			analyzer.closeVoxelSmokeVisit(player, reason)
		}
		delete(state.players, id)
	}
}
func (analyzer *Analyzer) updateVoxelSmokePlayer(sample smokePlayerSample, rate float64) {
	state := analyzer.smokeVoxels
	if state == nil {
		return
	}
	previous := state.players[sample.id]
	if previous == nil {
		inside, known := voxelSmokePointState(float64(sample.tick), sample.round, sample.eye, analyzer.smokeState().clouds)
		state.players[sample.id] = &voxelSmokePlayer{last: sample, outside: known && !inside}
		return
	}
	before := previous.last
	if before.tick == sample.tick && sample.frame > before.frame {
		previous.last = sample
		return
	}
	dt := float64(sample.tick-before.tick) / rate
	if !summonFinite(rate) || rate <= 0 || dt <= 0 || dt > smokeStatsMaxGap || sample.frame <= before.frame || sample.round != before.round || sample.side != before.side || sample.eye.Sub(before.eye).Norm() > 128 || !state.classAvailable || state.protocol <= 13963 {
		analyzer.closeVoxelSmokeVisit(previous, "unavailable")
		previous.last = sample
		previous.outside = false
		return
	}
	if analyzer.match.smokeVoxelObservedSeconds == nil {
		analyzer.match.smokeVoxelObservedSeconds = map[roundPlayerKey]float64{}
	}
	for _, interval := range voxelResidenceIntervals(before, sample, analyzer.smokeState().clouds) {
		if !interval.known {
			analyzer.closeVoxelSmokeVisit(previous, "unavailable")
			previous.outside = false
			continue
		}
		analyzer.match.smokeVoxelObservedSeconds[roundPlayerKey{sample.round, sample.id}] += (interval.b - interval.a) * dt
		if previous.visit != nil && (len(interval.spans) == 0 || interval.spans[0].Start > interval.a+1e-12) {
			reason := "outside_density"
			tick := float64(before.tick) + interval.a*float64(sample.tick-before.tick)
			if math.Abs(previous.visit.ExitTick-tick) < 1e-8 && voxelSmokeMovementBoundary(tick, sample.round, previous.visit.ExitPosition, sample.eye.Sub(before.eye), false, analyzer.smokeState().clouds) {
				reason = "movement"
				previous.visit.ExitOutwardNormal = voxelSmokeBoundaryNormal(tick, sample.round, previous.visit.ExitPosition, sample.eye.Sub(before.eye), false, analyzer.smokeState().clouds)
			}
			analyzer.closeVoxelSmokeVisit(previous, reason)
		}
		for _, span := range interval.spans {
			start, end := smokeLerp(before.eye, sample.eye, span.Start), smokeLerp(before.eye, sample.eye, span.End)
			if previous.visit == nil {
				reason := "observed_density"
				tick := float64(before.tick) + span.Start*float64(sample.tick-before.tick)
				if span.Start > interval.a+1e-12 || (previous.outside && voxelSmokeMovementBoundary(tick, sample.round, start, sample.eye.Sub(before.eye), true, analyzer.smokeState().clouds)) {
					reason = "movement"
				}
				previous.visit = &SmokeVoxelVisit{Model: smokeVoxelModel, RoundNumber: sample.round, PlayerSteamID64: sample.id, PlayerName: sample.name,
					Side: sample.side, EntryTick: tick, EntryPosition: start, EntryReason: reason, DensityThreshold: smokeVoxelThreshold}
				if reason == "movement" {
					previous.visit.EntryInwardNormal = voxelSmokeBoundaryNormal(tick, sample.round, start, sample.eye.Sub(before.eye), true, analyzer.smokeState().clouds)
				}
			}
			visit := previous.visit
			visit.SmokeIDs = append(visit.SmokeIDs, span.ids...)
			visit.Duration += (span.End - span.Start) * dt
			visit.ExitTick = float64(before.tick) + span.End*float64(sample.tick-before.tick)
			visit.ExitPosition = end
			if span.End < interval.b-1e-12 {
				visit.ExitOutwardNormal = voxelSmokeBoundaryNormal(visit.ExitTick, sample.round, end, sample.eye.Sub(before.eye), false, analyzer.smokeState().clouds)
				analyzer.closeVoxelSmokeVisit(previous, "movement")
			}
		}
		previous.outside = len(interval.spans) == 0 || interval.spans[len(interval.spans)-1].End < interval.b-1e-12
	}
	previous.last = sample
}

func (analyzer *Analyzer) markVoxelSmokePasses() {
	rate := analyzer.match.TickRate
	sort.Slice(analyzer.match.SmokeVoxelVisits, func(i, j int) bool {
		a, b := analyzer.match.SmokeVoxelVisits[i], analyzer.match.SmokeVoxelVisits[j]
		if a.RoundNumber != b.RoundNumber {
			return a.RoundNumber < b.RoundNumber
		}
		if a.EntryTick != b.EntryTick {
			return a.EntryTick < b.EntryTick
		}
		return a.PlayerSteamID64 < b.PlayerSteamID64
	})
	for _, visit := range analyzer.match.SmokeVoxelVisits {
		visit.IsFullPass, visit.IsDryPass, visit.HasAssistanceData = false, false, false
		visit.Assistance = nil
		visit.Reason = "not_full_pass"
		if visit.EntryReason != "movement" || visit.EndReason != "movement" || !summonFinite(rate) || rate <= 0 ||
			visit.ExitPosition.Sub(visit.EntryPosition).Norm() < smokeStatsMinPassDistance {
			continue
		}
		if visit.EntryInwardNormal.Norm() == 0 || visit.ExitOutwardNormal.Norm() == 0 {
			visit.Reason = "unknown_crossing_side"
			continue
		}
		if !voxelSmokeOppositeSides(visit) {
			visit.Reason = "same_side_return"
			continue
		}
		visit.IsFullPass = true
		visit.HasAssistanceData, visit.IsDryPass, visit.Reason, visit.Assistance = analyzer.smokePassAssistance(visit.RoundNumber, visit.Side, visit.EntryTick, visit.ExitTick, visit.ExitPosition)
	}
}

func (analyzer *Analyzer) pruneSmokeVoxelHistory() {
	state := analyzer.smokeVoxels
	if state == nil {
		return
	}
	cut := analyzer.currentTick() - int(math.Ceil(analyzer.parser.TickRate()*smokeStatsMaxGap)) - 1
	for entity, volume := range state.entities {
		if volume.destroyed {
			delete(state.entities, entity)
		}
	}
	for _, cloud := range analyzer.smokeState().clouds {
		if cloud.voxel == nil {
			continue
		}
		history := cloud.voxel.history
		keep := 0
		for i := range history {
			if history[i].tick <= cut {
				keep = i
			}
		}
		if keep > 0 {
			cloud.voxel.history = append([]voxelObservation(nil), history[keep:]...)
		}
	}
}

func (analyzer *Analyzer) finishVoxelSmokeTracking(reason string, discardRound int) {
	state := analyzer.smokeVoxels
	if state == nil {
		return
	}
	for id, player := range state.players {
		if discardRound > 0 && player.last.round != discardRound {
			continue
		}
		if reason != "discard" {
			analyzer.closeVoxelSmokeVisit(player, reason)
		}
		delete(state.players, id)
	}
	release := func(volume *voxelSmokeVolume) {
		if volume.cloud == nil || (discardRound > 0 && volume.cloud.RoundNumber != discardRound) {
			return
		}
		volume.cloud.voxel = nil
		volume.cloud = nil
		volume.replay.Clear()
		volume.last = smokevoxel.EntityData{}
		volume.history = nil
	}
	for _, volume := range state.entities {
		release(volume)
	}
	if analyzer.smokeTracker != nil {
		for _, cloud := range analyzer.smokeTracker.clouds {
			if cloud.voxel != nil {
				release(cloud.voxel)
			}
		}
	}
}
