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

func reportWriteTable(g *Globals, body json.RawMessage, header []string, rows func() ([][]string, error)) error {
	if outfmt.IsJSON(g.Context) {
		return writeResponseJSON(g.Stdout, body)
	}
	r, err := rows()
	if err != nil {
		return err
	}
	if outfmt.IsPlain(g.Context) {
		return outfmt.WritePlain(g.Stdout, r)
	}
	return outfmt.WriteTable(g.Stdout, header, r)
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
