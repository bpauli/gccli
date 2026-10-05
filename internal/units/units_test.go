package units

import (
	"reflect"
	"testing"
)

var allSystems = []System{Metric, StatuteUS, StatuteUK}

type formatCase struct {
	in   float64
	want [3]string
}

func runFormatCases(t *testing.T, name string, format func(System, float64) string, cases []formatCase) {
	t.Helper()
	for _, tc := range cases {
		for _, sys := range allSystems {
			if got := format(sys, tc.in); got != tc.want[sys] {
				t.Errorf("%s.%s(%v) = %q, want %q", sys, name, tc.in, got, tc.want[sys])
			}
		}
	}
}

func TestDistance(t *testing.T) {
	runFormatCases(t, "Distance", System.Distance, []formatCase{
		{0, [3]string{"-", "-", "-"}},
		{1000, [3]string{"1.00 km", "0.62 mi", "0.62 mi"}},
		{5123.45, [3]string{"5.12 km", "3.18 mi", "3.18 mi"}},
		{42195.0, [3]string{"42.20 km", "26.22 mi", "26.22 mi"}},
		{1609.344, [3]string{"1.61 km", "1.00 mi", "1.00 mi"}},
	})
}

func TestSpeed(t *testing.T) {
	runFormatCases(t, "Speed", System.Speed, []formatCase{
		{0, [3]string{"-", "-", "-"}},
		{2.778, [3]string{"10.0 km/h", "6.2 mph", "6.2 mph"}},
		{5.556, [3]string{"20.0 km/h", "12.4 mph", "12.4 mph"}},
		{4.4704, [3]string{"16.1 km/h", "10.0 mph", "10.0 mph"}},
	})
}

func TestPace(t *testing.T) {
	runFormatCases(t, "Pace", System.Pace, []formatCase{
		{0, [3]string{"-", "-", "-"}},
		{2.847, [3]string{"5:51 /km", "9:25 /mi", "9:25 /mi"}},
		{4.167, [3]string{"3:59 /km", "6:26 /mi", "6:26 /mi"}},
		{3.333, [3]string{"5:00 /km", "8:02 /mi", "8:02 /mi"}},
		{2.0833333333333335, [3]string{"8:00 /km", "12:52 /mi", "12:52 /mi"}},
	})
}

func TestElevation(t *testing.T) {
	runFormatCases(t, "Elevation", System.Elevation, []formatCase{
		{0, [3]string{"-", "-", "-"}},
		{85, [3]string{"85 m", "279 ft", "85 m"}},
		{1234.5, [3]string{"1235 m", "4050 ft", "1235 m"}},
		{304.8, [3]string{"305 m", "1000 ft", "305 m"}},
		{266, [3]string{"266 m", "873 ft", "266 m"}},
	})
}

func TestParseSystem(t *testing.T) {
	tests := []struct {
		in      string
		want    System
		wantErr bool
	}{
		{"metric", Metric, false},
		{"statute_us", StatuteUS, false},
		{"statute_uk", StatuteUK, false},
		{"  STATUTE_US ", StatuteUS, false},
		{"auto", Metric, true},
		{"", Metric, true},
		{"furlongs", Metric, true},
	}
	for _, tt := range tests {
		got, err := ParseSystem(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseSystem(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseSystem(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestSystemString_RoundTrips(t *testing.T) {
	for _, sys := range allSystems {
		got, err := ParseSystem(sys.String())
		if err != nil || got != sys {
			t.Errorf("ParseSystem(%q) = %v, %v; want %v", sys.String(), got, err, sys)
		}
	}
}

func TestParsePreference(t *testing.T) {
	tests := []struct {
		in      string
		want    Preference
		wantErr bool
	}{
		{"", Auto, false},
		{"auto", Auto, false},
		{"AUTO", Auto, false},
		{"metric", Pin(Metric), false},
		{"statute_us", Pin(StatuteUS), false},
		{"statute_uk", Pin(StatuteUK), false},
		{"imperial", Auto, true},
	}
	for _, tt := range tests {
		got, err := ParsePreference(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParsePreference(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParsePreference(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestPreference_Pinned(t *testing.T) {
	if _, ok := Auto.Pinned(); ok {
		t.Error("Auto.Pinned() reported pinned")
	}
	if (Preference{}) != Auto {
		t.Error("zero Preference is not Auto")
	}
	sys, ok := Pin(StatuteUK).Pinned()
	if !ok || sys != StatuteUK {
		t.Errorf("Pin(StatuteUK).Pinned() = %v, %v", sys, ok)
	}
	if Pin(Metric) == Auto {
		t.Error("Pin(Metric) equals Auto")
	}
}

func TestPreferenceNames(t *testing.T) {
	want := []string{"auto", "metric", "statute_us", "statute_uk"}
	if got := PreferenceNames(); !reflect.DeepEqual(got, want) {
		t.Errorf("PreferenceNames() = %v, want %v", got, want)
	}
	for _, name := range PreferenceNames() {
		p, err := ParsePreference(name)
		if err != nil || p.String() != name {
			t.Errorf("ParsePreference(%q) = %v, %v", name, p, err)
		}
	}
}
