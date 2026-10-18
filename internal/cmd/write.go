package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/bpauli/gccli/internal/outfmt"
)

var okJSON = json.RawMessage(`{"ok":true}`)

func reportWrite(g *Globals, body json.RawMessage, format string, args ...any) error {
	if outfmt.IsJSON(g.Context) {
		return writeResponseJSON(g.Stdout, body)
	}
	g.UI.Successf(format, args...)
	return nil
}

func reportWriteTable(g *Globals, body json.RawMessage, empty string, header []string, rows func() ([][]string, error)) error {
	if outfmt.IsJSON(g.Context) {
		return writeResponseJSON(g.Stdout, body)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		g.UI.Successf("%s", empty)
		return nil
	}
	r, err := rows()
	if err != nil {
		return err
	}
	return outfmt.WriteRows(g.Context, g.Stdout, header, r)
}

func writeResponseJSON(w io.Writer, body json.RawMessage) error {
	if len(bytes.TrimSpace(body)) == 0 {
		body = okJSON
	}
	if !json.Valid(body) {
		return fmt.Errorf("garmin saved the change, but its response is not JSON: %.200q", body)
	}
	return outfmt.WriteJSON(w, body)
}
