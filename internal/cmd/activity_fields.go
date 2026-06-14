package cmd

import "fmt"

// summaryField describes a single field that can appear in an activity summary.
type summaryField struct {
	Key    string
	Label  string
	Format func(summary, activity map[string]any) string
}

// fieldRegistry maps field keys to their definitions.
var fieldRegistry = map[string]summaryField{
	"distance": {
		Key: "distance", Label: "DISTANCE",
		Format: func(s, _ map[string]any) string { return formatDistance(jsonFloat(s, "distance")) },
	},
	"duration": {
		Key: "duration", Label: "DURATION",
		Format: func(s, _ map[string]any) string { return formatDuration(jsonFloat(s, "duration")) },
	},
	"avg_speed": {
		Key: "avg_speed", Label: "AVG SPEED",
		Format: func(s, _ map[string]any) string { return formatSpeed(jsonFloat(s, "averageSpeed")) },
	},
	"avg_pace": {
		Key: "avg_pace", Label: "AVG PACE",
		Format: func(s, _ map[string]any) string { return formatPace(jsonFloat(s, "averageSpeed")) },
	},
	"elevation_gain": {
		Key: "elevation_gain", Label: "ELEVATION",
		Format: func(s, _ map[string]any) string { return formatElevation(jsonFloat(s, "elevationGain")) },
	},
	"elevation_loss": {
		Key: "elevation_loss", Label: "ELEVATION LOSS",
		Format: func(s, _ map[string]any) string { return formatElevation(jsonFloat(s, "elevationLoss")) },
	},
	"avg_power": {
		Key: "avg_power", Label: "AVG POWER",
		Format: func(s, _ map[string]any) string { return formatPower(jsonFloat(s, "averagePower")) },
	},
	"avg_hr": {
		Key: "avg_hr", Label: "AVG HR",
		Format: func(s, _ map[string]any) string { return formatHeartRate(jsonFloat(s, "averageHR")) },
	},
	"max_hr": {
		Key: "max_hr", Label: "MAX HR",
		Format: func(s, _ map[string]any) string { return formatHeartRate(jsonFloat(s, "maxHR")) },
	},
	"calories": {
		Key: "calories", Label: "CALORIES",
		Format: func(s, _ map[string]any) string { return formatCalories(jsonFloat(s, "calories")) },
	},
	"sets": {
		Key: "sets", Label: "SETS",
		Format: func(s, _ map[string]any) string { return formatCount(jsonFloat(s, "activeSets")) },
	},
	"reps": {
		Key: "reps", Label: "REPS",
		Format: func(s, _ map[string]any) string { return formatCount(jsonFloat(s, "totalExerciseReps")) },
	},
	"aerobic_training_effect": {
		Key: "aerobic_training_effect", Label: "AEROBIC TRAINING EFFECT",
		Format: func(s, _ map[string]any) string { return formatDecimal(jsonFloat(s, "trainingEffect")) },
	},
	"anaerobic_training_effect": {
		Key: "anaerobic_training_effect", Label: "ANAEROBIC TRAINING EFFECT",
		Format: func(s, _ map[string]any) string { return formatDecimal(jsonFloat(s, "anaerobicTrainingEffect")) },
	},
	"feel": {
		Key: "feel", Label: "FEEL",
		Format: func(s, _ map[string]any) string { return formatWorkoutFeel(s) },
	},
	"rpe": {
		Key: "rpe", Label: "RPE",
		Format: func(s, _ map[string]any) string { return formatRPE(jsonFloat(s, "directWorkoutRpe")) },
	},
}

// categoryDefaults maps activity category names to their default field keys.
var categoryDefaults = map[string][]string{
	"cycling":           {"distance", "duration", "avg_speed", "elevation_gain", "elevation_loss", "avg_power", "aerobic_training_effect", "anaerobic_training_effect", "feel", "rpe"},
	"running":           {"distance", "duration", "avg_pace", "elevation_gain", "elevation_loss", "avg_hr", "aerobic_training_effect", "anaerobic_training_effect", "feel", "rpe"},
	"fitness_equipment": {"duration", "sets", "reps", "avg_hr", "calories", "aerobic_training_effect", "anaerobic_training_effect", "feel", "rpe"},
	"swimming":          {"distance", "duration", "avg_pace", "calories", "avg_hr", "aerobic_training_effect", "anaerobic_training_effect", "feel", "rpe"},
	"hiking":            {"distance", "duration", "avg_pace", "elevation_gain", "elevation_loss", "calories", "aerobic_training_effect", "anaerobic_training_effect", "feel", "rpe"},
	"multi_sport":       {"duration", "distance", "calories", "elevation_gain", "elevation_loss", "avg_speed", "aerobic_training_effect", "anaerobic_training_effect", "feel", "rpe"},
	"winter_sports":     {"distance", "duration", "avg_speed", "elevation_gain", "elevation_loss", "calories", "aerobic_training_effect", "anaerobic_training_effect", "feel", "rpe"},
	"water_sports":      {"distance", "duration", "avg_speed", "avg_hr", "calories", "aerobic_training_effect", "anaerobic_training_effect", "feel", "rpe"},
	"other":             {"distance", "duration", "avg_speed", "avg_hr", "calories", "aerobic_training_effect", "anaerobic_training_effect", "feel", "rpe"},
}

// parentTypeCategories maps Garmin parentTypeId to category name.
var parentTypeCategories = map[float64]string{
	2:  "cycling",
	4:  "swimming",
	17: "running",
	19: "winter_sports",
	29: "fitness_equipment",
	31: "water_sports",
}

// typeKeyCategories maps specific typeKey values to category names
// for types that aren't covered by parentTypeId.
var typeKeyCategories = map[string]string{
	"hiking":      "hiking",
	"multi_sport": "multi_sport",
}

// resolveCategory determines the activity category from the activity data.
// It checks parentTypeId first, then typeKey, then falls back to "other".
func resolveCategory(a map[string]any) string {
	if dto, ok := a["activityTypeDTO"].(map[string]any); ok {
		if pid := jsonFloat(dto, "parentTypeId"); pid != 0 {
			if cat, ok := parentTypeCategories[pid]; ok {
				return cat
			}
		}
		if tk := jsonString(dto, "typeKey"); tk != "" {
			if cat, ok := typeKeyCategories[tk]; ok {
				return cat
			}
		}
	}
	// Fallback for list-endpoint shape (activityType instead of activityTypeDTO).
	if at, ok := a["activityType"].(map[string]any); ok {
		if pid := jsonFloat(at, "parentTypeId"); pid != 0 {
			if cat, ok := parentTypeCategories[pid]; ok {
				return cat
			}
		}
		tk := jsonString(at, "typeKey")
		if cat, ok := typeKeyCategories[tk]; ok {
			return cat
		}
	}
	return "other"
}

// resolveFields returns the ordered field list for an activity, checking
// config overrides first, then built-in category defaults.
func resolveFields(category string, overrides map[string][]string) []summaryField {
	keys := categoryDefaults["other"]
	if catKeys, ok := categoryDefaults[category]; ok {
		keys = catKeys
	}
	if overrides != nil {
		if ovr, ok := overrides[category]; ok {
			keys = ovr
		}
	}

	fields := make([]summaryField, 0, len(keys))
	for _, k := range keys {
		if f, ok := fieldRegistry[k]; ok {
			fields = append(fields, f)
		}
	}
	return fields
}

// formatSpeed converts m/s to km/h.
func formatSpeed(mps float64) string {
	if mps == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f km/h", mps*3.6)
}

// formatPace converts m/s to min:sec /km.
func formatPace(mps float64) string {
	if mps == 0 {
		return "-"
	}
	paceSeconds := 1000.0 / mps
	mins := int(paceSeconds) / 60
	secs := int(paceSeconds) % 60
	return fmt.Sprintf("%d:%02d /km", mins, secs)
}

// formatElevation formats elevation gain in meters.
func formatElevation(meters float64) string {
	if meters == 0 {
		return "-"
	}
	return fmt.Sprintf("%d m", int(meters))
}

// formatPower formats power in watts.
func formatPower(watts float64) string {
	if watts == 0 {
		return "-"
	}
	return fmt.Sprintf("%d W", int(watts))
}

// formatCount formats an integer count.
func formatCount(n float64) string {
	if n == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", int(n))
}

// formatDecimal formats a decimal value to one fractional digit.
func formatDecimal(n float64) string {
	if n == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f", n)
}

// formatRPE converts Garmin's 0-100 workout RPE to a 0-10 scale.
func formatRPE(n float64) string {
	if n == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f", n/10)
}

// formatWorkoutFeel maps Garmin's five-level workout feel scale to labels.
func formatWorkoutFeel(summary map[string]any) string {
	value, ok := summary["directWorkoutFeel"]
	if !ok || value == nil {
		return "-"
	}
	n, ok := value.(float64)
	if !ok {
		return "-"
	}

	switch n {
	case 0:
		return "VERY WEAK"
	case 25:
		return "WEAK"
	case 50:
		return "NORMAL"
	case 75:
		return "STRONG"
	case 100:
		return "VERY STRONG"
	default:
		return fmt.Sprintf("%g", n)
	}
}
