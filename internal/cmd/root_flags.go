package cmd

// RootFlags contains global CLI flags available to all commands.
type RootFlags struct {
	JSON    bool   `help:"Output as JSON." short:"j" env:"GC_JSON"`
	Plain   bool   `help:"Output as plain text (TSV)." env:"GC_PLAIN"`
	Color   string `help:"Color mode: auto, always, never." default:"auto" enum:"auto,always,never" env:"GC_COLOR"`
	Account string `help:"Garmin account email." env:"GC_ACCOUNT"`
}
