package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
)

// HealthSleepNoteCmd sets or removes the note of one night of sleep.
type HealthSleepNoteCmd struct {
	Date  Date     `arg:"" help:"Date of the sleep (YYYY-MM-DD, today, yesterday, 3d), as 'health sleep' takes it."`
	Text  FreeText `arg:"" optional:"" help:"Note text. It replaces the note of that night."`
	Clear bool     `help:"Remove the note instead of setting a text."`
}

func (c *HealthSleepNoteCmd) Validate() error {
	switch {
	case c.Clear && c.Text != "":
		return errors.New("give a note text or --clear, not both")
	case !c.Clear && c.Text == "":
		return errors.New("give a note text, or --clear to remove the note")
	}
	return nil
}

func (c *HealthSleepNoteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	// Garmin removes the note when the text is empty.
	body, err := json.Marshal(map[string]string{"userNote": string(c.Text)})
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "health.sleep-note", map[string]string{"date": string(c.Date)}, body)
	if err != nil {
		return fmt.Errorf("set sleep note: %w", err)
	}
	if c.Clear {
		return reportWrite(g, data, "Removed the sleep note of %s", c.Date)
	}
	return reportWrite(g, data, "Set the sleep note of %s", c.Date)
}
