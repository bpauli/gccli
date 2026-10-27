package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
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

func (c *GroupsJoinCmd) Validate() error {
	if c.ID < 1 {
		return errors.New("the group ID must be a positive number")
	}
	return nil
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
