package cmd

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bpauli/gccli/internal/garminapi"
)

var activityCommentsGroup = Group{Name: "comments", Help: "List the comments on an activity.", Children: []Node{
	Leaf{
		Name: "list", Help: "List the comments on an activity.", Default: true,
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.comments",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get comments", Output: JSON{List: true},
	},
}}

var activityLikesGroup = Group{Name: "likes", Help: "List, add and remove likes of an activity.", Children: []Node{
	Leaf{
		Name: "list", Help: "List the likes of an activity.", Default: true,
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.likes",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get likes", Output: JSON{List: true},
	},
	Leaf{
		Name: "add", Help: "Like an activity.",
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.like",
		Params:   Params{"activityID": In("ID")},
		Verb:     "like activity", Output: Write{Done: "Liked activity {ID}"},
	},
	Custom{Name: "remove", Help: "Remove your like from an activity.", Cmd: &ActivityLikesRemoveCmd{}},
}}

// ActivityLikesRemoveCmd removes the like of the user from an activity.
type ActivityLikesRemoveCmd struct {
	ID ID `arg:"" help:"Activity ID."`
}

// Run reads the conversation of the activity before the unlike. Garmin
// answers 204 to an unlike whether a like was there or not, and the unlike
// path takes the conversation UUID, not the activity ID. A 404 means that
// the activity has no conversation, so no like.
func (c *ActivityLikesRemoveCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	notLiked := fmt.Errorf("activity %s is not liked", c.ID)
	data, err := client.Call(g.Context, "activities.conversation", map[string]string{"activityID": string(c.ID)}, nil)
	var apiErr *garminapi.GarminAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return notLiked
	}
	if err != nil {
		return fmt.Errorf("get conversation: %w", err)
	}
	var conversation struct {
		UUID  string `json:"conversationUuid"`
		Liked bool   `json:"likedByUser"`
	}
	if err := unmarshalAnswer(data, &conversation); err != nil {
		return fmt.Errorf("parse conversation: %w", err)
	}
	if !conversation.Liked {
		return notLiked
	}
	if conversation.UUID == "" {
		return errors.New("parse conversation: no conversationUuid")
	}
	data, err = client.Call(g.Context, "activities.unlike", map[string]string{"conversationUUID": conversation.UUID}, nil)
	if err != nil {
		return fmt.Errorf("remove like: %w", err)
	}
	return reportWrite(g, data, "Removed like from activity %s", c.ID)
}
