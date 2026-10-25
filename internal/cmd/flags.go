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

// FlightTime is a local date and time with the UTC offset of that place,
// such as 2026-11-02T10:00+01:00. Kong checks it at parse time, so a time
// without an offset is a usage error before the login. The zero value means
// that the flag was left out.
type FlightTime struct{ time.Time }

func (f *FlightTime) Decode(ctx *kong.DecodeContext) error {
	raw, err := popString(ctx, "date and time")
	if err != nil {
		return err
	}
	for _, layout := range []string{"2006-01-02T15:04Z07:00", time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			f.Time = t
			return nil
		}
	}
	return fmt.Errorf("invalid date and time %q: expected YYYY-MM-DDTHH:MM with the UTC offset, such as 2026-11-02T10:00+01:00", raw)
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

// Month is a month of the year, 1 to 12, in a request query. Kong checks it
// at parse time, so a bad month is a usage error before the login. The zero
// value is no month.
type Month string

func (m *Month) Decode(ctx *kong.DecodeContext) error {
	raw, err := popString(ctx, "month")
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 12 {
		return fmt.Errorf("invalid month %q: expected 1 to 12", raw)
	}
	*m = Month(strconv.Itoa(n))
	return nil
}

// BoundingBox is an area of the map, given as MINLAT,MINLON,MAXLAT,MAXLON in
// degrees. Kong checks it at parse time, so a bad box is a usage error
// before the login. The zero value is no box, because a valid box has each
// minimum less than its maximum.
type BoundingBox struct {
	MinLat, MinLon, MaxLat, MaxLon float64
}

func (b *BoundingBox) Decode(ctx *kong.DecodeContext) error {
	raw, err := popString(ctx, "bounding box")
	if err != nil {
		return err
	}
	box, err := parseBoundingBox(raw)
	if err != nil {
		return fmt.Errorf("invalid bounding box %q: %w", raw, err)
	}
	*b = box
	return nil
}

func parseBoundingBox(raw string) (BoundingBox, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return BoundingBox{}, errors.New("expected MINLAT,MINLON,MAXLAT,MAXLON")
	}
	var v [4]float64
	for i, p := range parts {
		n, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return BoundingBox{}, fmt.Errorf("%q is not a number: expected MINLAT,MINLON,MAXLAT,MAXLON", p)
		}
		v[i] = n
	}
	b := BoundingBox{MinLat: v[0], MinLon: v[1], MaxLat: v[2], MaxLon: v[3]}
	// The checks are written so that NaN fails them.
	switch {
	case !(b.MinLat >= -90 && b.MaxLat <= 90):
		return BoundingBox{}, errors.New("a latitude must be in -90 to 90")
	case !(b.MinLon >= -180 && b.MaxLon <= 180):
		return BoundingBox{}, errors.New("a longitude must be in -180 to 180")
	case !(b.MinLat < b.MaxLat):
		return BoundingBox{}, errors.New("MINLAT must be less than MAXLAT")
	case !(b.MinLon < b.MaxLon):
		return BoundingBox{}, errors.New("MINLON must be less than MAXLON")
	}
	return b, nil
}

// Upper is a lowercase kebab-case flag value that Garmin takes in upper
// snake case: the runner sends cross-country-skiing as CROSS_COUNTRY_SKIING.
// Its Kong enum tag lists the lowercase values.
type Upper string

func (u Upper) wire() string { return strings.ToUpper(strings.ReplaceAll(string(u), "-", "_")) }

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

// FreeText is a required free-text value that a request carries, such as a
// name or a type key in a body, or a search text in a query. Kong rejects an
// empty or blank value at parse time, so it is a usage error before the
// login. The value is sent as given.
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

// MaxItems is a --limit without an --offset, for a Garmin list that the
// command reads only from its first item.
type MaxItems struct {
	Limit int `help:"Maximum number of ${items} to return." default:"20" short:"l"`
}

func (m *MaxItems) Validate() error {
	if m.Limit < 1 {
		return fmt.Errorf("--limit must be at least 1, got %d", m.Limit)
	}
	return nil
}

// AlignedPagination is Pagination for a Garmin list that starts a page only
// at a multiple of the page size. Garmin answers any other offset with 400.
type AlignedPagination struct {
	Pagination `embed:""`
}

func (p *AlignedPagination) Validate() error {
	if err := p.Pagination.Validate(); err != nil {
		return err
	}
	if p.Offset%p.Limit != 0 {
		return fmt.Errorf("--offset %d is not a multiple of --limit %d: Garmin pages this list by --limit", p.Offset, p.Limit)
	}
	return nil
}
