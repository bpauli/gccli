package ui

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/muesli/termenv"
)

type contextKey struct{}

// UI provides colored terminal output to stderr.
type UI struct {
	out    *termenv.Output
	writer io.Writer
}

// New creates a UI that writes to stderr with the given color mode.
// Valid modes: "auto" (detect terminal), "always" (force colors), "never" (no colors).
func New(colorMode string) *UI {
	return NewWithWriter(os.Stderr, colorMode)
}

// NewWithWriter creates a UI that writes to w with the given color mode.
func NewWithWriter(w io.Writer, colorMode string) *UI {
	var out *termenv.Output
	switch colorMode {
	case "never":
		out = termenv.NewOutput(w, termenv.WithProfile(termenv.Ascii))
	case "always":
		out = termenv.NewOutput(w, termenv.WithProfile(termenv.TrueColor))
	default: // "auto"
		out = termenv.NewOutput(w)
	}
	return &UI{out: out, writer: w}
}

// NewContext stores the UI in the context.
func NewContext(ctx context.Context, u *UI) context.Context {
	return context.WithValue(ctx, contextKey{}, u)
}

// FromContext retrieves the UI from the context.
// Returns a default auto-mode UI if none is stored.
func FromContext(ctx context.Context) *UI {
	if u, ok := ctx.Value(contextKey{}).(*UI); ok {
		return u
	}
	return New("auto")
}

// Successf prints a green message.
func (u *UI) Successf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	styled := u.out.String(msg).Foreground(u.out.Color("2"))
	_, _ = fmt.Fprintln(u.writer, styled)
}

// Error prints a bold red error message.
func (u *UI) Error(err error) {
	styled := u.out.String(err.Error()).Foreground(u.out.Color("1")).Bold()
	_, _ = fmt.Fprintln(u.writer, styled)
}

// Warnf prints a yellow message.
func (u *UI) Warnf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	styled := u.out.String(msg).Foreground(u.out.Color("3"))
	_, _ = fmt.Fprintln(u.writer, styled)
}

// Infof prints a blue message.
func (u *UI) Infof(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	styled := u.out.String(msg).Foreground(u.out.Color("4"))
	_, _ = fmt.Fprintln(u.writer, styled)
}
