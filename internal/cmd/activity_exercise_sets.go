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
	Exercise []string `help:"Exercise set in format CATEGORY/NAME:reps@weightkg (e.g. BENCH_PRESS/BARBELL_BENCH_PRESS:12@20)." short:"e" required:""`
	Rest     int      `help:"Rest duration in seconds between sets (0 to skip)." default:"0"`
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

	sets, err := parseExerciseSets(c.Exercise, c.Rest)
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

	if err := client.SetExerciseSets(g.Context, c.ID, body); err != nil {
		return fmt.Errorf("set exercise sets: %w", err)
	}

	g.UI.Successf("Exercise sets updated for activity %s", c.ID)
	return nil
}

// parseExerciseSets parses exercise flag values into API-compatible sets.
// Format: CATEGORY/NAME:reps@weightkg
// Example: BENCH_PRESS/BARBELL_BENCH_PRESS:12@20
func parseExerciseSets(exercises []string, restSecs int) ([]exerciseSet, error) {
	var sets []exerciseSet

	for i, ex := range exercises {
		parsed, err := parseOneExercise(ex)
		if err != nil {
			return nil, fmt.Errorf("exercise %d (%q): %w", i+1, ex, err)
		}
		sets = append(sets, parsed)

		// Add rest between sets if requested (not after the last set).
		if restSecs > 0 && i < len(exercises)-1 {
			dur := float64(restSecs)
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

// parseOneExercise parses a single exercise string.
// Format: CATEGORY/NAME:reps@weightkg
func parseOneExercise(s string) (exerciseSet, error) {
	// Split on ':' to separate exercise identity from set details.
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return exerciseSet{}, fmt.Errorf("expected format CATEGORY/NAME:reps@weightkg")
	}

	exercisePart := parts[0]
	detailPart := parts[1]

	// Parse category/name.
	exParts := strings.SplitN(exercisePart, "/", 2)
	if len(exParts) != 2 {
		return exerciseSet{}, fmt.Errorf("expected CATEGORY/NAME before ':'")
	}
	category := strings.ToUpper(exParts[0])
	name := strings.ToUpper(exParts[1])

	// Parse reps@weight.
	var reps int
	var weightKg float64

	atParts := strings.SplitN(detailPart, "@", 2)
	reps, err := strconv.Atoi(atParts[0])
	if err != nil {
		return exerciseSet{}, fmt.Errorf("invalid rep count %q", atParts[0])
	}

	if len(atParts) == 2 {
		weightStr := strings.TrimSuffix(atParts[1], "kg")
		weightKg, err = strconv.ParseFloat(weightStr, 64)
		if err != nil {
			return exerciseSet{}, fmt.Errorf("invalid weight %q", atParts[1])
		}
	}

	// Garmin API expects weight in milligrams.
	weightMg := weightKg * 1000

	return exerciseSet{
		Exercises: []exerciseRef{
			{Probability: 100, Category: category, Name: name},
		},
		RepetitionCount: &reps,
		Weight:          weightMg,
		SetType:         "ACTIVE",
	}, nil
}
