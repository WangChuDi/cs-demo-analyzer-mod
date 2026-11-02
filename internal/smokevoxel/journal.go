// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.
//
// Go adaptation of osztenkurden/cs2parser smokeVoxel.ts, revision
// 0c6c22e95251dfd6432d470ee3bbd3be2d61173d (MPL-2.0).

// Package smokevoxel decodes the modern CS2 smoke simulation journal.
// Seeds are simulation inputs, not occupied cells or rendered opacity.
package smokevoxel

import (
	"encoding/binary"
	"fmt"
)

const (
	GridDim  = 32
	Cells    = GridDim * GridDim * GridDim
	CellSize = 20
)

type Frame struct {
	Sequence int    `json:"sequence"`
	Payload  []byte `json:"payload"`
}

type Seed struct {
	X, Y, Z byte
	State   [5]byte
}

type MaskUpdate struct {
	Index uint16
	Mask  uint64
}

type Inputs struct {
	StopSeeding bool
	// Nil retains the previous list; a non-nil empty list replaces it.
	Seeds   []Seed
	Blocked []MaskUpdate
	Extra   [][20]byte
}

// DecodeJournal copies each payload so it survives the next entity update.
// data must already be clipped to m_nVoxelFrameDataSize.
func DecodeJournal(data []byte) ([]Frame, error) {
	var frames []Frame
	for off := 0; off < len(data); {
		if len(data)-off < 4 {
			return nil, fmt.Errorf("truncated journal header at %d", off)
		}
		seq, size := int(binary.LittleEndian.Uint16(data[off:])), int(binary.LittleEndian.Uint16(data[off+2:]))
		off += 4
		if size > len(data)-off {
			return nil, fmt.Errorf("journal sequence %d overruns valid data", seq)
		}
		frames = append(frames, Frame{seq, append([]byte(nil), data[off:off+size]...)})
		off += size
	}
	return frames, nil
}

// DecodeFrame supports the byte-aligned seed format for patch versions >13963.
// Unknown sections, out-of-grid coordinates and trailing data are errors.
func DecodeFrame(payload []byte) (Inputs, error) {
	var in Inputs
	if len(payload) < 3 {
		return in, fmt.Errorf("truncated voxel frame")
	}
	flags := payload[1]
	if flags & ^byte(3) != 0 {
		return in, fmt.Errorf("unsupported section flags %d", flags)
	}
	in.StopSeeding = payload[0] != 0
	off := 2
	if flags&1 != 0 {
		count := int(payload[off])
		off++
		if count*8 > len(payload)-off {
			return Inputs{}, fmt.Errorf("truncated seed list")
		}
		in.Seeds = make([]Seed, count)
		for i := range in.Seeds {
			p := payload[off : off+8]
			if p[0] >= GridDim || p[1] >= GridDim || p[2] >= GridDim {
				return Inputs{}, fmt.Errorf("seed outside grid")
			}
			in.Seeds[i] = Seed{X: p[0], Y: p[1], Z: p[2]}
			copy(in.Seeds[i].State[:], p[3:])
			off += 8
		}
	}
	if flags&2 != 0 {
		if len(payload)-off < 2 {
			return Inputs{}, fmt.Errorf("truncated mask count")
		}
		count := int(binary.LittleEndian.Uint16(payload[off:]))
		off += 2
		if count*10 > len(payload)-off {
			return Inputs{}, fmt.Errorf("truncated rejection mask")
		}
		for range count {
			index := binary.LittleEndian.Uint16(payload[off:])
			if index >= Cells/64 {
				return Inputs{}, fmt.Errorf("rejection mask outside grid")
			}
			in.Blocked = append(in.Blocked, MaskUpdate{index, binary.LittleEndian.Uint64(payload[off+2:])})
			off += 10
		}
	}
	if off >= len(payload) {
		return Inputs{}, fmt.Errorf("missing auxiliary record count")
	}
	count := int(payload[off])
	off++
	if count*20 > len(payload)-off {
		return Inputs{}, fmt.Errorf("truncated auxiliary records")
	}
	for range count {
		var record [20]byte
		copy(record[:], payload[off:off+20])
		off += 20
		in.Extra = append(in.Extra, record)
	}
	if off != len(payload) {
		return Inputs{}, fmt.Errorf("unexpected trailing voxel data")
	}
	return in, nil
}

// Morton uses the low bit for X, then Y, then Z.
func Morton(x, y, z int) int {
	index := 0
	for b := range 5 {
		index |= ((x>>b)&1)<<(3*b) | ((y>>b)&1)<<(3*b+1) | ((z>>b)&1)<<(3*b+2)
	}
	return index
}

func Coordinates(index int) [3]int {
	var xyz [3]int
	for a := range 3 {
		for b := range 5 {
			xyz[a] |= ((index >> (3*b + a)) & 1) << b
		}
	}
	return xyz
}

func WorldCenter(xyz [3]int, origin [3]float32) [3]float32 {
	var point [3]float32
	for a := range 3 {
		point[a] = float32(float32(float32(xyz[a]-16)*CellSize+origin[a]) + 10)
	}
	return point
}
