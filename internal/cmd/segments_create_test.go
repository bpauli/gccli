package cmd

import (
	"slices"
	"testing"
)

func track(distances ...float64) []trackPoint {
	points := make([]trackPoint, len(distances))
	for i, d := range distances {
		points[i] = trackPoint{distance: d}
	}
	return points
}

func distancesOf(points []trackPoint) []float64 {
	var out []float64
	for _, p := range points {
		out = append(out, p.distance)
	}
	return out
}

// TestCutTrack pins the points of a new segment: from the first point at or
// after the start to the last point at or before the end, both inclusive,
// and no segment with fewer than two points.
func TestCutTrack(t *testing.T) {
	points := track(0, 80, 160, 250, 340, 420)
	tests := []struct {
		name         string
		start, end   float64
		wantDistance []float64
	}{
		{"inside", 100, 350, []float64{160, 250, 340}},
		{"on the points", 80, 340, []float64{80, 160, 250, 340}},
		{"whole track", 0, 1000, []float64{0, 80, 160, 250, 340, 420}},
		{"two points", 80, 160, []float64{80, 160}},
		{"one point", 100, 200, nil},
		{"no point", 170, 240, nil},
		{"after the track", 500, 600, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := distancesOf(cutTrack(points, tt.start, tt.end))
			if !slices.Equal(got, tt.wantDistance) {
				t.Errorf("cutTrack(%v, %v) = %v, want %v", tt.start, tt.end, got, tt.wantDistance)
			}
		})
	}
	if got := cutTrack(nil, 0, 100); got != nil {
		t.Errorf("cutTrack of no points = %v, want nil", got)
	}
}

// TestSegmentGrade pins the grade of a new segment: the climb from the
// first to the last point over the run, in percent with 2 decimals, and 0
// when it cannot be computed.
func TestSegmentGrade(t *testing.T) {
	at := func(d, e float64) trackPoint { return trackPoint{distance: d, elevation: &e} }
	tests := []struct {
		name string
		span []trackPoint
		want float64
	}{
		{"climb", []trackPoint{at(160, 45), at(250, 99), at(340, 50.6)}, 3.11},
		{"descent", []trackPoint{at(0, 50), at(300, 41)}, -3},
		{"no start elevation", []trackPoint{{distance: 0}, at(100, 10)}, 0},
		{"no end elevation", []trackPoint{at(0, 10), {distance: 100}}, 0},
		{"no run", []trackPoint{at(100, 10), at(100, 20)}, 0},
		{"negative run", []trackPoint{at(200, 10), at(100, 20)}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := segmentGrade(tt.span); got != tt.want {
				t.Errorf("segmentGrade = %v, want %v", got, tt.want)
			}
		})
	}
}
