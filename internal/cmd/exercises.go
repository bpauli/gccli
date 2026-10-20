package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/outfmt"
)

// fetchExerciseCatalogFn is overridable for tests.
var fetchExerciseCatalogFn = garminapi.FetchExerciseCatalog

var exercisesGroup = Group{Name: "exercises", Help: "Browse the exercise catalog.", Children: []Node{
	Custom{Name: "list", Help: "List available exercise categories and exercises.", Default: true, Cmd: &ExercisesListCmd{}},
}}

// ExercisesListCmd lists the Garmin exercise catalog.
type ExercisesListCmd struct {
	Category string `help:"Filter by category (e.g. BENCH_PRESS)." short:"c"`
}

func (c *ExercisesListCmd) Run(g *Globals) error {
	data, err := fetchExerciseCatalogFn(g.Context)
	if err != nil {
		return fmt.Errorf("list exercises: %w", err)
	}

	if outfmt.IsJSON(g.Context) && c.Category == "" {
		return writeObject(g, data)
	}

	// An empty catalog has no categories, so --category fails with
	// "unknown category" in every mode.
	var catalog exerciseCatalog
	if err := unmarshalAnswer(data, &catalog); err != nil {
		return fmt.Errorf("parse exercise catalog: %w", err)
	}

	if c.Category == "" {
		return c.printAllCategories(g, catalog)
	}
	cat := strings.ToUpper(c.Category)
	if outfmt.IsJSON(g.Context) {
		if exercises, ok := catalog.Categories[cat]; ok {
			filtered := map[string]any{cat: exercises}
			return outfmt.WriteJSON(g.Stdout, filtered)
		}
		return fmt.Errorf("unknown category %q", c.Category)
	}
	return c.printCategory(g, catalog, cat)
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

	return outfmt.WriteRows(g.Context, g.Stdout, []string{"CATEGORY", "COUNT"}, rows)
}

func (c *ExercisesListCmd) printCategory(g *Globals, catalog exerciseCatalog, category string) error {
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
		_, _ = fmt.Fprintf(g.Stdout, "%s/%s  %s\n", category, name, strings.Join(muscles, ", "))
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
