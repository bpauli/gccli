package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
	"github.com/bpauli/gccli/internal/outfmt"
)

var groupsGroup = Group{Name: "groups", Help: "List, search, create, edit, delete, join and leave groups.", Children: []Node{
	Leaf{
		Name: "list", Help: "List your groups.", Default: true,
		Endpoint: "groups.list",
		Params:   Params{"displayName": DisplayName},
		Verb:     "get groups", Output: JSON{List: true},
	},
	Custom{Name: "search", Help: "Search groups by keyword.", Cmd: &GroupsSearchCmd{}},
	Leaf{
		Name: "show", Help: "Show a group.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Group ID."`
		}{}),
		Endpoint: "groups.get",
		Params:   Params{"groupID": In("ID")},
		Verb:     "get group", Output: JSON{},
	},
	Leaf{
		Name: "members", Help: "List the members of a group.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Group ID."`
		}{}),
		Endpoint: "groups.members",
		Params:   Params{"groupID": In("ID")},
		Verb:     "get group members",
		Output: Rows{Noun: "group members", Columns: []Column{
			{Label: "ID", Path: "userProfileId"},
			{Label: "NAME", Path: "fullName"},
			{Label: "DISPLAY NAME", Path: "displayName"},
			{Label: "ROLE", Path: "groupRole"},
		}},
	},
	Leaf{
		Name: "feed", Help: "Show the latest posts and activities of the group feed.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Group ID."`
		}{}),
		Endpoint: "groups.feed",
		Params:   Params{"groupID": In("ID")},
		Verb:     "get group feed", Output: JSON{},
	},
	Leaf{
		Name: "activities", Help: "List the latest activities of the group members.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Group ID."`
		}{}),
		Endpoint: "groups.activities",
		Params:   Params{"groupID": In("ID")},
		Verb:     "get group activities",
		Output:   Rows{Noun: "group activities", Items: "activityList", Columns: activityColumns},
	},
	Leaf{
		Name: "challenges", Help: "List the open challenges of a group. Garmin answers 403 to a non-member.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Group ID."`
		}{}),
		Endpoint: "groups.challenges",
		Params:   Params{"groupID": In("ID")},
		Verb:     "get group challenges", Output: JSON{List: true},
	},
	Custom{Name: "leaderboard", Help: "Rank the members of a group by steps, calories, pushes or distance.", Cmd: &GroupsLeaderboardCmd{}},
	Leaf{
		Name: "requests", Help: "List your pending requests to join private groups.",
		Endpoint: "groups.requests",
		Params:   Params{"displayName": DisplayName},
		Verb:     "get group join requests", Output: JSON{List: true},
	},
	Custom{Name: "join", Help: "Join a public group. gccli does not send the request that a private group needs.", Cmd: &GroupsJoinCmd{}},
	Leaf{
		Name: "leave", Help: "Leave a group.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Group ID."`
		}{}),
		Endpoint: "groups.leave",
		Params:   Params{"groupID": In("ID"), "profileID": ProfilePK},
		Verb:     "leave group",
		Output:   Write{Done: "Left group {ID}", Confirm: "Leave group {ID}?"},
	},
	Custom{Name: "create", Help: "Create a group.", Cmd: &GroupsCreateCmd{}},
	Custom{Name: "edit", Help: "Edit a group you own.", Cmd: &GroupsEditCmd{}},
	Leaf{
		Name: "delete", Help: "Delete a group you own.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Group ID."`
		}{}),
		Endpoint: "groups.delete",
		Params:   Params{"groupID": In("ID")},
		Verb:     "delete group",
		Output:   Write{Done: "Deleted group {ID}", Confirm: "Delete group {ID}? This removes it for all members."},
	},
}}

var groupsSearchRows = Rows{Noun: "groups", Items: "groupDTOs", Columns: []Column{
	{Label: "ID", Path: "id"},
	{Label: "NAME", Path: "groupName"},
	{Label: "PRIVACY", Path: "groupPrivacy"},
	{Label: "LOCATION", Path: "location"},
}}

// GroupsSearchCmd searches all groups by keyword.
type GroupsSearchCmd struct {
	Keyword    FreeText `arg:"" help:"Text in the group name or description."`
	Pagination `embed:"" set:"items=groups"`
}

func (c *GroupsSearchCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	// Garmin counts start from 1, so the first group is start=1.
	form := url.Values{
		"keyword": {string(c.Keyword)},
		"start":   {strconv.Itoa(c.Offset + 1)},
		"limit":   {strconv.Itoa(c.Limit)},
	}
	data, err := client.Call(g.Context, "groups.search", nil, []byte(form.Encode()))
	if err != nil {
		return fmt.Errorf("search groups: %w", err)
	}
	return groupsSearchRows.print(g, client, data)
}

// GroupsJoinCmd joins a public group. The body carries both IDs as JSON
// numbers, as the web app sends them.
type GroupsJoinCmd struct {
	ID NumericID `arg:"" help:"Group ID."`
}

func (c *GroupsJoinCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	pk, err := client.GetProfilePK(g.Context)
	if err != nil {
		return err
	}
	id := c.ID.String()
	body, err := json.Marshal(map[string]json.Number{"groupId": json.Number(id), "userProfileId": json.Number(pk)})
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "groups.join", map[string]string{"groupID": id, "profileID": pk}, body)
	if err != nil {
		return fmt.Errorf("join group: %w", err)
	}
	return reportWrite(g, data, "Joined group %s", id)
}

// leaderboardMetric is one metric of the group leaderboard: the user type
// that Garmin puts in the path, the metric ID, the key of the entry list in
// the answer and how a value prints.
type leaderboardMetric struct {
	userType, id, key string
	format            Format
}

var leaderboardMetrics = map[string]leaderboardMetric{
	"steps":    {"wellness", "29", "WELLNESS_TOTAL_STEPS", Text},
	"calories": {"wellness", "22", "WELLNESS_ACTIVE_CALORIES", Text},
	"pushes":   {"wellness", "87", "WELLNESS_TOTAL_PUSHES", Text},
	"distance": {"activity", "17", "ACTIVITY_TOTAL_DISTANCE", Distance},
}

// leaderboardActivities maps each --activity to the activity type ID in
// Garmin's group activity types.
var leaderboardActivities = map[string]string{"walking": "9", "running": "1", "cycling": "2", "swimming": "26", "pushing": "199"}

// GroupsLeaderboardCmd ranks the members of a group by one metric over a
// range of days. Garmin sends the entries in rank order.
type GroupsLeaderboardCmd struct {
	ID         ID      `arg:"" help:"Group ID."`
	Metric     string  `help:"What to rank: steps, calories, pushes or distance. The first three show nothing for a group you have not joined." default:"steps" enum:"steps,calories,pushes,distance"`
	Activity   *string `help:"Count only the distance of one activity: walking, running, cycling, swimming or pushing. Only with --metric distance." enum:"walking,running,cycling,swimming,pushing" placeholder:"ACTIVITY"`
	Start      Date    `help:"First day (YYYY-MM-DD, today, yesterday, 3d, +30d). Default: Monday of this week."`
	End        Date    `help:"Last day. Default: 6 days after --start."`
	Pagination `embed:"" set:"items=members"`

	first, last string
}

func (c *GroupsLeaderboardCmd) Validate() error {
	if c.Activity != nil && c.Metric != "distance" {
		return fmt.Errorf("--activity %s goes only with --metric distance, not --metric %s", *c.Activity, c.Metric)
	}
	now := nowFn()
	day := func(date Date) (time.Time, error) {
		return time.ParseInLocation(time.DateOnly, string(date), now.Location())
	}
	first := addDays(now, -int(now.Weekday()+6)%7)
	if c.Start != "" {
		var err error
		if first, err = day(c.Start); err != nil {
			return err
		}
	}
	last := addDays(first, 6)
	if c.End != "" {
		var err error
		if last, err = day(c.End); err != nil {
			return err
		}
	}
	if last.Before(first) {
		return fmt.Errorf("--end %s is before --start %s", last.Format("2006-01-02"), first.Format("2006-01-02"))
	}
	c.first, c.last = first.Format("2006-01-02"), last.Format("2006-01-02")
	return nil
}

func (c *GroupsLeaderboardCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	m := leaderboardMetrics[c.Metric]
	var actType string
	if c.Activity != nil {
		actType = leaderboardActivities[*c.Activity]
	}
	data, err := client.Call(g.Context, "groups.leaderboard", map[string]string{
		"userType":  m.userType,
		"groupID":   string(c.ID),
		"start":     strconv.Itoa(c.Offset + 1),
		"limit":     strconv.Itoa(c.Limit),
		"todayDate": nowFn().Format("2006-01-02"),
		"startDate": c.first,
		"endDate":   c.last,
		"metricID":  m.id,
		"actTypeID": actType,
	}, nil)
	if err != nil {
		return fmt.Errorf("get group leaderboard: %w", err)
	}
	if outfmt.IsJSON(g.Context) {
		return writeObject(g, data)
	}
	var answer struct {
		AllMetrics struct {
			MetricsMap map[string][]map[string]any `json:"metricsMap"`
		} `json:"allMetrics"`
	}
	if err := unmarshalAnswer(data, &answer); err != nil {
		return fmt.Errorf("parse group leaderboard: %w", err)
	}
	entries := answer.AllMetrics.MetricsMap[m.key]
	for i, e := range entries {
		if e != nil {
			e["rank"] = c.Offset + i + 1
		}
	}
	ranked, err := json.Marshal(map[string]any{"entries": entries})
	if err != nil {
		return err
	}
	return Rows{Noun: "group leaderboard", Items: "entries", Columns: []Column{
		{Label: "RANK", Path: "rank"},
		{Label: "NAME", Path: "userInfo.fullname"},
		{Label: "VALUE", Path: "value", Format: m.format},
	}}.print(g, client, ranked)
}

type GroupsCreateCmd struct {
	Name        FreeText `arg:"" help:"Group name."`
	Description string   `help:"Group description."`
	Location    string   `help:"Group location."`
	Visibility  string   `help:"Who can find the group. A hidden group is invite-only." enum:"VISIBLE,HIDDEN" default:"VISIBLE"`
	Privacy     *string  `help:"Who can join: PUBLIC, PRIVATE or INVITE. Defaults to PUBLIC, or INVITE for a hidden group." enum:"PUBLIC,PRIVATE,INVITE" placeholder:"PRIVACY"`
	Image       string   `help:"Default group image." enum:"running_01,roadcycling_01,mountain_cycling_01,swimming_01,health_fitness_01,golf_01,para_sports_01" default:"running_01"`
}

func (c *GroupsCreateCmd) Validate() error {
	if c.Privacy != nil {
		return garminapi.CheckGroupAccess(c.Visibility, *c.Privacy)
	}
	return nil
}

func (c *GroupsCreateCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	privacy := ""
	if c.Privacy != nil {
		privacy = *c.Privacy
	}
	data, err := client.CreateGroup(g.Context, garminapi.NewGroup{
		Name:        string(c.Name),
		Description: c.Description,
		Location:    c.Location,
		Visibility:  c.Visibility,
		Privacy:     privacy,
		Image:       c.Image,
	})
	if err != nil {
		return fmt.Errorf("create group: %w", err)
	}
	var resp map[string]any
	_ = json.Unmarshal(data, &resp)
	if id := jsonString(resp, "id"); id != "" {
		return reportWrite(g, data, "Created group %q (ID: %s)", c.Name, id)
	}
	return reportWrite(g, data, "Created group %q", c.Name)
}

type GroupsEditCmd struct {
	ID          NumericID `arg:"" help:"Group ID."`
	Name        *string   `help:"New name." placeholder:"STRING"`
	Description *string   `help:"New description. An empty text clears it." placeholder:"STRING"`
	Location    *string   `help:"New location. An empty text clears it." placeholder:"STRING"`
	Visibility  *string   `help:"Who can find the group. Hiding a group makes it invite-only." enum:"VISIBLE,HIDDEN" placeholder:"VISIBILITY"`
	Privacy     *string   `help:"Who can join: PUBLIC, PRIVATE or INVITE." enum:"PUBLIC,PRIVATE,INVITE" placeholder:"PRIVACY"`
}

func (c *GroupsEditCmd) Validate() error {
	if c.ID < 1 {
		return errors.New("the group ID must be a positive number")
	}
	if c.Name == nil && c.Description == nil && c.Location == nil && c.Visibility == nil && c.Privacy == nil {
		return errors.New("nothing to change: give --name, --description, --location, --visibility or --privacy")
	}
	if c.Name != nil && strings.TrimSpace(*c.Name) == "" {
		return errors.New("the group name must not be empty")
	}
	return nil
}

func (c *GroupsEditCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.UpdateGroup(g.Context, c.ID.String(), garminapi.GroupEdit{
		Name:        c.Name,
		Description: c.Description,
		Location:    c.Location,
		Visibility:  c.Visibility,
		Privacy:     c.Privacy,
	})
	if err != nil {
		return fmt.Errorf("edit group: %w", err)
	}
	return reportWrite(g, data, "Updated group %s", c.ID)
}
