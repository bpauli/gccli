package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

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

// orToday returns the date, or the clock date when the flag was left out.
func (d Date) orToday() string {
	if d == "" {
		return nowFn().Format("2006-01-02")
	}
	return string(d)
}

// DateTime is a date and time in the local zone. Kong decodes it at parse
// time, so a bad value is a usage error before the login. The zero value
// means that the flag was left out.
type DateTime struct{ time.Time }

var dateTimeFormats = []string{
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

func (d *DateTime) Decode(ctx *kong.DecodeContext) error {
	tok, err := ctx.Scan.PopValue("date and time")
	if err != nil {
		return err
	}
	raw := fmt.Sprint(tok.Value)
	for _, f := range dateTimeFormats {
		if t, err := time.ParseInLocation(f, raw, nowFn().Location()); err == nil {
			d.Time = t
			return nil
		}
	}
	return fmt.Errorf("invalid date and time %q: expected YYYY-MM-DDTHH:MM:SS, YYYY-MM-DDTHH:MM or YYYY-MM-DD, with T or a space", raw)
}

// orNow returns the time, or the clock time when the flag was left out.
func (d DateTime) orNow() time.Time {
	if d.IsZero() {
		return nowFn()
	}
	return d.Time
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
