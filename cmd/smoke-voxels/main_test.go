package main

import (
	"testing"

	"github.com/akiver/cs-demo-analyzer/internal/smokevoxel"
)

func TestSparseGridMatchesDensitySummary(t *testing.T) {
	sim, err := smokevoxel.New([3]float32{100, 200, 300})
	if err != nil {
		t.Fatal(err)
	}
	if err := sim.Step(smokevoxel.Frame{Sequence: 0, Payload: []byte{0, 1, 1, 16, 16, 16, 5, 0, 0, 0, 0, 0}}); err != nil {
		t.Fatal(err)
	}
	full := summarize(sim, 5, true)
	if full.CellsAboveThreshold == 0 || len(full.Grid) != full.CellsAboveThreshold {
		t.Fatal("missing sparse grid")
	}
	for _, cell := range full.Grid {
		center := smokevoxel.WorldCenter(cell.Coordinates, [3]float32{100, 200, 300})
		value, known := sim.DensityAt([3]float64{float64(center[0]), float64(center[1]), float64(center[2])})
		if !known || value != cell.Density || value <= 5 {
			t.Fatal("grid and point query disagree")
		}
	}
	summary := summarize(sim, 5, false)
	if summary.Grid != nil || summary.SHA256 != full.SHA256 || summary.Sum != full.Sum {
		t.Fatal("grid option changed density")
	}
	excluded := summarize(sim, float64(full.Max), true)
	if excluded.CellsAboveThreshold != 0 || len(excluded.Grid) != 0 || excluded.SHA256 != full.SHA256 {
		t.Fatal("threshold must be strictly greater and must not change simulation")
	}
}
