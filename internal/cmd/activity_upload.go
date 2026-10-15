package cmd

import "fmt"

// ActivityUploadCmd uploads an activity file.
type ActivityUploadCmd struct {
	File string `arg:"" help:"Path to activity file (FIT, GPX, or TCX)." type:"existingfile"`
}

func (c *ActivityUploadCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}

	data, err := client.UploadActivity(g.Context, c.File)
	if err != nil {
		return fmt.Errorf("upload activity: %w", err)
	}

	return reportWrite(g, data, "Uploaded %s", c.File)
}
