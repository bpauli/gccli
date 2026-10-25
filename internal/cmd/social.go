package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"

	"github.com/bpauli/gccli/internal/garminapi"
)

var socialGroup = Group{Name: "social", Help: "Follow, mute and search users. View connections and the newsfeed.", Children: []Node{
	Leaf{
		Name: "connections", Help: "List your connections.",
		Spec: argsOf(&struct {
			Pagination `embed:"" set:"items=connections"`
		}{}),
		Endpoint: "connections.list",
		Params:   Params{"start": In("Offset"), "limit": In("Limit")},
		Verb:     "get connections", Output: JSON{},
	},
	Leaf{
		Name: "followers", Help: "List the followers of a user.",
		Spec: argsOf(&struct {
			DisplayName string `arg:"" optional:"" help:"Display name of the user. Defaults to your display name."`
		}{}),
		Endpoint: "followers.list",
		Params:   Params{"displayName": In("DisplayName").Or(DisplayName)},
		Verb:     "get followers", Output: JSON{List: true},
	},
	Leaf{
		Name: "following", Help: "List the users that a user follows.",
		Spec: argsOf(&struct {
			DisplayName string `arg:"" optional:"" help:"Display name of the user. Defaults to your display name."`
		}{}),
		Endpoint: "following.list",
		Params:   Params{"displayName": In("DisplayName").Or(DisplayName)},
		Verb:     "get following", Output: JSON{List: true},
	},
	Leaf{Name: "requests", Help: "List pending connection requests.", Endpoint: "connections.pending", Verb: "get connection requests", Output: JSON{List: true}},
	Leaf{Name: "feed", Help: "Show your newsfeed.", Endpoint: "newsfeed.list", Verb: "get newsfeed", Output: JSON{}},
	Custom{Name: "follow", Help: "Follow a user. A user with a private profile gets a follow request.", Cmd: &SocialFollowCmd{}},
	Custom{Name: "unfollow", Help: "Stop following a user.", Cmd: &SocialUnfollowCmd{}},
	Leaf{Name: "follow-requests", Help: "List the users who asked to follow you.", Endpoint: "follow-requests.list", Verb: "get follow requests", Output: JSON{}},
	Custom{Name: "accept", Help: "Accept the follow request of a user.", Cmd: &SocialAcceptCmd{}},
	Custom{Name: "decline", Help: "Decline the follow request of a user.", Cmd: &SocialDeclineCmd{}},
	Custom{Name: "remove-follower", Help: "Remove a user from your followers.", Cmd: &SocialRemoveFollowerCmd{}},
	Leaf{
		Name: "mute", Help: "Mute a user that you follow.",
		Spec: argsOf(&struct {
			DisplayName ID `arg:"" help:"Display name of the user."`
		}{}),
		Endpoint: "muted.add",
		Params:   Params{"displayName": In("DisplayName")},
		Verb:     "mute user", Output: Write{Done: "Muted {DisplayName}"},
	},
	Custom{Name: "unmute", Help: "Unmute a user.", Cmd: &SocialUnmuteCmd{}},
	Leaf{Name: "muted", Help: "List the users that you muted.", Endpoint: "muted.list", Verb: "get muted users", Output: JSON{}},
	Custom{Name: "search", Help: "Search Garmin Connect users by name.", Cmd: &SocialSearchCmd{}},
}}

// The relationshipStatus values of a follow status, seen from you: you
// follow the user, the user follows you, or both.
const (
	relFollowing = "FOLLOWING"
	relFollowed  = "FOLLOWED"
	relMutual    = "MUTUAL"
)

// relationship reads how you and a user follow each other.
func relationship(g *Globals, client *garminapi.Client, displayName ID) (string, error) {
	data, err := client.Call(g.Context, "follow.status", map[string]string{"displayName": string(displayName)}, nil)
	if err != nil {
		return "", fmt.Errorf("get follow status: %w", err)
	}
	var status struct {
		Profile struct {
			Follow struct {
				Relationship string `json:"relationshipStatus"`
			} `json:"followStatus"`
		} `json:"userFollowProfile"`
	}
	if err := unmarshalAnswer(data, &status); err != nil {
		return "", fmt.Errorf("parse follow status: %w", err)
	}
	return status.Profile.Follow.Relationship, nil
}

// SocialFollowCmd follows a user. Garmin answers with a follow request,
// whose status is CREATED while the user must still accept it.
type SocialFollowCmd struct {
	DisplayName ID `arg:"" help:"Display name of the user."`
}

func (c *SocialFollowCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "following.add", map[string]string{"displayName": string(c.DisplayName)}, nil)
	if err != nil {
		return fmt.Errorf("follow user: %w", err)
	}
	var req struct {
		Status string `json:"requestStatus"`
	}
	if unmarshalAnswer(data, &req) == nil && req.Status == "CREATED" {
		return reportWrite(g, data, "Sent a follow request to %s", c.DisplayName)
	}
	return reportWrite(g, data, "Followed %s", c.DisplayName)
}

// SocialUnfollowCmd stops following a user. Garmin answers 204 whether you
// follow the user or not, so the follow status comes first.
type SocialUnfollowCmd struct {
	DisplayName ID   `arg:"" help:"Display name of the user."`
	Force       bool `help:"Skip confirmation prompt." short:"f"`
}

func (c *SocialUnfollowCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	ok, err := askFollowRemoval(g, client, c.DisplayName, c.Force, []string{relFollowing, relMutual}, "you do not follow %s", "Unfollow %s?")
	if err != nil || !ok {
		return err
	}
	data, err := client.Call(g.Context, "following.remove", map[string]string{"displayName": string(c.DisplayName)}, nil)
	if err != nil {
		return fmt.Errorf("unfollow user: %w", err)
	}
	return reportWrite(g, data, "Unfollowed %s", c.DisplayName)
}

// SocialRemoveFollowerCmd removes a user from your followers. Garmin answers
// 204 whether the user follows you or not, so the follow status comes first.
type SocialRemoveFollowerCmd struct {
	DisplayName ID   `arg:"" help:"Display name of the user."`
	Force       bool `help:"Skip confirmation prompt." short:"f"`
}

func (c *SocialRemoveFollowerCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	ok, err := askFollowRemoval(g, client, c.DisplayName, c.Force, []string{relFollowed, relMutual}, "%s does not follow you", "Remove %s from your followers?")
	if err != nil || !ok {
		return err
	}
	data, err := client.Call(g.Context, "followers.remove", map[string]string{"displayName": string(c.DisplayName)}, nil)
	if err != nil {
		return fmt.Errorf("remove follower: %w", err)
	}
	return reportWrite(g, data, "Removed follower %s", c.DisplayName)
}

// askFollowRemoval checks that the relationship with the user is one of
// need, and then asks. missing and prompt take the display name. ok is false
// when the user cancels.
func askFollowRemoval(g *Globals, client *garminapi.Client, displayName ID, force bool, need []string, missing, prompt string) (bool, error) {
	rel, err := relationship(g, client, displayName)
	if err != nil {
		return false, err
	}
	if !slices.Contains(need, rel) {
		return false, fmt.Errorf(missing, displayName)
	}
	ok, err := confirm(g.Stderr, fmt.Sprintf(prompt, displayName), force)
	if err == nil && !ok {
		g.UI.Infof("Cancelled")
	}
	return ok, err
}

// SocialAcceptCmd accepts the follow request of a user.
type SocialAcceptCmd struct {
	DisplayName ID `arg:"" help:"Display name of the user, as social follow-requests lists it."`
}

func (c *SocialAcceptCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	id, err := followRequestID(g, client, c.DisplayName)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "follow-requests.accept", map[string]string{"requestID": id}, nil)
	if err != nil {
		return fmt.Errorf("accept follow request: %w", err)
	}
	return reportWrite(g, data, "Accepted the follow request of %s", c.DisplayName)
}

// SocialDeclineCmd declines the follow request of a user.
type SocialDeclineCmd struct {
	DisplayName ID `arg:"" help:"Display name of the user, as social follow-requests lists it."`
}

func (c *SocialDeclineCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	id, err := followRequestID(g, client, c.DisplayName)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "follow-requests.decline", map[string]string{"requestID": id}, nil)
	if err != nil {
		return fmt.Errorf("decline follow request: %w", err)
	}
	return reportWrite(g, data, "Declined the follow request of %s", c.DisplayName)
}

type inboxPage struct {
	Requests []struct {
		Data struct {
			DisplayName string `json:"displayName"`
			ID          *int64 `json:"followRequestId"`
		} `json:"data"`
	} `json:"requests"`
	NextCursor json.RawMessage `json:"nextCursor"`
}

// followRequestID finds the ID of the follow request of a user in the inbox,
// page by page. The accept and decline paths take this ID, and the pending
// list of social follow-requests has it as null.
func followRequestID(g *Globals, client *garminapi.Client, displayName ID) (string, error) {
	cursor := ""
	for {
		data, err := client.Call(g.Context, "follow-requests.inbox", map[string]string{"cursor": cursor}, nil)
		if err != nil {
			return "", fmt.Errorf("get follow requests: %w", err)
		}
		var page inboxPage
		if err := unmarshalAnswer(data, &page); err != nil {
			return "", fmt.Errorf("parse follow requests: %w", err)
		}
		for _, r := range page.Requests {
			if r.Data.DisplayName == string(displayName) && r.Data.ID != nil {
				return strconv.FormatInt(*r.Data.ID, 10), nil
			}
		}
		next := cursorText(page.NextCursor)
		if next == "" || next == cursor {
			return "", fmt.Errorf("no follow request from %s", displayName)
		}
		cursor = next
	}
}

// cursorText is the inbox cursor as a query value: a JSON string without its
// quotes, a number as written, and "" for null or no cursor.
func cursorText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	return ""
}

// SocialUnmuteCmd unmutes a user. Garmin answers 204 whether the user is
// muted or not, so the muted list comes first.
type SocialUnmuteCmd struct {
	DisplayName ID `arg:"" help:"Display name of the user."`
}

func (c *SocialUnmuteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "muted.list", nil, nil)
	if err != nil {
		return fmt.Errorf("get muted users: %w", err)
	}
	var muted struct {
		Users []struct {
			DisplayName string `json:"displayName"`
		} `json:"mutedUserList"`
	}
	if err := unmarshalAnswer(data, &muted); err != nil {
		return fmt.Errorf("parse muted users: %w", err)
	}
	found := false
	for _, u := range muted.Users {
		found = found || u.DisplayName == string(c.DisplayName)
	}
	if !found {
		return fmt.Errorf("%s is not muted", c.DisplayName)
	}
	data, err = client.Call(g.Context, "muted.remove", map[string]string{"displayName": string(c.DisplayName)}, nil)
	if err != nil {
		return fmt.Errorf("unmute user: %w", err)
	}
	return reportWrite(g, data, "Unmuted %s", c.DisplayName)
}

// SocialSearchCmd searches users by name.
type SocialSearchCmd struct {
	Query      FreeText `arg:"" help:"Name to search for, at least 2 characters."`
	Pagination `embed:"" set:"items=users"`
}

var userSearchRows = Rows{Noun: "users", Items: "profileList", Columns: []Column{
	{Label: "DISPLAY NAME", Path: "displayName"},
	{Label: "NAME", Path: "fullName"},
	{Label: "LOCATION", Path: "location"},
	{Label: "FOLLOW", Path: "followStatus.relationshipStatus"},
}}

func (c *SocialSearchCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	// Garmin counts start from 1.
	form := url.Values{"keyword": {string(c.Query)}, "start": {strconv.Itoa(c.Offset + 1)}, "limit": {strconv.Itoa(c.Limit)}}
	data, err := client.Call(g.Context, "users.search", nil, []byte(form.Encode()))
	if err != nil {
		return fmt.Errorf("search users: %w", err)
	}
	return userSearchRows.print(g, client, data)
}
