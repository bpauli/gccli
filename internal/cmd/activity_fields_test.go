package cmd

import "testing"

func TestResolveCategory_ParentTypeId(t *testing.T) {
	tests := []struct {
		name     string
		activity map[string]any
		want     string
	}{
		{
			name: "cycling by parentTypeId",
			activity: map[string]any{
				"activityTypeDTO": map[string]any{"typeKey": "road_biking", "parentTypeId": float64(2)},
			},
			want: "cycling",
		},
		{
			name: "running by parentTypeId",
			activity: map[string]any{
				"activityTypeDTO": map[string]any{"typeKey": "running", "parentTypeId": float64(17)},
			},
			want: "running",
		},
		{
			name: "swimming by parentTypeId",
			activity: map[string]any{
				"activityTypeDTO": map[string]any{"typeKey": "lap_swimming", "parentTypeId": float64(4)},
			},
			want: "swimming",
		},
		{
			name: "winter sports by parentTypeId",
			activity: map[string]any{
				"activityTypeDTO": map[string]any{"typeKey": "resort_skiing", "parentTypeId": float64(19)},
			},
			want: "winter_sports",
		},
		{
			name: "fitness equipment by parentTypeId",
			activity: map[string]any{
				"activityTypeDTO": map[string]any{"typeKey": "strength_training", "parentTypeId": float64(29)},
			},
			want: "fitness_equipment",
		},
		{
			name: "water sports by parentTypeId",
			activity: map[string]any{
				"activityTypeDTO": map[string]any{"typeKey": "kayaking", "parentTypeId": float64(31)},
			},
			want: "water_sports",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveCategory(tt.activity)
			if got != tt.want {
				t.Errorf("resolveCategory() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveCategory_TypeKey(t *testing.T) {
	tests := []struct {
		name     string
		activity map[string]any
		want     string
	}{
		{
			name: "hiking by typeKey",
			activity: map[string]any{
				"activityTypeDTO": map[string]any{"typeKey": "hiking", "parentTypeId": float64(0)},
			},
			want: "hiking",
		},
		{
			name: "multi_sport by typeKey",
			activity: map[string]any{
				"activityTypeDTO": map[string]any{"typeKey": "multi_sport", "parentTypeId": float64(0)},
			},
			want: "multi_sport",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveCategory(tt.activity)
			if got != tt.want {
				t.Errorf("resolveCategory() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveCategory_Fallback(t *testing.T) {
	tests := []struct {
		name     string
		activity map[string]any
	}{
		{"no activityTypeDTO", map[string]any{}},
		{"unknown typeKey", map[string]any{
			"activityTypeDTO": map[string]any{"typeKey": "unknown_sport"},
		}},
		{"unknown parentTypeId", map[string]any{
			"activityTypeDTO": map[string]any{"typeKey": "foo", "parentTypeId": float64(999)},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveCategory(tt.activity)
			if got != "other" {
				t.Errorf("resolveCategory() = %q, want %q", got, "other")
			}
		})
	}
}

func TestResolveCategory_ListEndpointShape(t *testing.T) {
	// List endpoint uses activityType instead of activityTypeDTO.
	activity := map[string]any{
		"activityType": map[string]any{"typeKey": "running", "parentTypeId": float64(17)},
	}
	got := resolveCategory(activity)
	if got != "running" {
		t.Errorf("resolveCategory() = %q, want %q", got, "running")
	}
}

func TestResolveFields_Defaults(t *testing.T) {
	for cat, keys := range categoryDefaults {
		fields := resolveFields(cat, nil)
		if len(fields) != len(keys) {
			t.Errorf("category %q: got %d fields, want %d", cat, len(fields), len(keys))
		}
		for i, f := range fields {
			if f.Key != keys[i] {
				t.Errorf("category %q: field %d key = %q, want %q", cat, i, f.Key, keys[i])
			}
		}
	}
}

func TestResolveFields_ConfigOverride(t *testing.T) {
	overrides := map[string][]string{
		"running": {"distance", "calories"},
	}
	fields := resolveFields("running", overrides)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	if fields[0].Key != "distance" {
		t.Errorf("field 0 = %q, want distance", fields[0].Key)
	}
	if fields[1].Key != "calories" {
		t.Errorf("field 1 = %q, want calories", fields[1].Key)
	}
}

func TestResolveFields_ConfigOverrideSkipsInvalidKeys(t *testing.T) {
	overrides := map[string][]string{
		"cycling": {"distance", "bogus_field", "duration"},
	}
	fields := resolveFields("cycling", overrides)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields (bogus skipped), got %d", len(fields))
	}
	if fields[0].Key != "distance" || fields[1].Key != "duration" {
		t.Errorf("got fields %v, want [distance duration]", fields)
	}
}

func TestResolveFields_UnknownCategoryFallsBackToOther(t *testing.T) {
	fields := resolveFields("unknown_cat", nil)
	otherFields := resolveFields("other", nil)
	if len(fields) != len(otherFields) {
		t.Fatalf("expected %d fields (other default), got %d", len(otherFields), len(fields))
	}
	for i := range fields {
		if fields[i].Key != otherFields[i].Key {
			t.Errorf("field %d: got %q, want %q", i, fields[i].Key, otherFields[i].Key)
		}
	}
}

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		mps  float64
		want string
	}{
		{0, "-"},
		{2.778, "10.0 km/h"},
		{5.556, "20.0 km/h"},
	}
	for _, tt := range tests {
		got := formatSpeed(tt.mps)
		if got != tt.want {
			t.Errorf("formatSpeed(%v) = %q, want %q", tt.mps, got, tt.want)
		}
	}
}

func TestFormatPace(t *testing.T) {
	tests := []struct {
		mps  float64
		want string
	}{
		{0, "-"},
		{2.847, "5:51 /km"}, // ~10.25 km/h
		{4.167, "3:59 /km"}, // ~15 km/h
		{3.333, "5:00 /km"}, // 12 km/h
	}
	for _, tt := range tests {
		got := formatPace(tt.mps)
		if got != tt.want {
			t.Errorf("formatPace(%v) = %q, want %q", tt.mps, got, tt.want)
		}
	}
}

func TestFormatElevation(t *testing.T) {
	tests := []struct {
		meters float64
		want   string
	}{
		{0, "-"},
		{85, "85 m"},
		{1234.5, "1234 m"},
	}
	for _, tt := range tests {
		got := formatElevation(tt.meters)
		if got != tt.want {
			t.Errorf("formatElevation(%v) = %q, want %q", tt.meters, got, tt.want)
		}
	}
}

func TestFormatPower(t *testing.T) {
	tests := []struct {
		watts float64
		want  string
	}{
		{0, "-"},
		{200, "200 W"},
		{185.7, "185 W"},
	}
	for _, tt := range tests {
		got := formatPower(tt.watts)
		if got != tt.want {
			t.Errorf("formatPower(%v) = %q, want %q", tt.watts, got, tt.want)
		}
	}
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		n    float64
		want string
	}{
		{0, "-"},
		{12, "12"},
		{3.9, "3"},
	}
	for _, tt := range tests {
		got := formatCount(tt.n)
		if got != tt.want {
			t.Errorf("formatCount(%v) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestFormatActivitySummary_CyclingCategory(t *testing.T) {
	activity := map[string]any{
		"activityName":    "Afternoon Ride",
		"activityTypeDTO": map[string]any{"typeKey": "road_biking", "parentTypeId": float64(2)},
		"summaryDTO": map[string]any{
			"startTimeLocal": "2024-06-15 14:00:00",
			"distance":       float64(40000),
			"duration":       float64(3600),
			"averageSpeed":   float64(11.111),
			"elevationGain":  float64(320),
			"averagePower":   float64(200),
		},
	}

	rows := formatActivitySummary(activity, nil)
	// 3 fixed + 5 cycling fields
	if len(rows) != 8 {
		t.Fatalf("expected 8 rows, got %d", len(rows))
	}

	expected := []struct {
		label string
		value string
	}{
		{"NAME", "Afternoon Ride"},
		{"TYPE", "road_biking"},
		{"DATE", "2024-06-15"},
		{"DISTANCE", "40.00 km"},
		{"DURATION", "1:00:00"},
		{"AVG SPEED", "40.0 km/h"},
		{"ELEVATION", "320 m"},
		{"AVG POWER", "200 W"},
	}

	for i, exp := range expected {
		if rows[i][0] != exp.label {
			t.Errorf("row %d: expected label %q, got %q", i, exp.label, rows[i][0])
		}
		if rows[i][1] != exp.value {
			t.Errorf("row %d: expected value %q, got %q", i, exp.value, rows[i][1])
		}
	}
}

func TestFormatActivitySummary_StrengthCategory(t *testing.T) {
	activity := map[string]any{
		"activityName":    "Chest Day",
		"activityTypeDTO": map[string]any{"typeKey": "strength_training", "parentTypeId": float64(29)},
		"summaryDTO": map[string]any{
			"startTimeLocal":    "2024-06-15 10:00:00",
			"duration":          float64(2700),
			"activeSets":        float64(12),
			"totalExerciseReps": float64(96),
			"averageHR":         float64(120),
			"calories":          float64(280),
		},
	}

	rows := formatActivitySummary(activity, nil)
	// 3 fixed + 5 fitness_equipment fields
	if len(rows) != 8 {
		t.Fatalf("expected 8 rows, got %d", len(rows))
	}

	expected := []struct {
		label string
		value string
	}{
		{"NAME", "Chest Day"},
		{"TYPE", "strength_training"},
		{"DATE", "2024-06-15"},
		{"DURATION", "45:00"},
		{"SETS", "12"},
		{"REPS", "96"},
		{"AVG HR", "120 bpm"},
		{"CALORIES", "280"},
	}

	for i, exp := range expected {
		if rows[i][0] != exp.label {
			t.Errorf("row %d: expected label %q, got %q", i, exp.label, rows[i][0])
		}
		if rows[i][1] != exp.value {
			t.Errorf("row %d: expected value %q, got %q", i, exp.value, rows[i][1])
		}
	}
}

func TestFormatActivitySummary_ConfigOverride(t *testing.T) {
	activity := map[string]any{
		"activityName":    "Morning Run",
		"activityTypeDTO": map[string]any{"typeKey": "running", "parentTypeId": float64(17)},
		"summaryDTO": map[string]any{
			"startTimeLocal": "2024-06-15 07:30:00",
			"distance":       float64(10000),
			"duration":       float64(3000),
			"calories":       float64(500),
		},
	}

	overrides := map[string][]string{
		"running": {"distance", "duration", "calories"},
	}
	rows := formatActivitySummary(activity, overrides)
	// 3 fixed + 3 overridden fields
	if len(rows) != 6 {
		t.Fatalf("expected 6 rows, got %d", len(rows))
	}
	if rows[3][0] != "DISTANCE" {
		t.Errorf("row 3 label = %q, want DISTANCE", rows[3][0])
	}
	if rows[4][0] != "DURATION" {
		t.Errorf("row 4 label = %q, want DURATION", rows[4][0])
	}
	if rows[5][0] != "CALORIES" {
		t.Errorf("row 5 label = %q, want CALORIES", rows[5][0])
	}
}

func TestFieldRegistry_AllCategoryDefaultKeysExist(t *testing.T) {
	for cat, keys := range categoryDefaults {
		for _, k := range keys {
			if _, ok := fieldRegistry[k]; !ok {
				t.Errorf("category %q references unknown field key %q", cat, k)
			}
		}
	}
}
