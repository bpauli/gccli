package payload

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ExerciseSets builds the body that replaces the exercise sets of an activity
// from parsed exercises.
func ExerciseSets(activityID int64, exercises []Exercise) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{
		"activityId":   activityID,
		"exerciseSets": exerciseSets(exercises),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal exercise sets: %w", err)
	}
	return body, nil
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

// exerciseSets turns parsed exercises into API-compatible sets. An exercise
// with a rest time gets a REST set after its active set.
func exerciseSets(exercises []Exercise) []exerciseSet {
	var sets []exerciseSet

	for _, ex := range exercises {
		sets = append(sets, ex.set)

		if ex.restSecs != nil {
			dur := float64(*ex.restSecs)
			sets = append(sets, exerciseSet{
				Exercises:       []exerciseRef{},
				RepetitionCount: nil,
				Duration:        &dur,
				Weight:          -1,
				SetType:         "REST",
			})
		}
	}

	return sets
}

// Exercise is one parsed exercise set with its optional per-set rest time.
type Exercise struct {
	set      exerciseSet
	restSecs *int
}

// ParseExercise parses one exercise value in the format
// CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS], for example
// BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:d30:r60.
func ParseExercise(s string) (Exercise, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 4 {
		return Exercise{}, fmt.Errorf("expected format CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS]")
	}

	// Parse category/name.
	exParts := strings.SplitN(parts[0], "/", 2)
	if len(exParts) != 2 || exParts[0] == "" || exParts[1] == "" {
		return Exercise{}, fmt.Errorf("expected CATEGORY/NAME before ':'")
	}
	category := strings.ToUpper(exParts[0])
	name := strings.ToUpper(exParts[1])

	// Parse reps@weight.
	var reps int
	var weightKg float64

	atParts := strings.SplitN(parts[1], "@", 2)
	reps, err := strconv.Atoi(atParts[0])
	if err != nil {
		return Exercise{}, fmt.Errorf("invalid rep count %q", atParts[0])
	}
	if reps < 0 {
		return Exercise{}, fmt.Errorf("invalid rep count %q: must not be negative", atParts[0])
	}

	if len(atParts) == 2 {
		weightStr := strings.TrimSuffix(atParts[1], "kg")
		weightKg, err = strconv.ParseFloat(weightStr, 64)
		if err != nil {
			return Exercise{}, fmt.Errorf("invalid weight %q", atParts[1])
		}
		if !finite(weightKg * 1000) {
			return Exercise{}, fmt.Errorf("invalid weight %q: out of range", atParts[1])
		}
	}

	// Parse optional :dSECS and :rSECS suffixes.
	var duration *float64
	var restSecs *int

	for _, suffix := range parts[2:] {
		switch {
		case strings.HasPrefix(suffix, "d"):
			if duration != nil {
				return Exercise{}, fmt.Errorf("duplicate :d suffix")
			}
			val := suffix[1:]
			if val == "" {
				d := 0.0
				duration = &d
			} else {
				n, err := strconv.Atoi(val)
				if err != nil || n < 0 {
					return Exercise{}, fmt.Errorf("invalid duration value %q", val)
				}
				d := float64(n)
				duration = &d
			}
		case strings.HasPrefix(suffix, "r"):
			if restSecs != nil {
				return Exercise{}, fmt.Errorf("duplicate :r suffix")
			}
			val := suffix[1:]
			if val == "" {
				r := 0
				restSecs = &r
			} else {
				n, err := strconv.Atoi(val)
				if err != nil || n < 0 {
					return Exercise{}, fmt.Errorf("invalid rest value %q", val)
				}
				restSecs = &n
			}
		default:
			return Exercise{}, fmt.Errorf("unknown suffix %q, expected :d or :r", suffix)
		}
	}

	// If rest is specified but duration is not, default duration to 0.
	if restSecs != nil && duration == nil {
		d := 0.0
		duration = &d
	}

	// Garmin API expects weight in milligrams.
	weightMg := weightKg * 1000

	return Exercise{
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
