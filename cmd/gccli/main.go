package main

import (
	"os"

	"github.com/bpauli/gccli/internal/cmd"
)

// version info set via ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(cmd.Execute(os.Args[1:], version, commit, date))
}
