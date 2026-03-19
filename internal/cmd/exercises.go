package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/outfmt"
)

// fetchExerciseCatalogFn is overridable for tests.
var fetchExerciseCatalogFn = garminapi.FetchExerciseCatalog

// ExercisesCmd groups exercise catalog subcommands.
type ExercisesCmd struct {
	List ExercisesListCmd `cmd:"" default:"withargs" help:"List available exercise categories and exercises."`
}

// ExercisesListCmd lists the Garmin exercise catalog.
type ExercisesListCmd struct {
	Category string `help:"Filter by category (e.g. BENCH_PRESS)." short:"c"`
}

func (c *ExercisesListCmd) Run(g *Globals) error {
	data, err := fetchExerciseCatalogFn(g.Context)
	if err != nil {
		return fmt.Errorf("list exercises: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		if c.Category == "" {
			return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
		}
		// Filter to a single category in JSON mode.
		var catalog exerciseCatalog
		if err := json.Unmarshal(data, &catalog); err != nil {
			return fmt.Errorf("parse exercise catalog: %w", err)
		}
		cat := strings.ToUpper(c.Category)
		if exercises, ok := catalog.Categories[cat]; ok {
			filtered := map[string]any{cat: exercises}
			return outfmt.WriteJSON(os.Stdout, filtered)
		}
		return fmt.Errorf("unknown category %q", c.Category)
	}

	var catalog exerciseCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("parse exercise catalog: %w", err)
	}

	if c.Category != "" {
		return c.printCategory(g, catalog, strings.ToUpper(c.Category))
	}

	return c.printAllCategories(g, catalog)
}

func (c *ExercisesListCmd) printAllCategories(g *Globals, catalog exerciseCatalog) error {
	categories := make([]string, 0, len(catalog.Categories))
	for cat := range catalog.Categories {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	rows := make([][]string, 0, len(categories))
	for _, cat := range categories {
		exercises := catalog.Categories[cat].Exercises
		rows = append(rows, []string{cat, fmt.Sprintf("%d exercises", len(exercises))})
	}

	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(os.Stdout, rows)
	}
	return outfmt.WriteTable(os.Stdout, []string{"CATEGORY", "COUNT"}, rows)
}

func (c *ExercisesListCmd) printCategory(_ *Globals, catalog exerciseCatalog, category string) error {
	cat, ok := catalog.Categories[category]
	if !ok {
		return fmt.Errorf("unknown category %q", category)
	}

	names := make([]string, 0, len(cat.Exercises))
	for name := range cat.Exercises {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		ex := cat.Exercises[name]
		muscles := append(ex.PrimaryMuscles, ex.SecondaryMuscles...)
		_, _ = fmt.Fprintf(os.Stdout, "%s/%s  %s\n", category, name, strings.Join(muscles, ", "))
	}
	return nil
}

// exerciseCatalog represents the Garmin exercise catalog JSON structure.
type exerciseCatalog struct {
	Categories map[string]exerciseCategory `json:"categories"`
}

// exerciseCategory holds exercises in a category.
type exerciseCategory struct {
	Exercises map[string]exerciseInfo `json:"exercises"`
}

// exerciseInfo holds muscle group info for an exercise.
type exerciseInfo struct {
	PrimaryMuscles   []string `json:"primaryMuscles"`
	SecondaryMuscles []string `json:"secondaryMuscles"`
}
