package cmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/muesli/termenv"
)

// colorHelpPrinter wraps Kong's default help printer with terminal colors.
// Section headers (e.g. "Flags:", "Commands:") are bold, and
// the "Run ... --help" hint is dimmed.
func colorHelpPrinter(options kong.HelpOptions, ctx *kong.Context) error {
	// Capture default help output into a buffer.
	var buf bytes.Buffer
	origStdout := ctx.Stdout
	ctx.Stdout = &buf

	if err := kong.DefaultHelpPrinter(options, ctx); err != nil {
		ctx.Stdout = origStdout
		return err
	}
	ctx.Stdout = origStdout

	// Apply terminal colors to the captured output.
	out := termenv.NewOutput(origStdout)
	raw := strings.TrimRight(buf.String(), "\n")
	if raw == "" {
		return nil
	}

	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		_, _ = fmt.Fprintln(origStdout, colorizeLine(out, line))
	}
	return nil
}

// colorizeLine applies terminal color to a single help output line.
func colorizeLine(out *termenv.Output, line string) string {
	trimmed := strings.TrimSpace(line)

	// "Usage: ..." line — bold.
	if strings.HasPrefix(trimmed, "Usage:") {
		return out.String(line).Bold().String()
	}

	// Section headers like "Flags:", "Commands:" — bold.
	// These are non-indented lines ending with a colon.
	if trimmed != "" && strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(line, " ") {
		return out.String(line).Bold().String()
	}

	// Hint lines like 'Run "gc <command> --help" ...' — dim.
	if strings.HasPrefix(trimmed, "Run \"") {
		return out.String(line).Faint().String()
	}

	return line
}
