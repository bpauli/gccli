package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/alecthomas/kong"
)

type Date string

func (d *Date) Decode(ctx *kong.DecodeContext) error {
	tok, err := ctx.Scan.PopValue("date")
	if err != nil {
		return err
	}
	raw := fmt.Sprint(tok.Value)
	if strings.TrimSpace(raw) == "" {
		return errors.New("empty date: expected YYYY-MM-DD, today, yesterday, Nd (e.g. 3d), or +Nd (e.g. +30d)")
	}
	resolved, err := resolveDate(raw)
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

func (p *Pagination) Validate() error {
	if p.Limit < 1 {
		return fmt.Errorf("--limit must be at least 1, got %d", p.Limit)
	}
	if p.Offset < 0 {
		return fmt.Errorf("--offset must not be negative, got %d", p.Offset)
	}
	return nil
}
