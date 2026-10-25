package cmd

import (
	"fmt"
	"net/url"
	"strconv"
)

var groupsGroup = Group{Name: "groups", Help: "List, search and view groups.", Children: []Node{
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
