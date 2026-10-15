package payload

import "testing"

func TestParseDistance(t *testing.T) {
	tests := []struct {
		input    string
		wantVal  float64
		wantUnit string
		wantErr  bool
	}{
		{"10km", 10, "kilometer", false},
		{"42.195km", 42.195, "kilometer", false},
		{"26.2mi", 26.2, "mile", false},
		{"400m", 400, "meter", false},
		{"abc", 0, "", true},
		{"10", 0, "", true},
		{"km", 0, "", true},
	}
	for _, tt := range tests {
		val, unit, err := parseDistance(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseDistance(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if val != tt.wantVal || unit != tt.wantUnit {
			t.Errorf("parseDistance(%q) = (%v, %q), want (%v, %q)", tt.input, val, unit, tt.wantVal, tt.wantUnit)
		}
	}
}

func TestParseGoalDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"50m", 3000, false},
		{"1h30m", 5400, false},
		{"2400s", 2400, false},
		{"3h30m0s", 12600, false},
		{"notaduration", 0, true},
	}
	for _, tt := range tests {
		got, err := parseGoalDuration(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseGoalDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("parseGoalDuration(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
