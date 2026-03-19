package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func mockExerciseCatalog() json.RawMessage {
	return json.RawMessage(`{
		"categories": {
			"BENCH_PRESS": {
				"exercises": {
					"BARBELL_BENCH_PRESS": {
						"primaryMuscles": ["CHEST"],
						"secondaryMuscles": ["TRICEPS", "SHOULDERS"]
					},
					"DUMBBELL_BENCH_PRESS": {
						"primaryMuscles": ["CHEST"],
						"secondaryMuscles": ["TRICEPS"]
					}
				}
			},
			"SQUAT": {
				"exercises": {
					"BARBELL_SQUAT": {
						"primaryMuscles": ["QUADS", "GLUTES"],
						"secondaryMuscles": ["HAMSTRINGS"]
					}
				}
			}
		}
	}`)
}

func overrideFetchExercises(t *testing.T, data json.RawMessage) {
	t.Helper()
	orig := fetchExerciseCatalogFn
	fetchExerciseCatalogFn = func(_ context.Context) (json.RawMessage, error) {
		return data, nil
	}
	t.Cleanup(func() { fetchExerciseCatalogFn = orig })
}

func TestExercisesList_JSON(t *testing.T) {
	overrideFetchExercises(t, mockExerciseCatalog())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "")
	// No account needed for exercises list.
	g.Account = ""

	cmd := &ExercisesListCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestExercisesList_Table(t *testing.T) {
	overrideFetchExercises(t, mockExerciseCatalog())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	g.Account = ""

	cmd := &ExercisesListCmd{}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestExercisesList_FilterCategory(t *testing.T) {
	overrideFetchExercises(t, mockExerciseCatalog())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	g.Account = ""

	cmd := &ExercisesListCmd{Category: "BENCH_PRESS"}
	err := cmd.Run(g)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestExercisesList_UnknownCategory(t *testing.T) {
	overrideFetchExercises(t, mockExerciseCatalog())

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "")
	g.Account = ""

	cmd := &ExercisesListCmd{Category: "NOPE"}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error for unknown category")
	}
	if !strings.Contains(err.Error(), "unknown category") {
		t.Fatalf("unexpected error: %v", err)
	}
}
