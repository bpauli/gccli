package cmd

import (
	"fmt"

	"github.com/alecthomas/kong"
)

type Date string

func (d *Date) Decode(ctx *kong.DecodeContext) error {
	tok, err := ctx.Scan.PopValue("date")
	if err != nil {
		return err
	}
	resolved, err := resolveDate(fmt.Sprint(tok.Value))
	if err != nil {
		return err
	}
	*d = Date(resolved)
	return nil
}

type Pagination struct {
	Limit  int `help:"Maximum number of ${items} to return." default:"20" short:"l"`
	Offset int `help:"Number of ${items} to skip." default:"0"`
}
