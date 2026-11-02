package smokevoxel

import (
	"math"
	"testing"
)

func TestSegmentCellIntersections(t *testing.T) {
	grid := make([]float32, Cells)
	grid[Morton(16, 16, 16)] = 6
	for _, tt := range []struct {
		name     string
		from, to [3]float64
		want     []Span
	}{
		{"axis", [3]float64{-10, 10, 10}, [3]float64{30, 10, 10}, []Span{{.25, .75}}},
		{"reverse", [3]float64{30, 10, 10}, [3]float64{-10, 10, 10}, []Span{{.25, .75}}},
		{"diagonal", [3]float64{-10, -10, -10}, [3]float64{30, 30, 30}, []Span{{.25, .75}}},
		{"stationary", [3]float64{10, 10, 10}, [3]float64{10, 10, 10}, []Span{{0, 1}}},
		{"upper face excluded", [3]float64{20, -10, 10}, [3]float64{20, 30, 10}, nil},
		{"lower face included", [3]float64{0, -10, 10}, [3]float64{0, 30, 10}, []Span{{.25, .75}}},
		{"another floor", [3]float64{-10, 10, 30}, [3]float64{30, 10, 30}, nil},
		{"outside grid", [3]float64{-400, 10, 10}, [3]float64{-350, 10, 10}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := SegmentSpans(grid, [3]float32{}, tt.from, tt.to, 5)
			if len(got) != len(tt.want) {
				t.Fatalf("%+v want %+v", got, tt.want)
			}
			for i := range got {
				if math.Abs(got[i].Start-tt.want[i].Start) > 1e-12 || math.Abs(got[i].End-tt.want[i].End) > 1e-12 {
					t.Fatal(got, tt.want)
				}
			}
		})
	}
	grid[Morton(16, 16, 16)] = 5
	if spans := SegmentSpans(grid, [3]float32{}, [3]float64{10, 10, 10}, [3]float64{10, 10, 10}, 5); len(spans) != 0 {
		t.Fatal("included density equal to threshold")
	}
}

func TestSegmentUnionAndGridOrigin(t *testing.T) {
	grid := make([]float32, Cells)
	grid[Morton(16, 16, 16)] = 10
	grid[Morton(18, 16, 16)] = 10
	from, to := [3]float64{100, 210, 310}, [3]float64{180, 210, 310}
	spans := SegmentSpans(grid, [3]float32{100, 200, 300}, from, to, 5)
	if len(spans) != 2 || spans[0] != (Span{0, .25}) || spans[1] != (Span{.5, .75}) {
		t.Fatal(spans)
	}
	grid[Morton(17, 16, 16)] = 10
	spans = SegmentSpans(grid, [3]float32{100, 200, 300}, from, to, 5)
	if len(spans) != 1 || spans[0] != (Span{0, .75}) {
		t.Fatal(spans)
	}
}
