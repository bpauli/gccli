package outfmt

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"text/tabwriter"
)

type contextKey struct{}

// Mode represents the output format.
type Mode int

const (
	// Table is the default mode using tabwriter-aligned columns.
	Table Mode = iota
	// JSON outputs pretty-printed JSON.
	JSON
	// Plain outputs TSV/plain text with no alignment.
	Plain
)

// NewContext stores the output mode in the context.
func NewContext(ctx context.Context, m Mode) context.Context {
	return context.WithValue(ctx, contextKey{}, m)
}

// ModeFromContext retrieves the output mode from the context.
// Returns Table if none is stored.
func ModeFromContext(ctx context.Context) Mode {
	if m, ok := ctx.Value(contextKey{}).(Mode); ok {
		return m
	}
	return Table
}

// IsJSON returns true if the context output mode is JSON.
func IsJSON(ctx context.Context) bool {
	return ModeFromContext(ctx) == JSON
}

// WriteJSON writes v as pretty-printed JSON to stdout.
// HTML characters are not escaped.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// WriteRows writes rows in the table or plain format of the mode in ctx.
// Table mode aligns the columns under header, and Plain mode does not write
// it. Each cell is escaped,
// so a value never adds a column or a line.
func WriteRows(ctx context.Context, w io.Writer, header []string, rows [][]string) error {
	mode := ModeFromContext(ctx)
	if mode == Plain {
		return writeLines(w, rows)
	}
	if len(header) > 0 {
		rows = append([][]string{header}, rows...)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if err := writeLines(tw, rows); err != nil {
		return err
	}
	return tw.Flush()
}

var cellEscaper = strings.NewReplacer(`\`, `\\`, "\t", `\t`, "\n", `\n`, "\r", `\r`, "\v", `\v`, "\f", `\f`)

func writeLines(w io.Writer, rows [][]string) error {
	var line []byte
	for _, row := range rows {
		line = line[:0]
		for i, cell := range row {
			if i > 0 {
				line = append(line, '\t')
			}
			line = append(line, cellEscaper.Replace(cell)...)
		}
		line = append(line, '\n')
		if _, err := w.Write(line); err != nil {
			return err
		}
	}
	return nil
}
