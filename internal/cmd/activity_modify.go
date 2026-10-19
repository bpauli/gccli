package cmd

import "fmt"

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

	return reportWrite(g, data, "Renamed activity %s to %q", c.ID, c.Name)
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

	return reportWrite(g, data, "Changed activity %s type to %s", c.ID, c.TypeKey)
}
