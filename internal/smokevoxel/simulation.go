// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Go adaptation of osztenkurden/cs2parser smokeDensity.ts, revision
// 0c6c22e95251dfd6432d470ee3bbd3be2d61173d (MPL-2.0).

package smokevoxel

import (
	"encoding/binary"
	"fmt"
	"math"
	"sync"
)

var axes = [6][3]int{{-1, 0, 0}, {1, 0, 0}, {0, -1, 0}, {0, 1, 0}, {0, 0, -1}, {0, 0, 1}}
var lookup = sync.OnceValue(func() *gridLookup {
	table := new(gridLookup)
	for i := range Cells {
		table.coordinates[i] = Coordinates(i)
		for k, delta := range axes {
			x, y, z := table.coordinates[i][0]+delta[0], table.coordinates[i][1]+delta[1], table.coordinates[i][2]+delta[2]
			table.neighbours[i][k] = -1
			if x >= 0 && x < GridDim && y >= 0 && y < GridDim && z >= 0 && z < GridDim {
				table.neighbours[i][k] = Morton(x, y, z)
			}
		}
	}
	return table
})

type gridLookup struct {
	coordinates [Cells][3]int
	neighbours  [Cells][6]int
}
type source struct {
	index int
	age   float32
}

// Simulation reconstructs raw density only. It excludes opacity/fade, render
// interpolation, lighting and bullet/HE shader effects. Auxiliary inputs fail
// explicitly. Arithmetic and traversal order match the float32 reference.
type Simulation struct {
	current, next      *[Cells][4]float32
	active, nextActive *[Cells]bool
	blocked            [Cells]bool
	seeds              []source
	sequence           int
	origin             [3]float32
}

func New(origin [3]float32) (*Simulation, error) {
	for _, v := range origin {
		if !finite(v) {
			return nil, fmt.Errorf("non-finite smoke origin")
		}
	}
	return &Simulation{origin: origin, sequence: -1, current: new([Cells][4]float32), next: new([Cells][4]float32), active: new([Cells]bool), nextActive: new([Cells]bool)}, nil
}
func finite(v float32) bool         { return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0) }
func (s *Simulation) Sequence() int { return s.sequence }
func (s *Simulation) Density() []float32 {
	values := make([]float32, Cells)
	for i := range values {
		values[i] = s.current[i][0]
	}
	return values
}

// DensityAt returns a raw cell density, not a boolean visual-in-smoke flag.
// Points outside the fixed grid are known zero; non-finite points are unknown.
func (s *Simulation) DensityAt(point [3]float64) (float32, bool) {
	if s.sequence < 0 {
		return 0, false
	}
	var xyz [3]int
	for a, v := range point {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		grid := math.Floor((v-float64(s.origin[a]))/CellSize + 16)
		if grid < 0 || grid >= GridDim {
			return 0, true
		}
		xyz[a] = int(grid)
	}
	return s.current[Morton(xyz[0], xyz[1], xyz[2])][0], true
}

// Step validates all inputs before mutating the simulation. Every sequence,
// including a heartbeat, must be applied exactly once, beginning with zero.
func (s *Simulation) Step(frame Frame) error {
	if frame.Sequence != s.sequence+1 || frame.Sequence > 65535 {
		return fmt.Errorf("expected sequence %d, got %d", s.sequence+1, frame.Sequence)
	}
	in, err := DecodeFrame(frame.Payload)
	if err != nil {
		return err
	}
	if len(in.Extra) != 0 {
		return fmt.Errorf("unsupported auxiliary smoke records: %d", len(in.Extra))
	}
	var seeds []source
	if in.Seeds != nil {
		seeds = make([]source, len(in.Seeds))
		for i, seed := range in.Seeds {
			age := math.Float32frombits(binary.LittleEndian.Uint32(seed.State[1:]))
			if !finite(age) {
				return fmt.Errorf("non-finite seed state")
			}
			seeds[i] = source{Morton(int(seed.X), int(seed.Y), int(seed.Z)), age}
		}
		s.seeds = seeds
	}
	for _, update := range in.Blocked {
		for bit := range 64 {
			s.blocked[int(update.Index)*64+bit] = (update.Mask>>bit)&1 != 0
		}
	}
	table := lookup()
	if s.sequence < 0 {
		for _, seed := range s.seeds {
			x, y, z := table.coordinates[seed.index][0], table.coordinates[seed.index][1], table.coordinates[seed.index][2]
			for dz := -1; dz <= 1; dz++ {
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if (dx == 0 && dy == 0 && dz == 0) || x+dx < 0 || x+dx >= 32 || y+dy < 0 || y+dy >= 32 || z+dz < 0 || z+dz >= 32 {
							continue
						}
						j := Morton(x+dx, y+dy, z+dz)
						if s.active[j] || s.blocked[j] {
							continue
						}
						s.current[j][0] = float32(s.current[j][0] + 30)
						s.next[j][0] = s.current[j][0]
						s.active[j] = true
					}
				}
			}
			s.active[seed.index] = true
		}
	} else {
		fresh := 0
		for _, seed := range s.seeds {
			if seed.age < float32(.01) {
				fresh++
			}
		}
		cap := float32(50 * max(float32(1), float32(float32(1-float32(float32(fresh)/40))*3.5)))
		for n := range s.seeds {
			seed := &s.seeds[n]
			i := seed.index
			if in.StopSeeding {
				s.current[i][0] = 0
			} else {
				position := WorldCenter(table.coordinates[i], s.origin)
				var relative [3]float32
				for a := range 3 {
					relative[a] = float32(position[a] - s.origin[a])
				}
				distance := float32(math.Sqrt(float64(float32(float32(float32(relative[2]*relative[2])+float32(relative[1]*relative[1])) + float32(relative[0]*relative[0])))))
				if distance <= 80 {
					seed.age = float32(seed.age * float32(.8))
				} else {
					average := float32(0)
					for _, j := range table.neighbours[i] {
						if j >= 0 && s.active[j] {
							average = float32(average + min(float32(1), max(float32(0), float32(s.current[j][0]/50))))
						}
					}
					average = float32(average / 6)
					if average > float32(.2) {
						if seed.age > 0 {
							seed.age = float32(seed.age * float32(.8))
						}
					} else {
						seed.age = 1
					}
				}
				s.current[i][0] = min(cap, float32(s.current[i][0]+float32(float32(1-seed.age)*60)))
			}
			s.active[i] = true
			s.next[i] = s.current[i]
		}
	}
	if in.StopSeeding {
		s.seeds = nil
	}
	for i := range Cells {
		if !s.active[i] {
			continue
		}
		if s.blocked[i] {
			s.next[i][0] = 0
			continue
		}
		saturation := min(float32(1), max(float32(0), float32(s.next[i][0]/50)))
		var densities [6]float32
		incoming, blockedAxes := float32(0), 0
		for k, j := range table.neighbours[i] {
			if j < 0 || s.blocked[j] {
				blockedAxes |= 1 << (k >> 1)
				continue
			}
			densities[k] = s.current[j][0]
			delta := axes[k]
			direction := float32(float32(float32(-float32(delta[1])*s.current[j][2])-float32(float32(delta[2])*s.current[j][3])) - float32(float32(delta[0])*s.current[j][1]))
			weight := float32(.9)
			if k == 4 {
				weight = .79
			} else if k == 5 {
				weight = 1.2
			} else if direction > float32(.2) {
				weight = 1.25
			}
			transfer := float32(float32(float32(1-saturation)*weight) * float32(s.current[j][0]/6))
			s.next[j][0] = max(float32(0), float32(s.next[j][0]-transfer))
			incoming = float32(incoming + transfer)
			if !s.active[j] && s.current[i][0] > 5 {
				s.nextActive[j] = true
			}
		}
		x, y, z := table.coordinates[i][0], table.coordinates[i][1], table.coordinates[i][2]
		if x == 0 || x == 31 || y == 0 || y == 31 || z == 0 || z == 31 {
			s.next[i][0] = float32(s.next[i][0] * float32(.3))
			continue
		}
		s.next[i][0] = float32(s.next[i][0] + incoming)
		vx, vy := float32(0), float32(0)
		if blockedAxes&1 == 0 {
			vx = float32(densities[0] - densities[1])
		}
		if blockedAxes&2 == 0 {
			vy = float32(densities[2] - densities[3])
		}
		length := float32(math.Sqrt(float64(float32(float32(vy*vy) + float32(vx*vx)))))
		if length != 0 {
			inv := float32(1 / length)
			vx, vy = float32(vx*inv), float32(vy*inv)
		}
		s.next[i][1], s.next[i][2], s.next[i][3] = vx, vy, 0
		decay := float32(.5)
		if in.StopSeeding {
			decay = .25
		}
		s.next[i][0] = max(float32(0), float32(s.next[i][0]-decay))
	}
	for i := range Cells {
		if s.active[i] {
			s.current[i] = s.next[i]
		}
		s.active[i] = s.active[i] || s.nextActive[i]
		s.nextActive[i] = s.active[i]
	}
	s.current, s.next = s.next, s.current
	s.active, s.nextActive = s.nextActive, s.active
	s.sequence = frame.Sequence
	return nil
}
