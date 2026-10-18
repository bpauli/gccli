package cmd

import (
	"errors"

	"github.com/bpauli/gccli/internal/config"
	"github.com/bpauli/gccli/internal/outfmt"
)

// RootFlags contains global CLI flags available to all commands.
type RootFlags struct {
	JSON    bool   `help:"Output as JSON." short:"j" env:"GCCLI_JSON"`
	Plain   bool   `help:"Output as plain text (TSV)." env:"GCCLI_PLAIN"`
	Header  bool   `help:"Print the column names as the first line of --plain output." env:"GCCLI_HEADER"`
	Color   string `help:"Color mode: auto, always, never." default:"auto" enum:"auto,always,never" env:"GCCLI_COLOR"`
	Account string `help:"Garmin account email." env:"GCCLI_ACCOUNT"`
	Units   string `help:"Unit system for distances, speeds, paces and elevations in table and plain output: auto (Garmin profile), metric, statute_us, statute_uk." default:"auto" enum:"${units}" env:"GCCLI_UNITS"`
}

// Kong calls Validate after parsing.
func (f *RootFlags) Validate() error {
	if f.Header && f.outputMode() != outfmt.PlainHeader {
		return errors.New("--header works only with --plain")
	}
	return nil
}

func (f *RootFlags) outputMode() outfmt.Mode {
	switch {
	case f.JSON || config.IsJSON():
		return outfmt.JSON
	case !f.Plain && !config.IsPlain():
		return outfmt.Table
	case f.Header:
		return outfmt.PlainHeader
	}
	return outfmt.Plain
}
