package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bpauli/gccli/internal/outfmt"
)

// ActivityTrimCmd trims the start or end of an activity. Nil means that
// the flag was left out.
type ActivityTrimCmd struct {
	ID   NumericID      `arg:"" help:"Activity ID."`
	From *time.Duration `help:"New start, as the time since the activity start (e.g. 2m, 1h5m). Default: the activity start."`
	To   *time.Duration `help:"New end, as the time since the activity start (e.g. 50m, 1h5m). Default: the activity end."`
}

func (c *ActivityTrimCmd) Help() string {
	return "For single-sport activities. Give --from, --to or both. Garmin applies the trim in a few seconds. To bring back the original, use activity restore."
}

func (c *ActivityTrimCmd) Validate() error {
	if c.From == nil && c.To == nil {
		return errors.New("nothing to trim: give --from, --to or both")
	}
	from := c.from()
	if from < 0 {
		return fmt.Errorf("--from must not be negative, got %s", from)
	}
	if c.To != nil && *c.To <= from {
		return fmt.Errorf("--to %s is not after --from %s", *c.To, from)
	}
	return nil
}

func (c *ActivityTrimCmd) from() time.Duration {
	if c.From == nil {
		return 0
	}
	return *c.From
}

func (c *ActivityTrimCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	args := map[string]string{"activityID": c.ID.String()}
	data, err := client.Call(g.Context, "activities.get", args, nil)
	if err != nil {
		return fmt.Errorf("get activity: %w", err)
	}
	start, elapsed, err := activityClock(data)
	if err != nil {
		return fmt.Errorf("activity %s: %w", c.ID, err)
	}

	from, to := c.from(), elapsed
	if c.To != nil {
		to = *c.To
	}
	switch {
	case to > elapsed:
		return fmt.Errorf("--to %s is after the end of activity %s at %s", to, c.ID, elapsed)
	case from >= to:
		return fmt.Errorf("--from %s is not before the end of activity %s at %s", from, c.ID, to)
	}

	body, err := json.Marshal(map[string]any{"timeRanges": []map[string]int64{{
		"startTime": start.Add(from).UnixMilli(),
		"endTime":   start.Add(to).UnixMilli(),
	}}})
	if err != nil {
		return err
	}
	data, err = client.Call(g.Context, "activities.reconstruct", args, body)
	if err != nil {
		return fmt.Errorf("trim activity: %w", err)
	}
	return reportWrite(g, data, "Trimmed activity %s to %s - %s", c.ID, from, to)
}

// activityClock reads the start time (UTC) and the elapsed duration from
// the answer of activities.get. Garmin writes the start without a zone,
// such as 2026-07-17T10:51:12.0, and the duration in seconds.
func activityClock(data json.RawMessage) (time.Time, time.Duration, error) {
	var a struct {
		Summary struct {
			StartTimeGMT    string   `json:"startTimeGMT"`
			ElapsedDuration *float64 `json:"elapsedDuration"`
		} `json:"summaryDTO"`
	}
	if err := unmarshalAnswer(data, &a); err != nil {
		return time.Time{}, 0, fmt.Errorf("parse activity: %w", err)
	}
	if a.Summary.StartTimeGMT == "" {
		return time.Time{}, 0, errors.New("the answer has no summaryDTO.startTimeGMT")
	}
	start, err := time.Parse("2006-01-02T15:04:05", a.Summary.StartTimeGMT)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("summaryDTO.startTimeGMT %q is not a time", a.Summary.StartTimeGMT)
	}
	if a.Summary.ElapsedDuration == nil || *a.Summary.ElapsedDuration <= 0 {
		return time.Time{}, 0, errors.New("the answer has no summaryDTO.elapsedDuration")
	}
	elapsed := time.Duration(*a.Summary.ElapsedDuration * float64(time.Second)).Round(time.Millisecond)
	return start, elapsed, nil
}

// ActivityRestoreCmd restores the original of a trimmed activity.
type ActivityRestoreCmd struct {
	ID NumericID `arg:"" help:"Activity ID."`
}

func (c *ActivityRestoreCmd) Help() string {
	return "For single-sport activities. An activity that is not trimmed does not change."
}

func (c *ActivityRestoreCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "activities.reconstruct", map[string]string{"activityID": c.ID.String()}, []byte("{}"))
	if err != nil {
		return fmt.Errorf("restore activity: %w", err)
	}
	return reportWrite(g, data, "Restored the original of activity %s", c.ID)
}

type ActivityLegsCmd struct {
	ID NumericID `arg:"" help:"Activity ID."`
}

func (c *ActivityLegsCmd) Help() string {
	return "START is in UTC. A single-sport activity has one leg."
}

func (c *ActivityLegsCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "activities.legs", map[string]string{"activityID": c.ID.String()}, nil)
	if err != nil {
		return fmt.Errorf("get activity legs: %w", err)
	}
	if outfmt.IsJSON(g.Context) {
		return writeObject(g, data)
	}
	var a struct {
		Reconstruct struct {
			Legs []struct {
				ActivityID int64 `json:"activityId"`
				Type       struct {
					Key string `json:"typeKey"`
				} `json:"activityTypeDTO"`
				StartTime int64 `json:"startTime"`
				EndTime   int64 `json:"endTime"`
			} `json:"reconstructRequest"`
		} `json:"activityReconstructDTO"`
	}
	if err := unmarshalAnswer(data, &a); err != nil {
		return fmt.Errorf("parse activity legs: %w", err)
	}
	rows := make([][]string, 0, len(a.Reconstruct.Legs))
	for _, leg := range a.Reconstruct.Legs {
		rows = append(rows, []string{
			strconv.FormatInt(leg.ActivityID, 10),
			leg.Type.Key,
			time.UnixMilli(leg.StartTime).UTC().Format(time.RFC3339),
			formatDuration(float64(leg.EndTime-leg.StartTime) / 1000),
		})
	}
	return outfmt.WriteRows(g.Context, g.Stdout, []string{"ID", "TYPE", "START", "DURATION"}, rows)
}
