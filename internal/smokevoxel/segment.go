package smokevoxel

import (
	"math"
	"sort"
)

type Span struct{ Start, End float64 }

// SegmentSpans intersects a linear path with the union of half-open cells above
// threshold in one observed raw-density grid. Time is a fraction of the path.
// This is geometric integration of a held grid, not render-time interpolation.
func SegmentSpans(density []float32, origin [3]float32, from, to [3]float64, threshold float32) []Span {
	if len(density) != Cells || !finite(threshold) || threshold < 0 {
		return nil
	}
	lo, hi := 0.0, 1.0
	var delta [3]float64
	for a := range 3 {
		if math.IsNaN(from[a]) || math.IsInf(from[a], 0) || math.IsNaN(to[a]) || math.IsInf(to[a], 0) || !finite(origin[a]) {
			return nil
		}
		delta[a] = to[a] - from[a]
		lower, upper := float64(origin[a])-GridDim*CellSize/2, float64(origin[a])+GridDim*CellSize/2
		if delta[a] == 0 {
			if from[a] < lower || from[a] >= upper {
				return nil
			}
			continue
		}
		a0, a1 := (lower-from[a])/delta[a], (upper-from[a])/delta[a]
		if a0 > a1 {
			a0, a1 = a1, a0
		}
		lo, hi = math.Max(lo, a0), math.Min(hi, a1)
		if hi <= lo {
			return nil
		}
	}
	cuts := []float64{lo, hi}
	for a := range 3 {
		if delta[a] == 0 {
			continue
		}
		for boundary := 1; boundary < GridDim; boundary++ {
			t := (float64(origin[a]) + float64((boundary-16)*CellSize) - from[a]) / delta[a]
			if t > lo && t < hi {
				cuts = append(cuts, t)
			}
		}
	}
	sort.Float64s(cuts)
	var spans []Span
	for i := 1; i < len(cuts); i++ {
		a, b := cuts[i-1], cuts[i]
		if b-a <= 1e-12 {
			continue
		}
		mid := (a + b) / 2
		var xyz [3]int
		for axis := range 3 {
			xyz[axis] = int(math.Floor((from[axis]+mid*delta[axis]-float64(origin[axis]))/CellSize + 16))
		}
		if xyz[0] < 0 || xyz[0] >= GridDim || xyz[1] < 0 || xyz[1] >= GridDim || xyz[2] < 0 || xyz[2] >= GridDim {
			continue
		}
		if !(density[Morton(xyz[0], xyz[1], xyz[2])] > threshold) {
			continue
		}
		if len(spans) > 0 && math.Abs(spans[len(spans)-1].End-a) <= 1e-12 {
			spans[len(spans)-1].End = b
		} else {
			spans = append(spans, Span{a, b})
		}
	}
	return spans
}
