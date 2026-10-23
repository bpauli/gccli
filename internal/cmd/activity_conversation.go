package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/bpauli/gccli/internal/garminapi"
)

var activityCommentsGroup = Group{Name: "comments", Help: "List, add, edit and delete the comments on an activity, and like comments.", Children: []Node{
	Leaf{
		Name: "list", Help: "List the comments on an activity.", Default: true,
		Spec: argsOf(&struct {
			ID ID `arg:"" help:"Activity ID."`
		}{}),
		Endpoint: "activities.comments",
		Params:   Params{"activityID": In("ID")},
		Verb:     "get comments", Output: JSON{List: true},
	},
	Custom{Name: "add", Help: "Add a comment to an activity.", Cmd: &ActivityCommentsAddCmd{}},
	Custom{Name: "edit", Help: "Change the text of your comment on an activity.", Cmd: &ActivityCommentsEditCmd{}},
	Custom{Name: "delete", Help: "Delete a comment on an activity.", Cmd: &ActivityCommentsDeleteCmd{}},
	activityCommentLikesGroup,
}}

var activityCommentLikesGroup = Group{Name: "likes", Help: "List, add and remove likes of a comment.", Children: []Node{
	Leaf{
		Name: "list", Help: "List the likes of a comment.", Default: true,
		Spec: argsOf(&struct {
			CommentID ID `arg:"" help:"Comment ID (conversationCommentUuid in activity comments list)."`
		}{}),
		Endpoint: "activities.comment-likes",
		Params:   Params{"commentUUID": In("CommentID")},
		Verb:     "get comment likes", Output: JSON{List: true},
	},
	Leaf{
		Name: "add", Help: "Like a comment.",
		Spec: argsOf(&struct {
			CommentID ID `arg:"" help:"Comment ID (conversationCommentUuid in activity comments list)."`
		}{}),
		Endpoint: "activities.like-comment",
		Params:   Params{"commentUUID": In("CommentID")},
		Verb:     "like comment", Output: Write{Done: "Liked comment {CommentID}"},
	},
	Custom{Name: "remove", Help: "Remove your like from a comment.", Cmd: &ActivityCommentLikesRemoveCmd{}},
}}

// ActivityCommentLikesRemoveCmd removes the like of the user from a comment.
// It is a Custom only because a DELETE Leaf must confirm, and a like comes
// back with one add. Garmin answers 403 when the comment is not liked.
type ActivityCommentLikesRemoveCmd struct {
	CommentID ID `arg:"" help:"Comment ID (conversationCommentUuid in activity comments list)."`
}

func (c *ActivityCommentLikesRemoveCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "activities.unlike-comment", map[string]string{"commentUUID": string(c.CommentID)}, nil)
	if err != nil {
		return fmt.Errorf("remove comment like: %w", err)
	}
	return reportWrite(g, data, "Removed like from comment %s", c.CommentID)
}

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
// path takes the conversation UUID, not the activity ID.
func (c *ActivityLikesRemoveCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	notLiked := fmt.Errorf("activity %s is not liked", c.ID)
	conv, err := readConversation(g, client, c.ID)
	if errors.Is(err, errNoConversation) {
		return notLiked
	}
	if err != nil {
		return err
	}
	if !conv.Liked {
		return notLiked
	}
	if err := conv.needUUID(); err != nil {
		return err
	}
	data, err := client.Call(g.Context, "activities.unlike", map[string]string{"conversationUUID": conv.UUID}, nil)
	if err != nil {
		return fmt.Errorf("remove like: %w", err)
	}
	return reportWrite(g, data, "Removed like from activity %s", c.ID)
}

type conversation struct {
	UUID  string `json:"conversationUuid"`
	Liked bool   `json:"likedByUser"`
}

// errNoConversation means that Garmin answered 404 to the conversation read:
// the activity has no likes and no comments yet.
var errNoConversation = errors.New("no conversation")

// readConversation reads the conversation of an activity, which holds its
// likes and comments. The like and comment paths take its UUID, not the
// activity ID.
func readConversation(g *Globals, client *garminapi.Client, id ID) (conversation, error) {
	var conv conversation
	data, err := client.Call(g.Context, "activities.conversation", map[string]string{"activityID": string(id)}, nil)
	var apiErr *garminapi.GarminAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return conv, errNoConversation
	}
	if err != nil {
		return conv, fmt.Errorf("get conversation: %w", err)
	}
	if err := unmarshalAnswer(data, &conv); err != nil {
		return conv, fmt.Errorf("parse conversation: %w", err)
	}
	return conv, nil
}

func (c conversation) needUUID() error {
	if c.UUID == "" {
		return errors.New("parse conversation: no conversationUuid")
	}
	return nil
}

// commentConversation is the conversation UUID for a write to a comment on
// the activity. An activity without a conversation has no comments.
func commentConversation(g *Globals, client *garminapi.Client, activityID, commentID ID) (string, error) {
	conv, err := readConversation(g, client, activityID)
	if errors.Is(err, errNoConversation) {
		return "", fmt.Errorf("activity %s has no comment %s", activityID, commentID)
	}
	if err != nil {
		return "", err
	}
	return conv.UUID, conv.needUUID()
}

// ActivityCommentsAddCmd adds a comment to an activity.
type ActivityCommentsAddCmd struct {
	ID   ID       `arg:"" help:"Activity ID."`
	Text FreeText `arg:"" help:"Comment text."`
}

func (c *ActivityCommentsAddCmd) Run(g *Globals) error {
	body, err := json.Marshal(map[string]string{"body": string(c.Text)})
	if err != nil {
		return err
	}
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "activities.add-comment", map[string]string{"activityID": string(c.ID)}, body)
	if err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	var comment struct {
		UUID string `json:"conversationCommentUuid"`
	}
	if err := unmarshalAnswer(data, &comment); err != nil || comment.UUID == "" {
		return reportWrite(g, data, "Added comment to activity %s", c.ID)
	}
	return reportWrite(g, data, "Added comment %s to activity %s", comment.UUID, c.ID)
}

// ActivityCommentsEditCmd changes the text of a comment. Garmin lets only
// the author of the comment change it.
type ActivityCommentsEditCmd struct {
	ID        ID       `arg:"" help:"Activity ID."`
	CommentID ID       `arg:"" help:"Comment ID (conversationCommentUuid in activity comments list)."`
	Text      FreeText `arg:"" help:"New comment text."`
}

func (c *ActivityCommentsEditCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	convUUID, err := commentConversation(g, client, c.ID, c.CommentID)
	if err != nil {
		return err
	}
	body, err := json.Marshal(map[string]string{"conversationUuid": convUUID, "conversationCommentUuid": string(c.CommentID), "body": string(c.Text)})
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "activities.edit-comment", map[string]string{"conversationUUID": convUUID, "commentUUID": string(c.CommentID)}, body)
	if err != nil {
		return fmt.Errorf("edit comment: %w", err)
	}
	return reportWrite(g, data, "Changed comment %s on activity %s", c.CommentID, c.ID)
}

// ActivityCommentsDeleteCmd deletes a comment. Garmin lets the author of the
// comment and the owner of the activity delete it.
type ActivityCommentsDeleteCmd struct {
	ID        ID   `arg:"" help:"Activity ID."`
	CommentID ID   `arg:"" help:"Comment ID (conversationCommentUuid in activity comments list)."`
	Force     bool `help:"Skip confirmation prompt." short:"f"`
}

func (c *ActivityCommentsDeleteCmd) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	convUUID, err := commentConversation(g, client, c.ID, c.CommentID)
	if err != nil {
		return err
	}
	ok, err := confirm(g.Stderr, fmt.Sprintf("Delete comment %s on activity %s?", c.CommentID, c.ID), c.Force)
	if err != nil {
		return err
	}
	if !ok {
		g.UI.Infof("Cancelled")
		return nil
	}
	data, err := client.Call(g.Context, "activities.delete-comment", map[string]string{"conversationUUID": convUUID, "commentUUID": string(c.CommentID)}, nil)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	return reportWrite(g, data, "Deleted comment %s on activity %s", c.CommentID, c.ID)
}
