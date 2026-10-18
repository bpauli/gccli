package cmd

import (
	"fmt"
)

var nutritionGroup = Group{Name: "nutrition", Help: "View nutrition data.", Children: []Node{
	Custom{Name: "food-log", Help: "Show daily food log for a date.", Default: true, Cmd: &NutritionFoodLogCmd{}},
	Custom{Name: "meals", Help: "Show meal summaries for a date.", Cmd: &NutritionMealsCmd{}},
	Custom{Name: "settings", Help: "Show nutrition settings for a date.", Cmd: &NutritionSettingsCmd{}},
}}

// NutritionFoodLogCmd shows nutrition food log data for a date.
type NutritionFoodLogCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *NutritionFoodLogCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "nutrition.daily-food-log", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get nutrition food log: %w", err)
	}

	return writeHealthJSON(g, data)
}

// NutritionMealsCmd shows nutrition meal summary data for a date.
type NutritionMealsCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *NutritionMealsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "nutrition.daily-meals", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get nutrition meals: %w", err)
	}

	return writeHealthJSON(g, data)
}

// NutritionSettingsCmd shows nutrition settings for a date.
type NutritionSettingsCmd struct {
	Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
}

func (c *NutritionSettingsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	date, err := resolveDate(c.Date)
	if err != nil {
		return err
	}

	data, err := client.Call(g.Context, "nutrition.daily-settings", map[string]string{"date": date}, nil)
	if err != nil {
		return fmt.Errorf("get nutrition settings: %w", err)
	}

	return writeHealthJSON(g, data)
}
