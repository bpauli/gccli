package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt(v int) *int             { return &v }

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

func TestActivityExerciseSetsSet_Success(t *testing.T) {
	var receivedPayload map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/activity-service/activity/99999/exerciseSets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &ActivityExerciseSetsSetCmd{
		ID: "99999",
		Exercise: []string{
			"BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:r60",
			"BENCH_PRESS/BARBELL_BENCH_PRESS:10@25",
		},
	}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if receivedPayload == nil {
		t.Fatal("expected payload to be received")
	}

	sets, ok := receivedPayload["exerciseSets"].([]any)
	if !ok {
		t.Fatal("expected exerciseSets to be an array")
	}
	// 2 active + 1 rest.
	if len(sets) != 3 {
		t.Errorf("sets count = %d, want 3", len(sets))
	}

	if !strings.Contains(buf.String(), "Exercise sets updated") {
		t.Errorf("expected success message, got: %q", buf.String())
	}
}

func TestActivityExerciseSetsSet_NoAccount(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	cmd := &ActivityExerciseSetsSetCmd{
		ID:       "99999",
		Exercise: []string{"BENCH_PRESS/BARBELL_BENCH_PRESS:12@20"},
	}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no account specified") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivityExerciseSetsSet_SentBody(t *testing.T) {
	cmd := &ActivityExerciseSetsSetCmd{
		ID: "111",
		Exercise: []string{
			"bench_press/barbell_bench_press:12@20.5kg:d30:r60",
			"SQUAT/BARBELL_BACK_SQUAT:8",
			"ROW/DUMBBELL_ROW:10@15:r",
			"PLANK/PLANK:1@0:d:r0",
		},
	}
	want := `{
		"activityId": 111,
		"exerciseSets": [
			{
				"exercises": [
					{
						"probability": 100,
						"category": "BENCH_PRESS",
						"name": "BARBELL_BENCH_PRESS"
					}
				],
				"repetitionCount": 12,
				"duration": 30,
				"weight": 20500,
				"setType": "ACTIVE",
				"startTime": null
			},
			{
				"exercises": [],
				"repetitionCount": null,
				"duration": 60,
				"weight": -1,
				"setType": "REST",
				"startTime": null
			},
			{
				"exercises": [
					{
						"probability": 100,
						"category": "SQUAT",
						"name": "BARBELL_BACK_SQUAT"
					}
				],
				"repetitionCount": 8,
				"duration": null,
				"weight": 0,
				"setType": "ACTIVE",
				"startTime": null
			},
			{
				"exercises": [
					{
						"probability": 100,
						"category": "ROW",
						"name": "DUMBBELL_ROW"
					}
				],
				"repetitionCount": 10,
				"duration": 0,
				"weight": 15000,
				"setType": "ACTIVE",
				"startTime": null
			},
			{
				"exercises": [],
				"repetitionCount": null,
				"duration": 0,
				"weight": -1,
				"setType": "REST",
				"startTime": null
			},
			{
				"exercises": [
					{
						"probability": 100,
						"category": "PLANK",
						"name": "PLANK"
					}
				],
				"repetitionCount": 1,
				"duration": 0,
				"weight": 0,
				"setType": "ACTIVE",
				"startTime": null
			},
			{
				"exercises": [],
				"repetitionCount": null,
				"duration": 0,
				"weight": -1,
				"setType": "REST",
				"startTime": null
			}
		]
	}`
	if got := sentBody(t, "PUT /activity-service/activity/111/exerciseSets", cmd); got != compactJSON(t, want) {
		t.Errorf("body\n got %s\nwant %s", got, want)
	}
}
