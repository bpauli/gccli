package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

var groupsGroup = Group{Name: "groups", Help: "List, search, join and leave groups.", Children: []Node{
	Leaf{
		Name: "list", Help: "List your groups.", Default: true,
		Endpoint: "groups.list",
		Params:   Params{"displayName": DisplayName},
		Verb:     "get groups", Output: JSON{List: true},
	},
	Custom{Name: "search", Help: "Search groups by keyword.", Cmd: &GroupsSearchCmd{}},
	Leaf{
		Name: "show", Help: "Show a group.",
		Spec:     argsOf(&struct{ ID ID `arg:"" help:"Group ID."` }{}),
		Endpoint: "groups.get",
		Params:   Params{"groupID": In("ID")},
		Verb:     "get group", Output: JSON{},
	},
	Leaf{
		Name: "members", Help: "List the members of a group.",
		Spec:     argsOf(&struct{ ID ID `arg:"" help:"Group ID."` }{}),
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
		Spec:     argsOf(&struct{ ID ID `arg:"" help:"Group ID."` }{}),
		Endpoint: "groups.feed",
		Params:   Params{"groupID": In("ID")},
		Verb:     "get group feed", Output: JSON{},
	},
	Leaf{
		Name: "activities", Help: "List the latest activities of the group members.",
		Spec:     argsOf(&struct{ ID ID `arg:"" help:"Group ID."` }{}),
		Endpoint: "groups.activities",
		Params:   Params{"groupID": In("ID")},
		Verb:     "get group activities",
		Output:   Rows{Noun: "group activities", Items: "activityList", Columns: activityColumns},
	},
	Leaf{
		Name: "challenges", Help: "List the open challenges of a group. Garmin answers 403 to a non-member.",
		Spec:     argsOf(&struct{ ID ID `arg:"" help:"Group ID."` }{}),
		Endpoint: "groups.challenges",
		Params:   Params{"groupID": In("ID")},
		Verb:     "get group challenges", Output: JSON{List: true},
	},
	Custom{Name: "join", Help: "Join a public group. gccli does not send the request that a private group needs.", Cmd: &GroupsJoinCmd{}},
	Leaf{
		Name: "leave", Help: "Leave a group.",
		Spec:     argsOf(&struct{ ID ID `arg:"" help:"Group ID."` }{}),
		Endpoint: "groups.leave",
		Params:   Params{"groupID": In("ID"), "profileID": ProfilePK},
		Verb:     "leave group",
		Output:   Write{Done: "Left group {ID}", Confirm: "Leave group {ID}?"},
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
