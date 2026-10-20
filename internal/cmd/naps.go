package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/alecthomas/kong"
)

var healthSleepNapsGroup = Group{Name: "naps", Help: "Show, add and delete naps.", Children: []Node{
	Leaf{
		Name: "view", Help: "Show naps for a date.", Default: true,
		Spec: argsOf(&struct {
			Date Date `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today."`
		}{}),
		Endpoint: "health.sleep-naps",
		Params:   Params{"date": In("Date")},
		Verb:     "get naps", Output: JSON{List: true},
	},
	Custom{Name: "add", Help: "Add a nap for today. Garmin takes naps only for the current day.", Cmd: &HealthSleepNapsAddCmd{}},
	Custom{Name: "delete", Help: "Delete a nap.", Cmd: &HealthSleepNapsDeleteCmd{}},
}}

// Clock is a wall-clock time HH:MM in the user's Garmin time zone.
type Clock string

func (c *Clock) Decode(ctx *kong.DecodeContext) error {
	tok, err := ctx.Scan.PopValue("time")
	if err != nil {
		return err
	}
	raw := fmt.Sprint(tok.Value)
	t, err := time.Parse("15:04", raw)
	if err != nil {
		return fmt.Errorf("invalid time %q: expected HH:MM (00:00 to 23:59)", raw)
	}
	*c = Clock(t.Format("15:04"))
	return nil
}

// on returns c on the date of day, in the location of day.
func (c Clock) on(day time.Time) time.Time {
	t, _ := time.Parse("15:04", string(c))
	return time.Date(day.Year(), day.Month(), day.Day(), t.Hour(), t.Minute(), 0, 0, day.Location())
}

// napTimestamp is the GMT time format of the nap endpoints.
const napTimestamp = "2006-01-02T15:04:05.000"

// HealthSleepNapsAddCmd adds a nap for today.
type HealthSleepNapsAddCmd struct {
	Start Clock `arg:"" help:"Start time (HH:MM) in your Garmin time zone."`
	End   Clock `arg:"" help:"End time (HH:MM) in your Garmin time zone. Must not be later than now."`
}

func (c *HealthSleepNapsAddCmd) Validate() error {
	if c.End <= c.Start {
		return fmt.Errorf("the end time %s must be after the start time %s", c.End, c.Start)
	}
	return nil
}

func (c *HealthSleepNapsAddCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	loc, err := client.GetTimeZone(g.Context)
	if err != nil {
		return fmt.Errorf("get time zone: %w", err)
	}
	now := nowFn().In(loc)
	start, end := c.Start.on(now), c.End.on(now)
	if end.After(now) {
		return fmt.Errorf("the end time %s is later than now (%s in your Garmin time zone)", c.End, now.Format("15:04"))
	}
	pk, err := client.GetProfilePK(g.Context)
	if err != nil {
		return err
	}
	date := now.Format("2006-01-02")
	body, err := json.Marshal(map[string]any{
		"calendarDate":         date,
		"napStartTimestampGMT": start.UTC().Format(napTimestamp),
		"napEndTimestampGMT":   end.UTC().Format(napTimestamp),
		"userProfilePK":        json.Number(pk),
	})
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "health.add-nap", nil, body)
	if err != nil {
		return fmt.Errorf("add nap: %w", err)
	}
	return reportWrite(g, data, "Added nap on %s from %s to %s", date, c.Start, c.End)
}

// HealthSleepNapsDeleteCmd deletes the nap that starts at a time of a day.
type HealthSleepNapsDeleteCmd struct {
	Date  Date  `arg:"" help:"Date of the nap (YYYY-MM-DD, today, yesterday, 3d, +30d)."`
	Start Clock `arg:"" help:"Start time of the nap (HH:MM) in your Garmin time zone."`
	Force bool  `help:"Skip confirmation prompt." short:"f"`
}

func (c *HealthSleepNapsDeleteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	loc, err := client.GetTimeZone(g.Context)
	if err != nil {
		return fmt.Errorf("get time zone: %w", err)
	}
	data, err := client.Call(g.Context, "health.sleep-naps", map[string]string{"date": string(c.Date)}, nil)
	if err != nil {
		return fmt.Errorf("get naps: %w", err)
	}
	start, err := napStart(data, loc, c.Start)
	if err != nil {
		return err
	}
	if start.IsZero() {
		return fmt.Errorf("no nap on %s starts at %s", c.Date, c.Start)
	}
	ok, err := confirm(g.Stderr, fmt.Sprintf("Delete nap on %s at %s?", c.Date, c.Start), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}
	params := map[string]string{"date": string(c.Date), "startMillis": strconv.FormatInt(start.UnixMilli(), 10)}
	data, err = client.Call(g.Context, "health.delete-nap", params, nil)
	if err != nil {
		return fmt.Errorf("delete nap: %w", err)
	}
	return reportWrite(g, data, "Deleted nap on %s at %s", c.Date, c.Start)
}

// napStart returns the GMT start of the first nap in naps that starts at
// the minute at in loc, or the zero time. A DELETE of a nap that does not
// exist also answers 204, so the start comes from the nap list: a missing
// nap is an error, and the path gets the exact start.
func napStart(naps json.RawMessage, loc *time.Location, at Clock) (time.Time, error) {
	var list []struct {
		Start string `json:"napStartTimestampGMT"`
	}
	if err := json.Unmarshal(naps, &list); err != nil {
		return time.Time{}, fmt.Errorf("parse naps: %w", err)
	}
	for _, n := range list {
		t, err := time.ParseInLocation("2006-01-02T15:04:05", n.Start, time.UTC)
		if err != nil {
			return time.Time{}, errors.New("parse naps: napStartTimestampGMT is not a time")
		}
		if Clock(t.In(loc).Format("15:04")) == at {
			return t, nil
		}
	}
	return time.Time{}, nil
}
