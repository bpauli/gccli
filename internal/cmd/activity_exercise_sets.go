package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ActivityExerciseSetsSetCmd sets exercise sets for a strength training activity.
type ActivityExerciseSetsSetCmd struct {
	ID       string   `arg:"" help:"Activity ID."`
	Exercise []string `help:"Exercise set in format CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS] (e.g. BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:d30:r60)." short:"e" required:""`
}

// exerciseSet represents a single exercise set for the Garmin API.
type exerciseSet struct {
	Exercises       []exerciseRef `json:"exercises"`
	RepetitionCount *int          `json:"repetitionCount"`
	Duration        *float64      `json:"duration"`
	Weight          float64       `json:"weight"`
	SetType         string        `json:"setType"`
	StartTime       *string       `json:"startTime"`
}

// exerciseRef identifies an exercise in the Garmin catalog.
type exerciseRef struct {
	Probability int    `json:"probability"`
	Category    string `json:"category"`
	Name        string `json:"name"`
}

func (c *ActivityExerciseSetsSetCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	activityID, err := strconv.ParseInt(c.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid activity ID: %w", err)
	}

	sets, err := parseExerciseSets(c.Exercise)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"activityId":   activityID,
		"exerciseSets": sets,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal exercise sets: %w", err)
	}

	if _, err := client.SetExerciseSets(g.Context, c.ID, body); err != nil {
		return fmt.Errorf("set exercise sets: %w", err)
	}

	g.UI.Successf("Exercise sets updated for activity %s", c.ID)
	return nil
}

// parseExerciseSets parses exercise flag values into API-compatible sets.
// Each exercise may include an optional :rSECS suffix for per-set rest time.
// When present, a REST set is inserted after the active set.
// Format: CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS]
// Example: BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:d30:r60
func parseExerciseSets(exercises []string) ([]exerciseSet, error) {
	var sets []exerciseSet

	for i, ex := range exercises {
		parsed, err := parseOneExercise(ex)
		if err != nil {
			return nil, fmt.Errorf("exercise %d (%q): %w", i+1, ex, err)
		}
		sets = append(sets, parsed.set)

		if parsed.restSecs != nil {
			dur := float64(*parsed.restSecs)
			sets = append(sets, exerciseSet{
				Exercises:       []exerciseRef{},
				RepetitionCount: nil,
				Duration:        &dur,
				Weight:          -1,
				SetType:         "REST",
			})
		}
	}

	return sets, nil
}

// parsedExercise holds the parsed exercise set and optional per-set rest duration.
type parsedExercise struct {
	set      exerciseSet
	restSecs *int
}

// parseOneExercise parses a single exercise string.
// Format: CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS]
func parseOneExercise(s string) (parsedExercise, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 4 {
		return parsedExercise{}, fmt.Errorf("expected format CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS]")
	}

	// Parse category/name.
	exParts := strings.SplitN(parts[0], "/", 2)
	if len(exParts) != 2 {
		return parsedExercise{}, fmt.Errorf("expected CATEGORY/NAME before ':'")
	}
	category := strings.ToUpper(exParts[0])
	name := strings.ToUpper(exParts[1])

	// Parse reps@weight.
	var reps int
	var weightKg float64

	atParts := strings.SplitN(parts[1], "@", 2)
	reps, err := strconv.Atoi(atParts[0])
	if err != nil {
		return parsedExercise{}, fmt.Errorf("invalid rep count %q", atParts[0])
	}

	if len(atParts) == 2 {
		weightStr := strings.TrimSuffix(atParts[1], "kg")
		weightKg, err = strconv.ParseFloat(weightStr, 64)
		if err != nil {
			return parsedExercise{}, fmt.Errorf("invalid weight %q", atParts[1])
		}
	}

	// Parse optional :dSECS and :rSECS suffixes.
	var duration *float64
	var restSecs *int

	for _, suffix := range parts[2:] {
		switch {
		case strings.HasPrefix(suffix, "d"):
			if duration != nil {
				return parsedExercise{}, fmt.Errorf("duplicate :d suffix")
			}
			val := suffix[1:]
			if val == "" {
				d := 0.0
				duration = &d
			} else {
				n, err := strconv.Atoi(val)
				if err != nil || n < 0 {
					return parsedExercise{}, fmt.Errorf("invalid duration value %q", val)
				}
				d := float64(n)
				duration = &d
			}
		case strings.HasPrefix(suffix, "r"):
			if restSecs != nil {
				return parsedExercise{}, fmt.Errorf("duplicate :r suffix")
			}
			val := suffix[1:]
			if val == "" {
				r := 0
				restSecs = &r
			} else {
				n, err := strconv.Atoi(val)
				if err != nil || n < 0 {
					return parsedExercise{}, fmt.Errorf("invalid rest value %q", val)
				}
				restSecs = &n
			}
		default:
			return parsedExercise{}, fmt.Errorf("unknown suffix %q, expected :d or :r", suffix)
		}
	}

	// If rest is specified but duration is not, default duration to 0.
	if restSecs != nil && duration == nil {
		d := 0.0
		duration = &d
	}

	// Garmin API expects weight in milligrams.
	weightMg := weightKg * 1000

	return parsedExercise{
		set: exerciseSet{
			Exercises:       []exerciseRef{{Probability: 100, Category: category, Name: name}},
			RepetitionCount: &reps,
			Duration:        duration,
			Weight:          weightMg,
			SetType:         "ACTIVE",
		},
		restSecs: restSecs,
	}, nil
}
