package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
)

// popString takes the next token of a flag or positional as its raw string.
// what names the value in the error for a missing token.
func popString(ctx *kong.DecodeContext, what string) (string, error) {
	tok, err := ctx.Scan.PopValue(what)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(tok.Value), nil
}

type Date string

func (d *Date) Decode(ctx *kong.DecodeContext) error {
	raw, err := popString(ctx, "date")
	if err != nil {
		return err
	}
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
	raw, err := popString(ctx, "date and time")
	if err != nil {
		return err
	}
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

// NumericID is an ID that a request body carries as a number, such as an
// activity ID. Kong checks it at parse time, so a non-numeric ID is a usage
// error before the login. String gives the decimal form for a request path.
type NumericID int64

func (id *NumericID) Decode(ctx *kong.DecodeContext) error {
	raw, err := popString(ctx, "id")
	if err != nil {
		return err
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID %q: expected a number", raw)
	}
	*id = NumericID(n)
	return nil
}

func (id NumericID) String() string { return strconv.FormatInt(int64(id), 10) }

// popNonEmpty takes the next token of a flag or positional and rejects an
// empty or blank one. Any other value is kept as given, spaces included.
func popNonEmpty(ctx *kong.DecodeContext, what string) (string, error) {
	raw, err := popString(ctx, what)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("must not be empty")
	}
	return raw, nil
}

// ID is a user-given value that names a Garmin object in a request path,
// such as an activity ID or a gear UUID. Kong rejects an empty or blank
// value at parse time, so it is a usage error before the login. The value
// is sent as given.
type ID string

func (id *ID) Decode(ctx *kong.DecodeContext) error {
	raw, err := popNonEmpty(ctx, "id")
	if err != nil {
		return err
	}
	*id = ID(raw)
	return nil
}

// FreeText is a required free-text value that a request body carries, such as
// a name or a type key. Kong rejects an empty or blank value at parse time,
// so it is a usage error before the login. The value is sent as given.
type FreeText string

func (t *FreeText) Decode(ctx *kong.DecodeContext) error {
	raw, err := popNonEmpty(ctx, "free text")
	if err != nil {
		return err
	}
	*t = FreeText(raw)
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
