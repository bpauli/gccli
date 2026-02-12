package cmd

import (
	"os"
	"text/tabwriter"

	"github.com/bpauli/gccli/internal/ui"
)

// TableWriter creates a tabwriter for aligned table output to stdout.
func TableWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

// PrintNextPageHint prints a hint about fetching more results when
// there are additional pages of data available.
func PrintNextPageHint(u *ui.UI, start, limit, total int) {
	next := start + limit
	if next < total {
		u.Infof("Showing %d-%d of %d. Use --start %d for more.", start+1, next, total, next)
	}
}
