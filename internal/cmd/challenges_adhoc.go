package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

// challengeTypes maps the --type values of challenges adhoc create to the
// Garmin activity types.
var challengeTypes = map[string]garminapi.ChallengeType{
	"walking":          garminapi.ChallengeWalking,
	"running":          garminapi.ChallengeRunning,
	"cycling":          garminapi.ChallengeCycling,
	"swimming":         garminapi.ChallengeSwimming,
	"steps":            garminapi.ChallengeSteps,
	"pushes":           garminapi.ChallengePushes,
	"pushing-distance": garminapi.ChallengePushingDistance,
}

// challengeDuration sets the days of a challenge for a --duration value, as
// the web app does.
type challengeDuration struct {
	id garminapi.ChallengeDuration
	// start is the first day when --start is left out.
	start func(today time.Time) time.Time
	// end is the last day for the first day. It is nil when --end gives it.
	end func(start time.Time) (time.Time, error)
}

// maxCustomDays is the most days that the last day of a custom challenge
// can be after its first day.
const maxCustomDays = 60

var challengeDurations = map[string]challengeDuration{
	"day": {
		id:    garminapi.ChallengeDay,
		start: func(today time.Time) time.Time { return today },
		end:   func(start time.Time) (time.Time, error) { return start, nil },
	},
	"weekend": {
		id: garminapi.ChallengeWeekend,
		start: func(today time.Time) time.Time {
			mondayFirst := (int(today.Weekday()) + 6) % 7
			return addDays(today, 5-mondayFirst)
		},
		end: func(start time.Time) (time.Time, error) {
			if start.Weekday() != time.Saturday {
				return time.Time{}, fmt.Errorf("a weekend challenge starts on a Saturday, and %s is a %s", start.Format(time.DateOnly), start.Weekday())
			}
			return addDays(start, 1), nil
		},
	},
	"week": {
		id:    garminapi.ChallengeWeek,
		start: func(today time.Time) time.Time { return today },
		end:   func(start time.Time) (time.Time, error) { return addDays(start, 6), nil },
	},
	"month": {
		id:    garminapi.ChallengeMonth,
		start: func(today time.Time) time.Time { return addDays(today, 1-today.Day()) },
		end: func(start time.Time) (time.Time, error) {
			return time.Date(start.Year(), start.Month()+1, 0, 0, 0, 0, 0, start.Location()), nil
		},
	},
	"custom": {
		id:    garminapi.ChallengeCustom,
		start: func(today time.Time) time.Time { return today },
	},
}

func addDays(day time.Time, n int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day()+n, 0, 0, 0, 0, day.Location())
}

// challengeDays returns the first and the last day of a challenge with the
// duration, from --start and --end and the clock time now.
func challengeDays(duration string, start, end Date, now time.Time) (first, last time.Time, err error) {
	d := challengeDurations[duration]
	day := func(date Date) (time.Time, error) {
		return time.ParseInLocation(time.DateOnly, string(date), now.Location())
	}
	first = d.start(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()))
	if start != "" {
		if first, err = day(start); err != nil {
			return first, last, err
		}
	}
	if d.end != nil {
		if end != "" {
			return first, last, errors.New("--end works only with --duration custom")
		}
		last, err = d.end(first)
		return first, last, err
	}
	if end == "" {
		return first, last, errors.New("--duration custom needs --end")
	}
	if last, err = day(end); err != nil {
		return first, last, err
	}
	switch {
	case last.Before(first):
		return first, last, fmt.Errorf("--end %s is before the first day %s", end, first.Format(time.DateOnly))
	case last.After(addDays(first, maxCustomDays)):
		return first, last, fmt.Errorf("--end %s is more than %d days after the first day %s", end, maxCustomDays, first.Format(time.DateOnly))
	}
	return first, last, nil
}

// distinct returns the names in their first order, without repeats.
func distinct(names []ID) []string {
	seen := map[ID]bool{}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			out = append(out, string(n))
		}
	}
	return out
}

// ChallengesAdhocCreateCmd creates an ad-hoc challenge.
type ChallengesAdhocCreateCmd struct {
	Name        FreeText `arg:"" help:"Challenge name."`
	Type        string   `help:"Activity: walking, running, cycling, swimming, steps, pushes or pushing-distance." required:"" enum:"walking,running,cycling,swimming,steps,pushes,pushing-distance"`
	Duration    string   `help:"Length: day, weekend (Saturday and Sunday), week (7 days), month (to the end of the month) or custom (to --end)." default:"week" enum:"day,weekend,week,month,custom"`
	Start       Date     `help:"First day (YYYY-MM-DD, today, +3d). Default: today, the Saturday of this week for weekend, the first day of this month for month."`
	End         Date     `help:"Last day. Only with --duration custom, at most 60 days after the first day."`
	Description FreeText `help:"Description. Defaults to the name, because Garmin requires one."`
	Invite      []ID     `help:"Display name of a person who follows you and whom you follow. Repeat it to invite more people." placeholder:"DISPLAY-NAME" sep:"none"`

	first, last time.Time
}

func (c *ChallengesAdhocCreateCmd) Validate() error {
	var err error
	c.first, c.last, err = challengeDays(c.Duration, c.Start, c.End, nowFn())
	return err
}

func (c *ChallengesAdhocCreateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.CreateAdHocChallenge(g.Context, garminapi.AdHocChallenge{
		Name:        string(c.Name),
		Description: string(c.Description),
		Type:        challengeTypes[c.Type],
		Duration:    challengeDurations[c.Duration].id,
		Start:       c.first,
		End:         c.last,
		Invite:      distinct(c.Invite),
	})
	if err != nil {
		return fmt.Errorf("create ad-hoc challenge: %w", err)
	}
	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	if uuid := jsonString(resp, "uuid"); uuid != "" {
		return reportWrite(g, data, "Created ad-hoc challenge %q (UUID: %s)", c.Name, uuid)
	}
	return reportWrite(g, data, "Created ad-hoc challenge %q", c.Name)
}

// ChallengesAdhocInviteCmd invites people to an ad-hoc challenge.
type ChallengesAdhocInviteCmd struct {
	UUID  ID   `arg:"" help:"Challenge UUID, as the list commands print it."`
	Names []ID `arg:"" help:"Display name of a person who follows you and whom you follow." name:"display-name"`
}

func (c *ChallengesAdhocInviteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	names := distinct(c.Names)
	data, err := client.InviteToAdHocChallenge(g.Context, string(c.UUID), names)
	if err != nil {
		return fmt.Errorf("invite to ad-hoc challenge: %w", err)
	}
	people := "people"
	if len(names) == 1 {
		people = "person"
	}
	return reportWrite(g, data, "Invited %d %s to ad-hoc challenge %s", len(names), people, c.UUID)
}
