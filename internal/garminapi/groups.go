package garminapi

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var groupsEndpoints = []Endpoint{
	{Name: "groups.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/group-service/groups/{displayName}?includeMembership=true"},
	// groups.search takes keyword, start and limit as a form body. start is
	// a 1-based item index: start=0 and start=1 give the same first group.
	{Name: "groups.search", Method: http.MethodPost, Host: ConnectAPI, Path: "/group-service/keyword", Body: true, Form: true, Read: true},
	{Name: "groups.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/group-service/group/{groupID}?includeMembership=true"},
	{Name: "groups.members", Method: http.MethodGet, Host: ConnectAPI, Path: "/group-service/group/{groupID}/members?supportOwner=true&orderByRelationship=true"},
	// groups.feed answers 207 with the feed, also for a member.
	{Name: "groups.feed", Method: http.MethodGet, Host: ConnectAPI, Path: "/web-gateway/group/v2/{groupID}/groupFeed"},
	{Name: "groups.activities", Method: http.MethodGet, Host: ConnectAPI, Path: "/web-gateway/group/v2/{groupID}/activityFeed"},
	// groups.challenges answers 403 ForbiddenException to an account that is
	// not a member of the group.
	{Name: "groups.challenges", Method: http.MethodGet, Host: ConnectAPI, Path: "/groupchallenge-service/group/{groupID}/groupchallenge/nonCompleted"},
	// groups.join adds the account to a public group at once. A private
	// group needs a join request, which is a different request.
	{Name: "groups.join", Method: http.MethodPost, Host: ConnectAPI, Path: "/group-service/group/{groupID}/member/{profileID}", Body: true},
	{Name: "groups.leave", Method: http.MethodDelete, Host: ConnectAPI, Path: "/group-service/group/{groupID}/member/{profileID}"},
	{Name: "groups.create", Method: http.MethodPost, Host: ConnectAPI, Path: "/group-service/group", Body: true},
	{Name: "groups.update", Method: http.MethodPut, Host: ConnectAPI, Path: "/group-service/group/{groupID}", Body: true},
	{Name: "groups.delete", Method: http.MethodDelete, Host: ConnectAPI, Path: "/group-service/group/{groupID}"},
}

const (
	GroupVisible = "VISIBLE"
	GroupHidden  = "HIDDEN"
	GroupPublic  = "PUBLIC"
	GroupInvite  = "INVITE"
)

// GroupImages are the default image names of the web form.
var GroupImages = []string{"running_01", "roadcycling_01", "mountain_cycling_01", "swimming_01", "health_fitness_01", "golf_01", "para_sports_01"}

// CheckGroupAccess rejects a hidden group that is not invite-only.
func CheckGroupAccess(visibility, privacy string) error {
	if visibility == GroupHidden && privacy != GroupInvite {
		return errors.New("a hidden group is invite-only: use --privacy INVITE or --visibility VISIBLE")
	}
	return nil
}

// NewGroup is the input of a new group. Visibility is VISIBLE or HIDDEN,
// Privacy is PUBLIC, PRIVATE or INVITE, and Image is one of GroupImages. An
// empty Visibility means VISIBLE, an empty Image means the first of
// GroupImages, and an empty Privacy means INVITE for a hidden group and
// PUBLIC for a visible one.
type NewGroup struct {
	Name        string
	Description string
	Location    string
	Visibility  string
	Privacy     string
	Image       string
}

func (c *Client) CreateGroup(ctx context.Context, g NewGroup) (json.RawMessage, error) {
	g.Visibility = cmp.Or(g.Visibility, GroupVisible)
	g.Image = cmp.Or(g.Image, GroupImages[0])
	if g.Privacy == "" {
		g.Privacy = GroupPublic
		if g.Visibility == GroupHidden {
			g.Privacy = GroupInvite
		}
	}
	if err := CheckGroupAccess(g.Visibility, g.Privacy); err != nil {
		return nil, err
	}
	body, err := json.Marshal(map[string]any{
		"groupName":             g.Name,
		"groupDescription":      g.Description,
		"location":              g.Location,
		"groupVisibility":       g.Visibility,
		"groupPrivacy":          g.Privacy,
		"defaultImageName":      g.Image,
		"profileImageUrlLarge":  nil,
		"profileImageUrlMedium": nil,
		"profileImageUrlSmall":  nil,
		"facebookUrl":           "",
		"twitterUrl":            "",
		"websiteUrl":            "",
		"primaryActivities":     []any{},
		"otherPrimaryActivity":  "",
		"activityFeedTypes":     []any{},
		"leaderboardTypes":      []any{},
		"featureTypes":          []any{},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal group: %w", err)
	}
	return c.Call(ctx, "groups.create", nil, body)
}

// GroupEdit holds the new values of a group update. A nil field keeps the
// saved value, and an empty Description or Location clears the text.
type GroupEdit struct {
	Name        *string
	Description *string
	Location    *string
	Visibility  *string
	Privacy     *string
}

// UpdateGroup reads the group, sets the fields of e and sends the full object
// back, because Garmin rejects a partial body. Hiding a group makes it
// invite-only.
func (c *Client) UpdateGroup(ctx context.Context, id string, e GroupEdit) (json.RawMessage, error) {
	params := map[string]string{"groupID": id}
	data, err := c.Call(ctx, "groups.get", params, nil)
	if err != nil {
		return nil, err
	}
	var saved struct {
		Visibility string `json:"groupVisibility"`
		Privacy    string `json:"groupPrivacy"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, fmt.Errorf("parse group %s: not a JSON object", id)
	}
	fields := map[string]any{}
	for key, value := range map[string]*string{"groupName": e.Name, "groupDescription": e.Description, "location": e.Location} {
		if value != nil {
			fields[key] = *value
		}
	}
	if e.Visibility != nil || e.Privacy != nil {
		visibility, privacy := saved.Visibility, saved.Privacy
		if e.Visibility != nil {
			visibility = *e.Visibility
			if visibility == GroupHidden && e.Privacy == nil {
				privacy = GroupInvite
			}
		}
		if e.Privacy != nil {
			privacy = *e.Privacy
		}
		if err := CheckGroupAccess(visibility, privacy); err != nil {
			return nil, err
		}
		fields["groupVisibility"], fields["groupPrivacy"] = visibility, privacy
	}
	body, err := setFields(data, "group "+id, fields)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, "groups.update", params, body)
}
