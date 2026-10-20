//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/bpauli/gccli/internal/garminapi"
)

// TestLikeUnlikeActivity likes the first activity, reads its likes, removes
// the like by the conversation UUID and reads the likes again. It skips an
// activity that is already liked, so it never removes a like it did not
// add.
func TestLikeUnlikeActivity(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	data, err := client.Call(ctx, "activities.list", map[string]string{"start": "0", "limit": "1"}, nil)
	if err != nil {
		t.Fatalf("activities.list failed: %v", err)
	}
	var activities []map[string]any
	if err := json.Unmarshal(data, &activities); err != nil {
		t.Fatalf("unmarshal activities: %v", err)
	}
	if len(activities) == 0 {
		t.Skip("no activities in account")
	}
	id := map[string]string{"activityID": formatID(activities[0]["activityId"])}

	if _, liked := likeState(t, client, id); liked {
		t.Skip("the first activity is already liked")
	}
	before := likeCount(t, client, id)

	data, err = client.Call(ctx, "activities.like", id, nil)
	if err != nil {
		t.Fatalf("activities.like failed: %v", err)
	}
	var like struct {
		UUID string `json:"conversationUuid"`
	}
	if err := json.Unmarshal(data, &like); err != nil || like.UUID == "" {
		t.Fatalf("like answer %s has no conversationUuid (err %v)", data, err)
	}
	RegisterCleanup(t, func() {
		if uuid, liked := likeState(t, client, id); liked {
			if _, err := client.Call(context.Background(), "activities.unlike", map[string]string{"conversationUUID": uuid}, nil); err != nil {
				t.Errorf("cleanup: activities.unlike failed: %v", err)
			}
		}
	})

	uuid, liked := likeState(t, client, id)
	if !liked || uuid != like.UUID {
		t.Fatalf("after the like: likedByUser %v, conversationUuid %q, want true and %q", liked, uuid, like.UUID)
	}
	if got := likeCount(t, client, id); got != before+1 {
		t.Fatalf("after the like: %d likes, want %d", got, before+1)
	}

	if _, err := client.Call(ctx, "activities.unlike", map[string]string{"conversationUUID": uuid}, nil); err != nil {
		t.Fatalf("activities.unlike failed: %v", err)
	}
	if _, liked := likeState(t, client, id); liked {
		t.Error("after the unlike: likedByUser is still true")
	}
	if got := likeCount(t, client, id); got != before {
		t.Errorf("after the unlike: %d likes, want %d", got, before)
	}
}

// likeState reads the conversation of an activity: its UUID and whether
// the user likes it. A 404 means no conversation, so no like.
func likeState(t *testing.T, client *garminapi.Client, id map[string]string) (string, bool) {
	t.Helper()
	data, err := client.Call(context.Background(), "activities.conversation", id, nil)
	if isNotFound(err) {
		return "", false
	}
	if err != nil {
		t.Fatalf("activities.conversation failed: %v", err)
	}
	var conversation struct {
		UUID  string `json:"conversationUuid"`
		Liked bool   `json:"likedByUser"`
	}
	if err := json.Unmarshal(data, &conversation); err != nil {
		t.Fatalf("unmarshal conversation: %v", err)
	}
	return conversation.UUID, conversation.Liked
}

func likeCount(t *testing.T, client *garminapi.Client, id map[string]string) int {
	t.Helper()
	data, err := client.Call(context.Background(), "activities.likes", id, nil)
	if err != nil {
		t.Fatalf("activities.likes failed: %v", err)
	}
	var likes []json.RawMessage
	if err := json.Unmarshal(data, &likes); err != nil {
		t.Fatalf("unmarshal likes: %v", err)
	}
	return len(likes)
}
