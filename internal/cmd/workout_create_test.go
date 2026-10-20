package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/payload"
	"github.com/bpauli/gccli/internal/secrets"
)

func mustSteps(t *testing.T, values ...string) []Step {
	t.Helper()
	out := make([]Step, len(values))
	for i, v := range values {
		s, err := payload.ParseStep(v)
		if err != nil {
			t.Fatalf("ParseStep(%q): %v", v, err)
		}
		out[i] = Step(s)
	}
	return out
}

func TestWorkoutCreateCmd_Run_Table(t *testing.T) {
	server := workoutsTestServer(t)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	cmd := &WorkoutCreateCmd{
		Name:  "Test Run",
		Type:  "run",
		Steps: mustSteps(t, "warmup:1m", "run:5m@pace:5:15-5:30", "cooldown:1m"),
		Unit:  "km",
	}

	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected output, got empty buffer")
	}
}

func compactJSON(t *testing.T, s string) string {
	t.Helper()
	var b bytes.Buffer
	if err := json.Compact(&b, []byte(s)); err != nil {
		t.Fatalf("compact %q: %v", s, err)
	}
	return b.String()
}

func sentBody(t *testing.T, route string, cmd interface{ Run(*Globals) error }) string {
	t.Helper()
	var mu sync.Mutex
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Method + " " + r.URL.Path; got != route {
			t.Errorf("request %s, want %s", got, route)
		}
		mu.Lock()
		body, _ = io.ReadAll(r.Body)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	var buf bytes.Buffer
	if err := cmd.Run(testGlobals(t, &buf, outfmt.JSON, "test@example.com")); err != nil {
		t.Fatalf("Run: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	return string(body)
}

func TestWorkoutCreateCmd_SentBody(t *testing.T) {
	tests := []struct {
		name string
		cmd  WorkoutCreateCmd
		want string
	}{
		{
			name: "every step type and target in km",
			cmd: WorkoutCreateCmd{
				Name: "Mixed",
				Type: "run",
				Unit: "km",
				Steps: mustSteps(t,
					"warmup:10m@pace:5:30-6:00",
					"run:1m30s@hr:140-160",
					"interval:45s@power:250-280",
					"recovery:2m@cadence:80-90",
					"rest:30s",
					"other:1m",
					"cooldown:5m",
				),
			},
			want: `{
				"sportType": {
					"sportTypeId": 1,
					"sportTypeKey": "running"
				},
				"workoutName": "Mixed",
				"workoutSegments": [
					{
						"segmentOrder": 1,
						"sportType": {
							"sportTypeId": 1,
							"sportTypeKey": "running"
						},
						"workoutSteps": [
							{
								"endCondition": {
									"conditionTypeId": 2,
									"conditionTypeKey": "time"
								},
								"endConditionValue": 600,
								"stepOrder": 1,
								"stepType": {
									"stepTypeId": 1,
									"stepTypeKey": "warmup"
								},
								"targetType": {
									"workoutTargetTypeId": 6,
									"workoutTargetTypeKey": "pace.zone"
								},
								"targetValueOne": 3.0303030303030303,
								"targetValueTwo": 2.7777777777777777,
								"type": "ExecutableStepDTO"
							},
							{
								"endCondition": {
									"conditionTypeId": 2,
									"conditionTypeKey": "time"
								},
								"endConditionValue": 90,
								"stepOrder": 2,
								"stepType": {
									"stepTypeId": 3,
									"stepTypeKey": "interval"
								},
								"targetType": {
									"workoutTargetTypeId": 4,
									"workoutTargetTypeKey": "heart.rate.zone"
								},
								"targetValueOne": 140,
								"targetValueTwo": 160,
								"type": "ExecutableStepDTO"
							},
							{
								"endCondition": {
									"conditionTypeId": 2,
									"conditionTypeKey": "time"
								},
								"endConditionValue": 45,
								"stepOrder": 3,
								"stepType": {
									"stepTypeId": 3,
									"stepTypeKey": "interval"
								},
								"targetType": {
									"workoutTargetTypeId": 2,
									"workoutTargetTypeKey": "power.zone"
								},
								"targetValueOne": 250,
								"targetValueTwo": 280,
								"type": "ExecutableStepDTO"
							},
							{
								"endCondition": {
									"conditionTypeId": 2,
									"conditionTypeKey": "time"
								},
								"endConditionValue": 120,
								"stepOrder": 4,
								"stepType": {
									"stepTypeId": 4,
									"stepTypeKey": "recovery"
								},
								"targetType": {
									"workoutTargetTypeId": 3,
									"workoutTargetTypeKey": "cadence"
								},
								"targetValueOne": 80,
								"targetValueTwo": 90,
								"type": "ExecutableStepDTO"
							},
							{
								"endCondition": {
									"conditionTypeId": 2,
									"conditionTypeKey": "time"
								},
								"endConditionValue": 30,
								"stepOrder": 5,
								"stepType": {
									"stepTypeId": 5,
									"stepTypeKey": "rest"
								},
								"targetType": {
									"workoutTargetTypeId": 1,
									"workoutTargetTypeKey": "no.target"
								},
								"type": "ExecutableStepDTO"
							},
							{
								"endCondition": {
									"conditionTypeId": 2,
									"conditionTypeKey": "time"
								},
								"endConditionValue": 60,
								"stepOrder": 6,
								"stepType": {
									"stepTypeId": 7,
									"stepTypeKey": "other"
								},
								"targetType": {
									"workoutTargetTypeId": 1,
									"workoutTargetTypeKey": "no.target"
								},
								"type": "ExecutableStepDTO"
							},
							{
								"endCondition": {
									"conditionTypeId": 2,
									"conditionTypeKey": "time"
								},
								"endConditionValue": 300,
								"stepOrder": 7,
								"stepType": {
									"stepTypeId": 2,
									"stepTypeKey": "cooldown"
								},
								"targetType": {
									"workoutTargetTypeId": 1,
									"workoutTargetTypeKey": "no.target"
								},
								"type": "ExecutableStepDTO"
							}
						]
					}
				]
			}`,
		},
		{
			name: "pace in miles",
			cmd: WorkoutCreateCmd{
				Name:  "Miles",
				Type:  "bike",
				Unit:  "mi",
				Steps: mustSteps(t, "run:20m@pace:8:00-8:30"),
			},
			want: `{
				"sportType": {
					"sportTypeId": 2,
					"sportTypeKey": "cycling"
				},
				"workoutName": "Miles",
				"workoutSegments": [
					{
						"segmentOrder": 1,
						"sportType": {
							"sportTypeId": 2,
							"sportTypeKey": "cycling"
						},
						"workoutSteps": [
							{
								"endCondition": {
									"conditionTypeId": 2,
									"conditionTypeKey": "time"
								},
								"endConditionValue": 1200,
								"stepOrder": 1,
								"stepType": {
									"stepTypeId": 3,
									"stepTypeKey": "interval"
								},
								"targetType": {
									"workoutTargetTypeId": 6,
									"workoutTargetTypeKey": "pace.zone"
								},
								"targetValueOne": 3.3528000000000002,
								"targetValueTwo": 3.1555764705882354,
								"type": "ExecutableStepDTO"
							}
						]
					}
				]
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sentBody(t, "POST /workout-service/workout", &tt.cmd); got != compactJSON(t, tt.want) {
				t.Errorf("body\n got %s\nwant %s", got, tt.want)
			}
		})
	}
}

// runBadInput runs args with and without an account and requires a usage
// error (exit 2, the usage block, the error line) with no credential store
// opened: a bad value is checked at parse time, before the login.
func runBadInput(t *testing.T, args []string, wantErr string) {
	t.Helper()
	for _, account := range []string{harnessAccount, ""} {
		newHarness(t, harnessInput{})
		stub(t, &loadSecretsStore, func() (*secrets.Store, error) {
			t.Errorf("account %q: the credential store was opened for bad input", account)
			return nil, errors.New("no store in this test")
		})
		var stdout, stderr bytes.Buffer
		global := []string{"--color", "never"}
		if account != "" {
			global = append(global, "--account", account)
		}
		code := execute(append(global, args...), "test", "none", "never", &stdout, &stderr)
		errLine := "\ngccli: error: " + wantErr + "\n"
		if code != 2 || stdout.Len() != 0 || !strings.HasPrefix(stderr.String(), "Usage: gccli ") || !strings.HasSuffix(stderr.String(), errLine) {
			t.Errorf("account %q: exit %d, stdout %q, stderr %q; want exit 2, no stdout, the usage block and %q", account, code, stdout.String(), stderr.String(), errLine)
		}
	}
}

func TestWorkoutCreateCmd_BadStepFailsBeforeCredentials(t *testing.T) {
	runBadInput(t, []string{"workouts", "create", "--type", "run", "--step", "bad", "Foo"},
		"--step: invalid step \"bad\": expected type:duration[@target:values]")
}

func TestWorkoutCreateCmd_OutOfRangeStepFailsBeforeCredentials(t *testing.T) {
	digits400 := strings.Repeat("9", 400)
	overflowsTimes60 := "2" + strings.Repeat("0", 307)
	tests := []struct {
		name    string
		step    string
		wantErr string
	}{
		{"NaN target", "run:5m@hr:nan-160", "--step: invalid step \"run:5m@hr:nan-160\": invalid range low value \"nan\": out of range"},
		{"Inf target", "run:5m@hr:100-inf", "--step: invalid step \"run:5m@hr:100-inf\": invalid range high value \"inf\": out of range"},
		{"400-digit duration", "run:" + digits400 + "m", "--step: invalid step \"run:" + digits400 + "m\": invalid duration \"" + digits400 + "m\": out of range"},
		{"minutes overflow in seconds", "run:" + overflowsTimes60 + "m", "--step: invalid step \"run:" + overflowsTimes60 + "m\": invalid duration \"" + overflowsTimes60 + "m\": out of range"},
		{"400-digit pace", "run:5m@pace:" + digits400 + ":00-" + digits400 + ":01", "--step: invalid step \"run:5m@pace:" + digits400 + ":00-" + digits400 + ":01\": invalid pace \"" + digits400 + ":00\": out of range"},
		{"empty step", "", "workouts create: invalid step \"\": expected type:duration[@target:values]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runBadInput(t, []string{"workouts", "create", "--type", "run", "--step", tt.step, "Foo"}, tt.wantErr)
		})
	}
}
