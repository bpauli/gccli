package payload

import (
	"strings"
	"testing"
)

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
		d, err := ParseDistance(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseDistance(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if d != (Distance{tt.wantVal, tt.wantUnit}) {
			t.Errorf("ParseDistance(%q) = %+v, want (%v, %q)", tt.input, d, tt.wantVal, tt.wantUnit)
		}
	}
}

func TestParseGoal(t *testing.T) {
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
		{"", 0, true},
		{"0s", 0, true},
		{"500ms", 0, true},
		{"-1h", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseGoal(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseGoal(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if int(got) != tt.want {
			t.Errorf("ParseGoal(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseDistance_RejectsOutOfRange(t *testing.T) {
	digits400 := strings.Repeat("9", 400)
	_, err := ParseDistance(digits400 + "km")
	if want := `invalid distance "` + digits400 + `km": out of range`; err == nil || err.Error() != want {
		t.Fatalf("ParseDistance = %v, want error %q", err, want)
	}
}

func TestEvent_RejectsBadValues(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{"hour and minute out of range", Event{Time: "99:99"}, `invalid time "99:99" (use HH:MM, e.g. 07:30)`},
		{"hour 24", Event{Time: "24:00"}, `invalid time "24:00" (use HH:MM, e.g. 07:30)`},
		{"minute 60", Event{Time: "12:60"}, `invalid time "12:60" (use HH:MM, e.g. 07:30)`},
		{"no minutes", Event{Time: "7"}, `invalid time "7" (use HH:MM, e.g. 07:30)`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.event.Name, tt.event.Date, tt.event.Type = "x", "2026-11-01", "running"
			body, err := tt.event.JSON()
			if err == nil || err.Error() != tt.want {
				t.Fatalf("JSON() = %s, %v; want error %q", body, err, tt.want)
			}
		})
	}
}

func TestEvent_AcceptsValidTimes(t *testing.T) {
	for _, tm := range []string{"00:00", "07:00", "7:00", "09:15", "23:59"} {
		body, err := Event{Name: "x", Date: "2026-11-01", Type: "running", Time: tm}.JSON()
		if err != nil {
			t.Errorf("time %q: %v", tm, err)
			continue
		}
		if !strings.Contains(string(body), `"startTimeHhMm":"`+tm+`"`) {
			t.Errorf("time %q: body %s does not carry it", tm, body)
		}
	}
}
