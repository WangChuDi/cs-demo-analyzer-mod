package smokevoxel

import (
	"bytes"
	"fmt"
)

// Replay owns a single entity lifetime's journal and current density state.
// Never key this state by entity index alone: indices are reused by the game.
type Replay struct {
	Simulation *Simulation
	journal    []byte
}

// Update handles appended journals and replacement snapshots. On failure the
// density is unavailable, even if earlier frames in this update were valid.
// A later complete prefix can recover it. Inputs remain available via decoding
// the raw journal, including auxiliary records unsupported by the simulation.
func (r *Replay) Update(data []byte, origin [3]float32) (frames []Frame, reset bool, err error) {
	defer func() {
		if err != nil {
			r.Simulation = nil
			r.journal = nil
		}
	}()
	reset = r.Simulation == nil || r.Simulation.origin != origin || !bytes.HasPrefix(data, r.journal)
	start := len(r.journal)
	if reset {
		r.Simulation, err = New(origin)
		if err != nil {
			return nil, true, err
		}
		start = 0
	}
	frames, err = DecodeJournal(data[start:])
	if err != nil {
		return nil, reset, err
	}
	for _, frame := range frames {
		if err = r.Simulation.Step(frame); err != nil {
			return frames, reset, fmt.Errorf("sequence %d: %w", frame.Sequence, err)
		}
	}
	r.journal = append(r.journal[:0], data...)
	return frames, reset, nil
}

func (r *Replay) Clear() { r.Simulation = nil; r.journal = nil }
