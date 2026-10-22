package cmd

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/units"
)

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
	// Weight prints grams in the unit system, or "-" for 0.
	Weight
	// Kilometers prints kilometers as a distance in the unit system, or "-"
	// for 0.
	Kilometers
	// Milliseconds prints milliseconds as H:MM:SS or M:SS, or "-" for 0.
	Milliseconds
)

var formatNames = [...]string{Text: "Text", DateOnly: "DateOnly", Distance: "Distance", Duration: "Duration", Count: "Count", Weight: "Weight", Kilometers: "Kilometers", Milliseconds: "Milliseconds"}

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
		return formatCount(numberOf(v))
	case Weight:
		return sys.Weight(numberOf(v))
	case Kilometers:
		return sys.Distance(numberOf(v) * 1000)
	case Milliseconds:
		return formatDuration(numberOf(v) / 1000)
	}
	return textOf(v)
}

func usesUnits(columns []Column) bool {
	return slices.ContainsFunc(columns, func(c Column) bool { return slices.Contains([]Format{Distance, Kilometers, Weight}, c.Format) })
}

func (r Rows) print(g *Globals, client *garminapi.Client, data json.RawMessage) error {
	header := make([]string, len(r.Columns))
	for i, c := range r.Columns {
		header[i] = c.Label
	}
	return list{noun: r.Noun, keys: []string{r.Items}, header: header, rows: func(items []map[string]any) [][]string {
		if len(items) == 0 {
			return nil
		}
		sys := units.Metric
		if usesUnits(r.Columns) {
			sys = resolveUnits(g, client)
		}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			row := make([]string, len(r.Columns))
			for i, c := range r.Columns {
				row[i] = c.Format.cell(valueAt(item, c.Path), sys)
			}
			rows = append(rows, row)
		}
		return rows
	}}.print(g, data)
}

func valueAt(m map[string]any, path string) any {
	var v any = m
	for _, key := range strings.Split(path, ".") {
		switch x := v.(type) {
		case map[string]any:
			v = x[key]
		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(x) {
				return nil
			}
			v = x[i]
		default:
			return nil
		}
	}
	return v
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
