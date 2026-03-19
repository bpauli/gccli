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

func TestParseOneExercise_Valid(t *testing.T) {
	tests := []struct {
		input    string
		category string
		name     string
		reps     int
		weightMg float64
	}{
		{
			input:    "BENCH_PRESS/BARBELL_BENCH_PRESS:12@20",
			category: "BENCH_PRESS",
			name:     "BARBELL_BENCH_PRESS",
			reps:     12,
			weightMg: 20000,
		},
		{
			input:    "lunge/dumbbell_lunge:20@24kg",
			category: "LUNGE",
			name:     "DUMBBELL_LUNGE",
			reps:     20,
			weightMg: 24000,
		},
		{
			input:    "PULL_UP/WIDE_GRIP_LAT_PULLDOWN:12@41.5",
			category: "PULL_UP",
			name:     "WIDE_GRIP_LAT_PULLDOWN",
			reps:     12,
			weightMg: 41500,
		},
		{
			input:    "SQUAT/BODYWEIGHT_SQUAT:15@0",
			category: "SQUAT",
			name:     "BODYWEIGHT_SQUAT",
			reps:     15,
			weightMg: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			set, err := parseOneExercise(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
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

func TestParseExerciseSets_WithRest(t *testing.T) {
	exercises := []string{
		"BENCH_PRESS/BARBELL_BENCH_PRESS:12@20",
		"BENCH_PRESS/BARBELL_BENCH_PRESS:10@20",
	}

	sets, err := parseExerciseSets(exercises, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 active + 1 rest between them.
	if len(sets) != 3 {
		t.Fatalf("sets count = %d, want 3", len(sets))
	}
	if sets[0].SetType != "ACTIVE" {
		t.Errorf("set[0].setType = %q, want ACTIVE", sets[0].SetType)
	}
	if sets[1].SetType != "REST" {
		t.Errorf("set[1].setType = %q, want REST", sets[1].SetType)
	}
	if sets[1].Duration == nil || *sets[1].Duration != 60 {
		t.Errorf("rest duration = %v, want 60", sets[1].Duration)
	}
	if sets[2].SetType != "ACTIVE" {
		t.Errorf("set[2].setType = %q, want ACTIVE", sets[2].SetType)
	}
}

func TestParseExerciseSets_NoRest(t *testing.T) {
	exercises := []string{
		"BENCH_PRESS/BARBELL_BENCH_PRESS:12@20",
		"BENCH_PRESS/BARBELL_BENCH_PRESS:10@20",
	}

	sets, err := parseExerciseSets(exercises, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sets) != 2 {
		t.Fatalf("sets count = %d, want 2", len(sets))
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
			"BENCH_PRESS/BARBELL_BENCH_PRESS:12@20",
			"BENCH_PRESS/BARBELL_BENCH_PRESS:10@25",
		},
		Rest: 60,
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
