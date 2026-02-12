package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bpauli/gccli/internal/outfmt"
)

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

	if outfmt.IsJSON(g.Context) {
		return outfmt.WriteJSON(os.Stdout, json.RawMessage(data))
	}

	g.UI.Successf("Uploaded %s", c.File)
	return nil
}
