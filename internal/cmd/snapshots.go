package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
)

var healthSnapshotsGroup = Group{Name: "snapshots", Help: "List, show and edit Health Snapshot sessions.", Children: []Node{
	Leaf{
		Name: "list", Help: "List Health Snapshot sessions up to a date.", Default: true,
		Spec: argsOf(&struct {
			Pagination `embed:"" set:"items=sessions"`
			Until      Date `help:"Last date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
		}{}),
		Endpoint: "health.snapshots",
		Params:   Params{"limit": In("Limit"), "start": StartIndex, "until": In("Until").Or(Today)},
		Verb:     "get health snapshots", Output: JSON{List: true},
	},
	Leaf{
		Name: "show", Help: "Show the summary of a Health Snapshot session.",
		Spec: argsOf(&struct {
			Date Date `arg:"" help:"Date of the session (YYYY-MM-DD, today, yesterday, 3d)."`
			UUID ID   `arg:"" help:"Session UUID (activityUuid.uuid in 'health snapshots list')."`
		}{}),
		Endpoint: "health.snapshot",
		Params:   Params{"date": In("Date"), "uuid": In("UUID")},
		Verb:     "get health snapshot", Output: JSON{},
	},
	Leaf{
		Name: "epochs", Help: "Show the measurements of a Health Snapshot session.",
		Spec: argsOf(&struct {
			UUID ID `arg:"" help:"Session UUID (activityUuid.uuid in 'health snapshots list')."`
		}{}),
		Endpoint: "health.snapshot-epochs",
		Params:   Params{"uuid": In("UUID")},
		Verb:     "get health snapshot epochs", Output: JSON{},
	},
	Custom{Name: "report", Help: "Save the PDF report of a Health Snapshot session.", Cmd: &HealthSnapshotReportCmd{}},
	Custom{Name: "rename", Help: "Rename a Health Snapshot session.", Cmd: &HealthSnapshotRenameCmd{}},
	Custom{Name: "note", Help: "Set or remove the note of a Health Snapshot session.", Cmd: &HealthSnapshotNoteCmd{}},
	Leaf{
		Name: "delete", Help: "Delete a Health Snapshot session.",
		Spec: argsOf(&struct {
			Date Date `arg:"" help:"Date of the session (YYYY-MM-DD, today, yesterday, 3d)."`
			UUID ID   `arg:"" help:"Session UUID (activityUuid.uuid in 'health snapshots list')."`
		}{}),
		Endpoint: "health.delete-snapshot",
		Params:   Params{"date": In("Date"), "uuid": In("UUID")},
		Verb:     "delete health snapshot",
		Output:   Write{Done: "Deleted Health Snapshot {UUID}", Confirm: "Delete Health Snapshot {UUID} of {Date}?"},
	},
}}

// HealthSnapshotRenameCmd renames one Health Snapshot session.
type HealthSnapshotRenameCmd struct {
	Date Date     `arg:"" help:"Date of the session (YYYY-MM-DD, today, yesterday, 3d)."`
	UUID ID       `arg:"" help:"Session UUID (activityUuid.uuid in 'health snapshots list')."`
	Name FreeText `arg:"" help:"New session name."`
}

func (c *HealthSnapshotRenameCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]string{"name": string(c.Name)})
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "health.rename-snapshot", map[string]string{"date": string(c.Date), "uuid": string(c.UUID)}, body)
	if err != nil {
		return fmt.Errorf("rename health snapshot: %w", err)
	}
	return reportWrite(g, data, "Renamed Health Snapshot %s to %q", c.UUID, c.Name)
}

// HealthSnapshotNoteCmd sets or removes the note of one Health Snapshot
// session.
type HealthSnapshotNoteCmd struct {
	Date  Date     `arg:"" help:"Date of the session (YYYY-MM-DD, today, yesterday, 3d)."`
	UUID  ID       `arg:"" help:"Session UUID (activityUuid.uuid in 'health snapshots list')."`
	Text  FreeText `arg:"" optional:"" help:"Note text. It replaces the note of the session."`
	Clear bool     `help:"Remove the note instead of setting a text."`
}

func (c *HealthSnapshotNoteCmd) Validate() error {
	switch {
	case c.Clear && c.Text != "":
		return errors.New("give a note text or --clear, not both")
	case !c.Clear && c.Text == "":
		return errors.New("give a note text, or --clear to remove the note")
	}
	return nil
}

func (c *HealthSnapshotNoteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]string{"notes": string(c.Text)})
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "health.snapshot-note", map[string]string{"date": string(c.Date), "uuid": string(c.UUID)}, body)
	if err != nil {
		return fmt.Errorf("set health snapshot note: %w", err)
	}
	if c.Clear {
		return reportWrite(g, data, "Removed the note of Health Snapshot %s", c.UUID)
	}
	return reportWrite(g, data, "Set the note of Health Snapshot %s", c.UUID)
}
