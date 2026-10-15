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

func TestActivityExerciseSetsSet_BadInputFailsBeforeCredentials(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{
			name:       "bad exercise",
			args:       []string{"activity", "exercise-sets", "set", "--exercise", "nope", "111"},
			wantStderr: "exercise 1 (\"nope\"): expected format CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS]\n",
		},
		{
			name:       "bad activity ID before bad exercise",
			args:       []string{"activity", "exercise-sets", "set", "--exercise", "nope", "abc"},
			wantStderr: "invalid activity ID: strconv.ParseInt: parsing \"abc\": invalid syntax\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runBadInput(t, tt.args, tt.wantStderr)
		})
	}
}
