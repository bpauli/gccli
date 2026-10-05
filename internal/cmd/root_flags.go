package cmd

// RootFlags contains global CLI flags available to all commands.
type RootFlags struct {
	JSON    bool   `help:"Output as JSON." short:"j" env:"GCCLI_JSON"`
	Plain   bool   `help:"Output as plain text (TSV)." env:"GCCLI_PLAIN"`
	Color   string `help:"Color mode: auto, always, never." default:"auto" enum:"auto,always,never" env:"GCCLI_COLOR"`
	Account string `help:"Garmin account email." env:"GCCLI_ACCOUNT"`
	Units   string `help:"Unit system for distances, speeds, paces and elevations in table and plain output: auto (Garmin profile), metric, statute_us, statute_uk." default:"auto" enum:"${units}" env:"GCCLI_UNITS"`
}
