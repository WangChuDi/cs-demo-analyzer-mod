// smoke-voxels is an opt-in diagnostic exporter. It deliberately does not
// replace the match analyzer's sphere-based smoke statistics.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"sort"

	"github.com/akiver/cs-demo-analyzer/internal/smokevoxel"
	dem "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/events"
	"github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/msg"
	st "github.com/markus-wa/demoinfocs-golang/v6/pkg/demoinfocs/sendtables"
)

type options struct {
	demo, output  string
	grid, players bool
	threshold     float64
}
type smoke struct {
	lifetime string
	replay   smokevoxel.Replay
	last     []byte
	origin   [3]float32
	effect   int
	tick     int
	visible  bool
	seenData bool
}
type gridCell struct {
	Coordinates [3]int  `json:"coordinates"`
	Density     float32 `json:"density"`
}
type densitySummary struct {
	Sequence            int        `json:"sequence"`
	CellsAboveThreshold int        `json:"cellsAboveThreshold"`
	Max                 float32    `json:"max"`
	Sum                 float64    `json:"sum"`
	SHA256              string     `json:"sha256"`
	Grid                []gridCell `json:"grid,omitempty"`
}
type totals struct {
	Type               string `json:"type"`
	Lifetimes          int    `json:"lifetimesWithData"`
	Updates            int    `json:"updates"`
	RecoveredUpdates   int    `json:"recoveredUpdates"`
	UnavailableUpdates int    `json:"unavailableUpdates"`
	AppliedFrames      int    `json:"appliedFrames"`
	PlayerCellQueries  int    `json:"playerCellQueriesAboveThreshold"`
}

func summarize(sim *smokevoxel.Simulation, threshold float64, grid bool) densitySummary {
	result := densitySummary{Sequence: sim.Sequence()}
	values := sim.Density()
	data := make([]byte, len(values)*4)
	for i, value := range values {
		binary.LittleEndian.PutUint32(data[i*4:], math.Float32bits(value))
		result.Max = max(result.Max, value)
		result.Sum += float64(value)
		if float64(value) > threshold {
			result.CellsAboveThreshold++
			if grid {
				result.Grid = append(result.Grid, gridCell{smokevoxel.Coordinates(i), value})
			}
		}
	}
	hash := sha256.Sum256(data)
	result.SHA256 = hex.EncodeToString(hash[:])
	return result
}

// Network vectors may be reused and updated in place. Validate and copy only
// the declared valid prefix; never truncate an out-of-range integer to a byte.
func voxelBytes(entity *st.Entity) ([]byte, error) {
	sizeValue, ok := entity.PropertyValue("m_nVoxelFrameDataSize")
	if !ok {
		return nil, errors.New("missing voxel valid size")
	}
	size, ok := sizeValue.Any.(int32)
	if !ok || size < 0 {
		return nil, errors.New("invalid voxel valid size")
	}
	if size == 0 {
		return nil, nil
	}
	value, ok := entity.PropertyValue("m_VoxelFrameData")
	if !ok {
		return nil, errors.New("missing voxel byte array")
	}
	values, ok := value.Any.([]any)
	if !ok || int(size) > len(values) {
		return nil, errors.New("voxel valid size exceeds array")
	}
	data := make([]byte, int(size))
	for i := range data {
		v, ok := values[i].(uint64)
		if !ok || v > 255 {
			return nil, fmt.Errorf("voxel element %d is not a byte", i)
		}
		data[i] = byte(v)
	}
	return data, nil
}

func vector(entity *st.Entity, name string) ([3]float32, error) {
	value, ok := entity.PropertyValue(name)
	if !ok {
		return [3]float32{}, fmt.Errorf("missing %s", name)
	}
	var origin [3]float32
	switch v := value.Any.(type) {
	case [3]float32:
		origin = v
	case []float32:
		if len(v) != 3 {
			return origin, errors.New("invalid origin length")
		}
		copy(origin[:], v)
	default:
		return origin, errors.New("invalid origin type")
	}
	for _, v := range origin {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return origin, errors.New("non-finite origin")
		}
	}
	return origin, nil
}
func integer(entity *st.Entity, name string) int {
	value, ok := entity.PropertyValue(name)
	if !ok {
		return 0
	}
	v, _ := value.Any.(int32)
	return int(v)
}

func inspect(input io.Reader, output io.Writer, opts options) error {
	parser := dem.NewParser(input)
	defer parser.Close()
	encoder := json.NewEncoder(output)
	var writeErr error
	emit := func(value any) {
		if writeErr == nil {
			writeErr = encoder.Encode(value)
		}
	}
	protocol, created, foundClass := 0, 0, false
	states := map[*st.Entity]*smoke{}
	total := totals{Type: "summary"}
	parser.RegisterNetMessageHandler(func(header *msg.CDemoFileHeader) {
		protocol = int(header.GetPatchVersion())
		emit(map[string]any{"type": "header", "model": "smoke_journal_density_v1_experimental", "patchVersion": protocol,
			"gridDim": 32, "cellSize": 20, "densityThreshold": opts.threshold, "referenceRevision": "0c6c22e95251dfd6432d470ee3bbd3be2d61173d",
			"densityIsOpacity": false, "bulletAndHEHoles": false})
	})
	parser.RegisterEventHandler(func(events.DataTablesParsed) {
		class := parser.ServerClasses().FindByName("CSmokeGrenadeProjectile")
		if class == nil {
			return
		}
		foundClass = true
		class.OnEntityCreated(func(entity *st.Entity) {
			created++
			state := &smoke{lifetime: fmt.Sprintf("%d:%d:%d", entity.ID(), entity.SerialNum(), created), visible: true}
			states[entity] = state
			entity.OnLeavePVS(func() { state.visible = false })
			entity.OnEnterPVS(func() { state.visible = true })
			entity.OnDestroy(func() {
				if state.last != nil {
					emit(map[string]any{"type": "end", "lifetime": state.lifetime, "demoTick": parser.GameState().IngameTick(), "replaced": entity.Replaced()})
				}
				delete(states, entity)
			})
			entity.OnUpdateFinished(func() {
				data, err := voxelBytes(entity)
				if len(data) == 0 && err == nil {
					if state.last != nil {
						emit(map[string]any{"type": "clear", "lifetime": state.lifetime, "demoTick": parser.GameState().IngameTick()})
					}
					state.last = nil
					state.replay.Clear()
					return
				}
				origin, originErr := vector(entity, "m_vSmokeDetonationPos")
				if err == nil {
					err = originErr
				}
				effect := integer(entity, "m_nSmokeEffectTickBegin")
				if err == nil && bytes.Equal(data, state.last) && origin == state.origin && effect == state.effect {
					return
				}
				if !state.seenData {
					total.Lifetimes++
					state.seenData = true
				}
				total.Updates++
				record := map[string]any{"type": "update", "lifetime": state.lifetime, "entityId": entity.ID(),
					"demoTick": parser.GameState().IngameTick(), "demoFrame": parser.CurrentFrame(), "origin": origin,
					"effectTickServer": effect, "networkUpdate": integer(entity, "m_nVoxelUpdate"), "validBytes": len(data)}
				if state.effect != effect {
					state.replay.Clear()
				}
				if err == nil && protocol <= 13963 {
					err = fmt.Errorf("unsupported or unknown patch version %d", protocol)
				}
				if err == nil {
					frames, reset, replayErr := state.replay.Update(data, origin)
					record["reset"], record["frames"] = reset, frames
					err = replayErr
					if err == nil {
						record["density"] = summarize(state.replay.Simulation, opts.threshold, opts.grid)
						total.AppliedFrames += len(frames)
						total.RecoveredUpdates++
					}
				}
				if err != nil {
					state.replay.Clear()
					total.UnavailableUpdates++
					record["error"], record["journal"] = err.Error(), data
				}
				state.last, state.origin, state.effect, state.tick = append(state.last[:0], data...), origin, effect, parser.GameState().IngameTick()
				emit(record)
			})
		})
	})
	if opts.players {
		parser.RegisterEventHandler(func(events.FrameDone) {
			// Query after all entities in the packet have updated. Ticks here are
			// observation times; journal sequence is not an exact render timestamp.
			ordered := make([]*smoke, 0, len(states))
			for _, state := range states {
				ordered = append(ordered, state)
			}
			sort.Slice(ordered, func(i, j int) bool { return ordered[i].lifetime < ordered[j].lifetime })
			players := parser.GameState().Participants().Playing()
			sort.Slice(players, func(i, j int) bool {
				if players[i] == nil {
					return players[j] != nil
				}
				return players[j] != nil && players[i].SteamID64 < players[j].SteamID64
			})
			for _, state := range ordered {
				if !state.visible || state.replay.Simulation == nil || state.effect == 0 {
					continue
				}
				for _, player := range players {
					if player == nil || player.SteamID64 == 0 || player.IsBot || player.IsControllingBot() || !player.IsAlive() {
						continue
					}
					eye, known := player.PositionEyes()
					if !known {
						continue
					}
					position := [3]float64{eye.X, eye.Y, eye.Z}
					density, known := state.replay.Simulation.DensityAt(position)
					if !known || float64(density) <= opts.threshold {
						continue
					}
					emit(map[string]any{"type": "player_cell", "lifetime": state.lifetime, "demoTick": parser.GameState().IngameTick(),
						"demoFrame": parser.CurrentFrame(), "stateObservedDemoTick": state.tick, "sequence": state.replay.Simulation.Sequence(),
						"steamId": player.SteamID64, "name": player.Name, "eye": position, "rawDensity": density})
					total.PlayerCellQueries++
				}
			}
		})
	}
	for {
		more, err := parser.ParseNextFrame()
		if writeErr != nil {
			return writeErr
		}
		if err != nil {
			return err
		}
		if !more {
			break
		}
	}
	if !foundClass {
		return errors.New("CS2 smoke projectile class was not found")
	}
	emit(total)
	return writeErr
}

func run(args []string) error {
	var opts options
	flags := flag.NewFlagSet("smoke-voxels", flag.ContinueOnError)
	flags.StringVar(&opts.demo, "demo-path", "", "CS2 demo path (required)")
	flags.StringVar(&opts.output, "output", "", "NDJSON output file (required; must not exist)")
	flags.BoolVar(&opts.grid, "grid", false, "Include cells above threshold for each updated density grid")
	flags.BoolVar(&opts.players, "players", false, "Include live player-eye cell queries above threshold")
	flags.Float64Var(&opts.threshold, "density-threshold", 5, "Raw density reporting threshold, not opacity")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if opts.demo == "" || opts.output == "" {
		return errors.New("-demo-path and -output are required")
	}
	if opts.threshold < 0 || math.IsNaN(opts.threshold) || math.IsInf(opts.threshold, 0) {
		return errors.New("density threshold must be finite and nonnegative")
	}
	input, err := os.Open(opts.demo)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(opts.output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	buffer := bufio.NewWriter(output)
	parseErr := inspect(input, buffer, opts)
	flushErr := buffer.Flush()
	closeErr := output.Close()
	return errors.Join(parseErr, flushErr, closeErr)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
