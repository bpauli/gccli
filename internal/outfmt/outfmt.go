package outfmt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
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

// IsPlain returns true if the context output mode is Plain.
func IsPlain(ctx context.Context) bool {
	return ModeFromContext(ctx) == Plain
}

// WriteJSON writes v as pretty-printed JSON to stdout.
// HTML characters are not escaped.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// NewTabWriter creates a tabwriter suitable for aligned table output.
// Caller must call Flush() when done writing.
func NewTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

// WriteTable writes header and rows in aligned table format to stdout.
func WriteTable(w io.Writer, header []string, rows [][]string) error {
	tw := NewTabWriter(w)
	if len(header) > 0 {
		for i, h := range header {
			if i > 0 {
				if _, err := fmt.Fprint(tw, "\t"); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprint(tw, h); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(tw); err != nil {
			return err
		}
	}
	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				if _, err := fmt.Fprint(tw, "\t"); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprint(tw, col); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(tw); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// WritePlain writes rows as tab-separated values with no alignment.
func WritePlain(w io.Writer, rows [][]string) error {
	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				if _, err := fmt.Fprint(w, "\t"); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprint(w, col); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

// Write writes data in the format specified by the context mode.
// For JSON mode, v is written as pretty-printed JSON.
// For Plain mode, rows are written as TSV.
// For Table mode, header and rows are written as aligned columns.
func Write(ctx context.Context, v any, header []string, rows [][]string) error {
	w := os.Stdout
	switch ModeFromContext(ctx) {
	case JSON:
		return WriteJSON(w, v)
	case Plain:
		return WritePlain(w, rows)
	default:
		return WriteTable(w, header, rows)
	}
}
