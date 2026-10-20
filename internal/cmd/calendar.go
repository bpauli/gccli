package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
)

var calendarGroup = Group{Name: "calendar", Help: "Manage the Garmin Connect calendar.", Children: []Node{
	calendarNotesGroup,
}}

var calendarNotesGroup = Group{Name: "notes", Help: "Add, change, move and delete calendar notes.", Children: []Node{
	Custom{Name: "add", Help: "Add a note to a calendar day.", Cmd: &CalendarNotesAddCmd{}},
	Custom{Name: "update", Help: "Change the name or text of a calendar note.", Cmd: &CalendarNotesUpdateCmd{}},
	Leaf{
		Name: "move", Help: "Move a calendar note to another day.",
		Spec: argsOf(&struct {
			ID   ID   `arg:"" help:"Note ID."`
			Date Date `arg:"" help:"New date (YYYY-MM-DD, today, yesterday, 3d, +30d)."`
		}{}),
		Endpoint: "calendar.note.move",
		Params:   Params{"noteID": In("ID"), "date": In("Date")},
		Verb:     "move note",
		Output:   Write{Done: "Moved note {ID} to {Date}"},
	},
	Leaf{
		Name: "delete", Help: "Delete a calendar note.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Note ID."`
		}{}),
		Endpoint: "calendar.note.delete",
		Params:   Params{"noteID": In("ID")},
		Verb:     "delete note",
		Output:   Write{Done: "Deleted note {ID}", Confirm: "Delete note {ID}?"},
	},
}}

// CalendarNotesAddCmd adds a note to a calendar day.
type CalendarNotesAddCmd struct {
	Date    Date   `arg:"" help:"Date of the note (YYYY-MM-DD, today, yesterday, 3d, +30d)."`
	Name    string `arg:"" help:"Note name."`
	Content string `help:"Note text."`
}

func (c *CalendarNotesAddCmd) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("the note name must not be empty")
	}
	return nil
}

func (c *CalendarNotesAddCmd) Run(g *Globals) error {
	note := map[string]any{"noteName": c.Name, "date": string(c.Date)}
	if c.Content != "" {
		note["content"] = c.Content
	}
	body, err := json.Marshal(note)
	if err != nil {
		return err
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "calendar.note.create", nil, body)
	if err != nil {
		return fmt.Errorf("add note: %w", err)
	}
	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	if id := jsonString(resp, "id"); id != "" {
		return reportWrite(g, data, "Added note %q on %s (ID: %s)", c.Name, c.Date, id)
	}
	return reportWrite(g, data, "Added note %q on %s", c.Name, c.Date)
}

// CalendarNotesUpdateCmd changes the name or text of a calendar note.
type CalendarNotesUpdateCmd struct {
	ID      ID      `arg:"" help:"Note ID."`
	Name    *string `help:"New name." placeholder:"STRING"`
	Content *string `help:"New text. An empty text clears it." placeholder:"STRING"`
}

func (c *CalendarNotesUpdateCmd) Validate() error {
	if c.Name == nil && c.Content == nil {
		return errors.New("nothing to change: give --name or --content")
	}
	if c.Name != nil && strings.TrimSpace(*c.Name) == "" {
		return errors.New("the note name must not be empty")
	}
	return nil
}

func (c *CalendarNotesUpdateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.UpdateNote(g.Context, string(c.ID), garminapi.NoteEdit{Name: c.Name, Content: c.Content})
	if err != nil {
		return fmt.Errorf("update note: %w", err)
	}
	return reportWrite(g, data, "Updated note %s", c.ID)
}
