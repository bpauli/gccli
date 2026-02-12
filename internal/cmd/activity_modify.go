package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bpauli/gccli/internal/outfmt"
)

// ActivityRenameCmd renames an activity.
type ActivityRenameCmd struct {
	ID   string `arg:"" help:"Activity ID."`
	Name string `arg:"" help:"New activity name."`
}

func (c *ActivityRenameCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.RenameActivity(g.Context, c.ID, c.Name)
	if err != nil {
		return fmt.Errorf("rename activity: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	g.UI.Successf("Renamed activity %s to %q", c.ID, c.Name)
	return nil
}

// ActivityRetypeCmd changes the type of an activity.
type ActivityRetypeCmd struct {
	ID           string `arg:"" help:"Activity ID."`
	TypeID       int    `help:"Activity type ID." required:"" name:"type-id"`
	TypeKey      string `help:"Activity type key (e.g. running, cycling)." required:"" name:"type-key"`
	ParentTypeID int    `help:"Parent activity type ID." default:"0" name:"parent-type-id"`
}

func (c *ActivityRetypeCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.RetypeActivity(g.Context, c.ID, c.TypeID, c.TypeKey, c.ParentTypeID)
	if err != nil {
		return fmt.Errorf("retype activity: %w", err)
	}

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	g.UI.Successf("Changed activity %s type to %s", c.ID, c.TypeKey)
	return nil
}

// ActivityDeleteCmd deletes an activity.
type ActivityDeleteCmd struct {
	ID    string `arg:"" help:"Activity ID."`
	Force bool   `help:"Skip confirmation prompt." short:"f"`
}

func (c *ActivityDeleteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	ok, err := confirm(os.Stderr, fmt.Sprintf("Delete activity %s?", c.ID), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}

	if err := client.DeleteActivity(g.Context, c.ID); err != nil {
		return fmt.Errorf("delete activity: %w", err)
	}

	g.UI.Successf("Deleted activity %s", c.ID)
	return nil
}
