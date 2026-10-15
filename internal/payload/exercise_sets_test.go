package payload

import (
	"strings"
	"testing"
)

func ptrFloat64(v float64) *float64 { return &v }

func ptrInt(v int) *int { return &v }

func TestParseOneExercise_Valid(t *testing.T) {
	tests := []struct {
		input    string
		category string
		name     string
		reps     int
		weightMg float64
		duration *float64
		restSecs *int
	}{
		{
			input:    "BENCH_PRESS/BARBELL_BENCH_PRESS:12@20",
			category: "BENCH_PRESS",
			name:     "BARBELL_BENCH_PRESS",
			reps:     12,
			weightMg: 20000,
			duration: nil,
			restSecs: nil,
		},
		{
			input:    "lunge/dumbbell_lunge:20@24kg",
			category: "LUNGE",
			name:     "DUMBBELL_LUNGE",
			reps:     20,
			weightMg: 24000,
			duration: nil,
			restSecs: nil,
		},
		{
			input:    "PULL_UP/WIDE_GRIP_LAT_PULLDOWN:12@41.5",
			category: "PULL_UP",
			name:     "WIDE_GRIP_LAT_PULLDOWN",
			reps:     12,
			weightMg: 41500,
			duration: nil,
			restSecs: nil,
		},
		{
			input:    "SQUAT/BODYWEIGHT_SQUAT:15@0",
			category: "SQUAT",
			name:     "BODYWEIGHT_SQUAT",
			reps:     15,
			weightMg: 0,
			duration: nil,
			restSecs: nil,
		},
		{
			input:    "BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:d30:r60",
			category: "BENCH_PRESS",
			name:     "BARBELL_BENCH_PRESS",
			reps:     12,
			weightMg: 20000,
			duration: ptrFloat64(30),
			restSecs: ptrInt(60),
		},
		{
			input:    "BENCH_PRESS/BARBELL_BENCH_PRESS:10@25:r45:d25",
			category: "BENCH_PRESS",
			name:     "BARBELL_BENCH_PRESS",
			reps:     10,
			weightMg: 25000,
			duration: ptrFloat64(25),
			restSecs: ptrInt(45),
		},
		{
			input:    "BENCH_PRESS/BARBELL_BENCH_PRESS:8@30:d20",
			category: "BENCH_PRESS",
			name:     "BARBELL_BENCH_PRESS",
			reps:     8,
			weightMg: 30000,
			duration: ptrFloat64(20),
			restSecs: nil,
		},
		{
			input:    "BENCH_PRESS/BARBELL_BENCH_PRESS:8@30:r60",
			category: "BENCH_PRESS",
			name:     "BARBELL_BENCH_PRESS",
			reps:     8,
			weightMg: 30000,
			duration: ptrFloat64(0),
			restSecs: ptrInt(60),
		},
		{
			input:    "BENCH_PRESS/BARBELL_BENCH_PRESS:8@30:d:r",
			category: "BENCH_PRESS",
			name:     "BARBELL_BENCH_PRESS",
			reps:     8,
			weightMg: 30000,
			duration: ptrFloat64(0),
			restSecs: ptrInt(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parsed, err := parseOneExercise(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			set := parsed.set
			if set.SetType != "ACTIVE" {
				t.Errorf("setType = %q, want ACTIVE", set.SetType)
			}
			if len(set.Exercises) != 1 {
				t.Fatalf("exercises count = %d, want 1", len(set.Exercises))
			}
			if set.Exercises[0].Category != tt.category {
				t.Errorf("category = %q, want %q", set.Exercises[0].Category, tt.category)
			}
			if set.Exercises[0].Name != tt.name {
				t.Errorf("name = %q, want %q", set.Exercises[0].Name, tt.name)
			}
			if *set.RepetitionCount != tt.reps {
				t.Errorf("reps = %d, want %d", *set.RepetitionCount, tt.reps)
			}
			if set.Weight != tt.weightMg {
				t.Errorf("weight = %f, want %f", set.Weight, tt.weightMg)
			}
			if tt.duration == nil {
				if set.Duration != nil {
					t.Errorf("duration = %v, want nil", set.Duration)
				}
			} else {
				if set.Duration == nil || *set.Duration != *tt.duration {
					t.Errorf("duration = %v, want %v", set.Duration, *tt.duration)
				}
			}
			if tt.restSecs == nil {
				if parsed.restSecs != nil {
					t.Errorf("restSecs = %v, want nil", parsed.restSecs)
				}
			} else {
				if parsed.restSecs == nil || *parsed.restSecs != *tt.restSecs {
					t.Errorf("restSecs = %v, want %v", parsed.restSecs, *tt.restSecs)
				}
			}
		})
	}
}

func TestParseOneExercise_Invalid(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"BENCH_PRESS", "expected format"},
		{"BENCH_PRESS:12@20", "expected CATEGORY/NAME"},
		{"BENCH_PRESS/BARBELL:abc@20", "invalid rep count"},
		{"BENCH_PRESS/BARBELL:12@abc", "invalid weight"},
		{"BENCH_PRESS/BARBELL:12@20:dabc", "invalid duration"},
		{"BENCH_PRESS/BARBELL:12@20:r-5", "invalid rest"},
		{"BENCH_PRESS/BARBELL:12@20:d10:d20", "duplicate :d"},
		{"BENCH_PRESS/BARBELL:12@20:r10:r20", "duplicate :r"},
		{"BENCH_PRESS/BARBELL:12@20:x10", "unknown suffix"},
		{"BENCH_PRESS/BARBELL:12@20:d1:r2:extra", "expected format"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parseOneExercise(tt.input)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.want)
			}
		})
	}
}

func TestParseExerciseSets_PerSetRest(t *testing.T) {
	exercises := []string{
		"BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:d30:r60",
		"BENCH_PRESS/BARBELL_BENCH_PRESS:10@25:d25:r45",
		"BENCH_PRESS/BARBELL_BENCH_PRESS:8@30:d20",
	}

	sets, err := parseExerciseSets(exercises)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3 active + 2 rest = 5 sets total.
	if len(sets) != 5 {
		t.Fatalf("sets count = %d, want 5", len(sets))
	}
	if sets[0].SetType != "ACTIVE" {
		t.Errorf("set[0].setType = %q, want ACTIVE", sets[0].SetType)
	}
	if sets[0].Duration == nil || *sets[0].Duration != 30 {
		t.Errorf("set[0].duration = %v, want 30", sets[0].Duration)
	}
	if sets[1].SetType != "REST" {
		t.Errorf("set[1].setType = %q, want REST", sets[1].SetType)
	}
	if sets[1].Duration == nil || *sets[1].Duration != 60 {
		t.Errorf("set[1].rest duration = %v, want 60", sets[1].Duration)
	}
	if sets[2].SetType != "ACTIVE" {
		t.Errorf("set[2].setType = %q, want ACTIVE", sets[2].SetType)
	}
	if sets[3].SetType != "REST" {
		t.Errorf("set[3].setType = %q, want REST", sets[3].SetType)
	}
	if sets[3].Duration == nil || *sets[3].Duration != 45 {
		t.Errorf("set[3].rest duration = %v, want 45", sets[3].Duration)
	}
	if sets[4].SetType != "ACTIVE" {
		t.Errorf("set[4].setType = %q, want ACTIVE", sets[4].SetType)
	}
	if sets[4].Duration == nil || *sets[4].Duration != 20 {
		t.Errorf("set[4].duration = %v, want 20", sets[4].Duration)
	}
}

func TestParseExerciseSets_NoSuffixes(t *testing.T) {
	exercises := []string{
		"BENCH_PRESS/BARBELL_BENCH_PRESS:12@20",
		"BENCH_PRESS/BARBELL_BENCH_PRESS:10@20",
	}

	sets, err := parseExerciseSets(exercises)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sets) != 2 {
		t.Fatalf("sets count = %d, want 2", len(sets))
	}
	if sets[0].Duration != nil {
		t.Errorf("set[0].duration = %v, want nil", sets[0].Duration)
	}
}

func TestParseExerciseSets_LastExerciseWithRest(t *testing.T) {
	exercises := []string{
		"BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:r60",
	}

	sets, err := parseExerciseSets(exercises)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1 active + 1 rest = 2 (REST inserted even for last exercise).
	if len(sets) != 2 {
		t.Fatalf("sets count = %d, want 2", len(sets))
	}
	if sets[1].SetType != "REST" {
		t.Errorf("set[1].setType = %q, want REST", sets[1].SetType)
	}
}

func TestExerciseSets_RejectsBadValues(t *testing.T) {
	tests := []struct {
		exercise string
		want     string
	}{
		{"A/B:12@inf", `exercise 1 ("A/B:12@inf"): invalid weight "inf": out of range`},
		{"A/B:12@nan", `exercise 1 ("A/B:12@nan"): invalid weight "nan": out of range`},
		{"A/B:12@-Inf", `exercise 1 ("A/B:12@-Inf"): invalid weight "-Inf": out of range`},
		{"A/B:12@1.7976931348623157e308", `exercise 1 ("A/B:12@1.7976931348623157e308"): invalid weight "1.7976931348623157e308": out of range`},
		{"A/B:-1", `exercise 1 ("A/B:-1"): invalid rep count "-1": must not be negative`},
		{"A/:1", `exercise 1 ("A/:1"): expected CATEGORY/NAME before ':'`},
		{"/B:1", `exercise 1 ("/B:1"): expected CATEGORY/NAME before ':'`},
	}
	for _, tt := range tests {
		t.Run(tt.exercise, func(t *testing.T) {
			body, err := ExerciseSets(111, []string{tt.exercise})
			if err == nil || err.Error() != tt.want {
				t.Fatalf("ExerciseSets(%q) = %s, %v; want error %q", tt.exercise, body, err, tt.want)
			}
		})
	}
}

func TestExerciseSets_AcceptsZeroReps(t *testing.T) {
	body, err := ExerciseSets(111, []string{"A/B:0"})
	if err != nil {
		t.Fatalf("ExerciseSets: %v", err)
	}
	if !strings.Contains(string(body), `"repetitionCount":0`) {
		t.Errorf("body %s does not carry zero reps", body)
	}
}
