// Package payload builds the JSON bodies that gccli sends to Garmin Connect
// from command input.
package payload

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/bpauli/gccli/internal/units"
)

const metersPerKm = 1000.0

// finite reports whether v can go into a JSON body. strconv.ParseFloat
// accepts "nan" and "inf", and large inputs overflow to Inf.
func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

type workoutStep struct {
	stepType       string
	durationSecs   float64
	targetType     string  // "pace", "hr", "power", "cadence", or "" for no target
	targetValueOne float64 // higher/faster value
	targetValueTwo float64 // lower/slower value
}

var stepTypeMap = map[string]struct {
	id  int
	key string
}{
	"warmup":   {1, "warmup"},
	"cooldown": {2, "cooldown"},
	"run":      {3, "interval"},
	"interval": {3, "interval"},
	"recovery": {4, "recovery"},
	"rest":     {5, "rest"},
	"other":    {7, "other"},
}

var sportTypeMap = map[string]struct {
	id  int
	key string
}{
	"run":        {1, "running"},
	"bike":       {2, "cycling"},
	"custom":     {3, "other"},
	"swim":       {4, "swimming"},
	"strength":   {5, "strength_training"},
	"cardio":     {6, "cardio_training"},
	"yoga":       {7, "yoga"},
	"pilates":    {8, "pilates"},
	"hiit":       {9, "hiit"},
	"multisport": {10, "multi_sport"},
	"mobility":   {11, "mobility"},
}

var targetTypeMap = map[string]struct {
	id  int
	key string
}{
	"pace":    {6, "pace.zone"},
	"hr":      {4, "heart.rate.zone"},
	"power":   {2, "power.zone"},
	"cadence": {3, "cadence"},
}

// durationRegexp matches durations like "1m", "30s", "1m30s".
var durationRegexp = regexp.MustCompile(`^(?:(\d+)m)?(?:(\d+)s)?$`)

// paceRegexp matches pace like "5:30".
var paceRegexp = regexp.MustCompile(`^(\d+):(\d{2})$`)

// Workout builds the body of a new workout from a sport type such as "run",
// step values such as "run:5m@pace:5:30-6:00" and a pace unit, "km" or "mi".
func Workout(name, sportType, unit string, rawSteps []string) (json.RawMessage, error) {
	steps, err := parseSteps(rawSteps, unit)
	if err != nil {
		return nil, err
	}
	body, err := buildWorkoutJSON(name, sportType, steps)
	if err != nil {
		return nil, fmt.Errorf("build workout: %w", err)
	}
	return body, nil
}

func parseSteps(raw []string, unit string) ([]workoutStep, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf(`invalid step "": expected type:duration[@target:values]`)
	}
	steps := make([]workoutStep, 0, len(raw))
	for _, s := range raw {
		step, err := parseStep(s, unit)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func parseStep(s string, unit string) (workoutStep, error) {
	// Split on first "@" to separate step part from optional target part.
	stepPart, targetPart, _ := strings.Cut(s, "@")

	// Parse step part: "type:duration"
	typeName, durStr, found := strings.Cut(stepPart, ":")
	if !found {
		return workoutStep{}, fmt.Errorf("invalid step %q: expected type:duration[@target:values]", s)
	}

	if _, ok := stepTypeMap[typeName]; !ok {
		return workoutStep{}, fmt.Errorf("invalid step type %q: expected warmup, run, interval, cooldown, recovery, rest, or other", typeName)
	}

	dur, err := parseStepDuration(durStr)
	if err != nil {
		return workoutStep{}, fmt.Errorf("invalid step %q: %w", s, err)
	}

	step := workoutStep{
		stepType:     typeName,
		durationSecs: dur,
	}

	if targetPart != "" {
		targetName, values, found := strings.Cut(targetPart, ":")
		if !found {
			return workoutStep{}, fmt.Errorf("invalid step %q: target requires values after ':'", s)
		}

		if _, ok := targetTypeMap[targetName]; !ok {
			return workoutStep{}, fmt.Errorf("invalid target type %q: expected pace, hr, power, or cadence", targetName)
		}

		step.targetType = targetName

		switch targetName {
		case "pace":
			high, low, err := parsePaceRange(values, unit)
			if err != nil {
				return workoutStep{}, fmt.Errorf("invalid step %q: %w", s, err)
			}
			step.targetValueOne = high
			step.targetValueTwo = low
		default:
			low, high, err := parseNumericRange(values)
			if err != nil {
				return workoutStep{}, fmt.Errorf("invalid step %q: %w", s, err)
			}
			step.targetValueOne = low
			step.targetValueTwo = high
		}
	}

	return step, nil
}

func parseStepDuration(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	m := durationRegexp.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid duration %q: expected format like 1m, 30s, or 1m30s", s)
	}

	var total float64
	if m[1] != "" {
		mins, _ := strconv.ParseFloat(m[1], 64)
		total += mins * 60
	}
	if m[2] != "" {
		secs, _ := strconv.ParseFloat(m[2], 64)
		total += secs
	}

	if total == 0 {
		return 0, fmt.Errorf("duration must be greater than zero")
	}
	if !finite(total) {
		return 0, fmt.Errorf("invalid duration %q: out of range", s)
	}

	return total, nil
}

// parseNumericRange parses "140-160" into (140.0, 160.0).
func parseNumericRange(s string) (float64, float64, error) {
	low, high, found := strings.Cut(s, "-")
	if !found {
		return 0, 0, fmt.Errorf("invalid range %q: expected LOW-HIGH", s)
	}

	lowVal, err := strconv.ParseFloat(low, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range low value %q: %w", low, err)
	}
	if !finite(lowVal) {
		return 0, 0, fmt.Errorf("invalid range low value %q: out of range", low)
	}

	highVal, err := strconv.ParseFloat(high, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range high value %q: %w", high, err)
	}
	if !finite(highVal) {
		return 0, 0, fmt.Errorf("invalid range high value %q: out of range", high)
	}

	if lowVal >= highVal {
		return 0, 0, fmt.Errorf("range low value %.0f must be less than high value %.0f", lowVal, highVal)
	}

	return lowVal, highVal, nil
}

// parsePaceRange parses "5:30-6:00" into (fastMPS, slowMPS).
func parsePaceRange(s string, unit string) (float64, float64, error) {
	idx := findPaceSeparator(s)
	if idx < 0 {
		return 0, 0, fmt.Errorf("invalid pace range %q: expected format like 5:30-6:00", s)
	}

	fastStr := s[:idx]
	slowStr := s[idx+1:]

	fastMPS, err := parsePaceToMPS(fastStr, unit)
	if err != nil {
		return 0, 0, err
	}

	slowMPS, err := parsePaceToMPS(slowStr, unit)
	if err != nil {
		return 0, 0, err
	}

	if fastMPS < slowMPS {
		return 0, 0, fmt.Errorf("fast pace %s must be faster (lower) than slow pace %s", fastStr, slowStr)
	}

	return fastMPS, slowMPS, nil
}

// findPaceSeparator finds the hyphen index in a pace range like "5:30-6:00".
// Returns -1 if not found.
func findPaceSeparator(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '-' {
			left := s[:i]
			right := s[i+1:]
			if paceRegexp.MatchString(left) && paceRegexp.MatchString(right) {
				return i
			}
		}
	}
	return -1
}

func parsePaceToMPS(s string, unit string) (float64, error) {
	m := paceRegexp.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid pace %q: expected format like 5:30", s)
	}

	mins, _ := strconv.ParseFloat(m[1], 64)
	secs, _ := strconv.ParseFloat(m[2], 64)
	totalSecs := mins*60 + secs

	if totalSecs == 0 {
		return 0, fmt.Errorf("pace must be greater than zero")
	}
	if !finite(totalSecs) {
		return 0, fmt.Errorf("invalid pace %q: out of range", s)
	}

	dist := metersPerKm
	if unit == "mi" {
		dist = units.MetersPerMile
	}

	return dist / totalSecs, nil
}

func buildWorkoutJSON(name string, sportType string, steps []workoutStep) (json.RawMessage, error) {
	sport, ok := sportTypeMap[sportType]
	if !ok {
		return nil, fmt.Errorf("unknown sport type: %s", sportType)
	}

	sportObj := map[string]any{
		"sportTypeId":  sport.id,
		"sportTypeKey": sport.key,
	}

	workoutSteps := make([]map[string]any, 0, len(steps))

	for i, s := range steps {
		st := stepTypeMap[s.stepType]

		step := map[string]any{
			"type":      "ExecutableStepDTO",
			"stepOrder": i + 1,
			"stepType": map[string]any{
				"stepTypeId":  st.id,
				"stepTypeKey": st.key,
			},
			"endCondition": map[string]any{
				"conditionTypeId":  2,
				"conditionTypeKey": "time",
			},
			"endConditionValue": s.durationSecs,
		}

		if s.targetType != "" {
			tt := targetTypeMap[s.targetType]
			step["targetType"] = map[string]any{
				"workoutTargetTypeId":  tt.id,
				"workoutTargetTypeKey": tt.key,
			}
			step["targetValueOne"] = s.targetValueOne
			step["targetValueTwo"] = s.targetValueTwo
		} else {
			step["targetType"] = map[string]any{
				"workoutTargetTypeId":  1,
				"workoutTargetTypeKey": "no.target",
			}
		}

		workoutSteps = append(workoutSteps, step)
	}

	payload := map[string]any{
		"workoutName": name,
		"sportType":   sportObj,
		"workoutSegments": []map[string]any{
			{
				"segmentOrder": 1,
				"sportType":    sportObj,
				"workoutSteps": workoutSteps,
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(data), nil
}
