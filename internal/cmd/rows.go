package cmd

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/outfmt"
	"github.com/bpauli/gccli/internal/units"
)

// Format is how a Rows column renders the value at its Path.
type Format uint8

const (
	// Text prints a string as is, a whole number as an integer and any
	// other number with %g. A missing value is "".
	Text Format = iota
	// DateOnly prints the text before the first space of a date-time.
	DateOnly
	// Distance prints meters in the unit system, or "-" for 0.
	Distance
	// Duration prints seconds as H:MM:SS or M:SS, or "-" for 0.
	Duration
	// Count prints a number as an integer, or "-" for 0.
	Count
)

var formatNames = [...]string{Text: "Text", DateOnly: "DateOnly", Distance: "Distance", Duration: "Duration", Count: "Count"}

func (f Format) String() string {
	if int(f) < len(formatNames) {
		return formatNames[f]
	}
	return fmt.Sprintf("Format(%d)", uint8(f))
}

func (f Format) cell(v any, sys units.System) string {
	switch f {
	case DateOnly:
		return formatDate(textOf(v))
	case Distance:
		return sys.Distance(numberOf(v))
	case Duration:
		return formatDuration(numberOf(v))
	case Count:
		return formatCalories(numberOf(v))
	}
	return textOf(v)
}

func (r Rows) print(g *Globals, client *garminapi.Client, data json.RawMessage) error {
	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(g.Stdout, data)
	}
	items, err := r.items(data)
	if err != nil {
		return fmt.Errorf("parse %s: %w", r.Noun, err)
	}
	sys := units.Metric
	if slices.ContainsFunc(r.Columns, func(c Column) bool { return c.Format == Distance }) {
		sys = resolveUnits(g, client)
	}
	header := make([]string, len(r.Columns))
	for i, c := range r.Columns {
		header[i] = c.Label
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := make([]string, len(r.Columns))
		for i, c := range r.Columns {
			row[i] = c.Format.cell(valueAt(item, c.Path), sys)
		}
		rows = append(rows, row)
	}
	return outfmt.WriteRows(g.Context, g.Stdout, header, rows)
}

// items reads the list. A root array must hold objects. Under a wrapper
// key, a missing key or a value that is not a list gives no rows, and an
// element that is not an object is skipped.
func (r Rows) items(data json.RawMessage) ([]map[string]any, error) {
	if r.Items == "" {
		var items []map[string]any
		err := json.Unmarshal(data, &items)
		return items, err
	}
	var wrapper map[string]any
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}
	list, _ := wrapper[r.Items].([]any)
	var items []map[string]any
	for _, v := range list {
		if m, ok := v.(map[string]any); ok {
			items = append(items, m)
		}
	}
	return items, nil
}

// valueAt follows a dotted path through nested objects. A step that is
// missing or not an object gives nil.
func valueAt(m map[string]any, path string) any {
	key, rest, nested := strings.Cut(path, ".")
	if !nested {
		return m[key]
	}
	next, _ := m[key].(map[string]any)
	return valueAt(next, rest)
}

func textOf(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	}
	return fmt.Sprintf("%v", v)
}

func numberOf(v any) float64 {
	f, _ := v.(float64)
	return f
}
