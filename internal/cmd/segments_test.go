package cmd

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestSegmentKindsMatchEnums pins that the enum of each segment kind flag,
// which Kong checks at parse time, names exactly the keys of its map, so
// no value can reach Run without a Garmin ID.
func TestSegmentKindsMatchEnums(t *testing.T) {
	tests := []struct {
		cmd   reflect.Type
		field string
		keys  []string
	}{
		{reflect.TypeFor[SegmentsSearchCmd](), "ActivityType", slices.Collect(maps.Keys(segmentActivityTypes))},
		{reflect.TypeFor[SegmentsSearchCmd](), "Classification", slices.Collect(maps.Keys(segmentClassifications))},
		{reflect.TypeFor[SegmentsSearchCmd](), "Surface", slices.Collect(maps.Keys(segmentSurfaces))},
		{reflect.TypeFor[SegmentsLeaderboardCmd](), "Scope", slices.Collect(maps.Keys(leaderboardScopes))},
	}
	for _, tt := range tests {
		t.Run(tt.cmd.Name()+"."+tt.field, func(t *testing.T) {
			f, ok := tt.cmd.FieldByName(tt.field)
			if !ok {
				t.Fatalf("%s has no %s field", tt.cmd.Name(), tt.field)
			}
			got := strings.Split(f.Tag.Get("enum"), ",")
			if want := slices.Sorted(slices.Values(tt.keys)); !slices.Equal(got, want) {
				t.Errorf("enum of %s = %v, want the sorted map keys %v", tt.field, got, want)
			}
		})
	}
}
