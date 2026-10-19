//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

const e2eNotePrefix = "E2E_TEST_"

// TestAddUpdateMoveDeleteCalendarNote adds a calendar note, changes its name,
// moves it to the next day and deletes it, with a GET after each step.
func TestAddUpdateMoveDeleteCalendarNote(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	cleanupOrphanNotes(t, client)

	today := time.Now()
	tomorrow := today.AddDate(0, 0, 1)
	name := fmt.Sprintf("%sNOTE_%d", e2eNotePrefix, today.UnixNano())
	body, err := json.Marshal(map[string]any{"noteName": name, "content": "E2E content", "date": today.Format("2006-01-02")})
	if err != nil {
		t.Fatal(err)
	}
	data, err := client.Call(ctx, "calendar.note.create", nil, body)
	if err != nil {
		t.Fatalf("calendar.note.create failed: %v", err)
	}
	var created struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal(data, &created); err != nil || created.ID == "" {
		t.Fatalf("created note has no id (err %v)", err)
	}
	id := created.ID.String()
	t.Log("Added calendar note")

	note := map[string]string{"noteID": id}
	RegisterCleanup(t, func() {
		if _, err := client.Call(context.Background(), "calendar.note.delete", note, nil); err != nil && !isNotFound(err) {
			t.Logf("WARNING: failed to clean up calendar note: %v", err)
		}
	})

	got := readNote(t, client, id)
	if got.Name != name || got.Content != "E2E content" || got.Date != today.Format("2006-01-02") {
		t.Error("note GET does not match the added note")
	}
	if !monthHasNote(t, client, today, id) {
		t.Error("month view does not list the added note")
	}

	time.Sleep(2 * time.Second)
	renamed := name + "_2"
	if _, err := client.UpdateNote(ctx, id, garminapi.NoteEdit{Name: &renamed}); err != nil {
		t.Fatalf("UpdateNote failed: %v", err)
	}
	got = readNote(t, client, id)
	if got.Name != renamed {
		t.Error("note name did not change")
	}
	if got.Content != "E2E content" || got.Date != today.Format("2006-01-02") {
		t.Error("update changed the text or the date")
	}

	time.Sleep(2 * time.Second)
	if _, err := client.Call(ctx, "calendar.note.move", map[string]string{"noteID": id, "date": tomorrow.Format("2006-01-02")}, nil); err != nil {
		t.Fatalf("calendar.note.move failed: %v", err)
	}
	if got := readNote(t, client, id); got.Date != tomorrow.Format("2006-01-02") {
		t.Error("note date is not the next day after the move")
	}

	time.Sleep(2 * time.Second)
	if _, err := client.Call(ctx, "calendar.note.delete", note, nil); err != nil {
		t.Fatalf("calendar.note.delete failed: %v", err)
	}
	t.Log("Deleted calendar note")
	if _, err := client.Call(ctx, "calendar.note.get", note, nil); !isNotFound(err) {
		t.Errorf("note GET after delete: %v, want 404", err)
	}
}

type calendarNote struct {
	Name    string `json:"noteName"`
	Content string `json:"content"`
	Date    string `json:"date"`
}

func readNote(t *testing.T, client *garminapi.Client, id string) calendarNote {
	t.Helper()
	data, err := client.Call(context.Background(), "calendar.note.get", map[string]string{"noteID": id}, nil)
	if err != nil {
		t.Fatalf("calendar.note.get failed: %v", err)
	}
	var n calendarNote
	if err := json.Unmarshal(data, &n); err != nil {
		t.Fatalf("unmarshal calendar note: %v", err)
	}
	return n
}

type calendarItem struct {
	ID       json.Number `json:"id"`
	ItemType string      `json:"itemType"`
	Title    string      `json:"title"`
}

// monthItems reads the calendar month view of day. The month in the path is
// 0-based. No command reads the month view yet, so there is no table row.
func monthItems(t *testing.T, client *garminapi.Client, day time.Time) ([]calendarItem, error) {
	t.Helper()
	path := fmt.Sprintf("/calendar-service/year/%d/month/%d", day.Year(), int(day.Month())-1)
	data, err := client.ConnectAPI(context.Background(), "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var month struct {
		CalendarItems []calendarItem `json:"calendarItems"`
	}
	if err := json.Unmarshal(data, &month); err != nil {
		return nil, err
	}
	return month.CalendarItems, nil
}

func monthHasNote(t *testing.T, client *garminapi.Client, day time.Time, id string) bool {
	t.Helper()
	items, err := monthItems(t, client, day)
	if err != nil {
		t.Fatalf("read calendar month: %v", err)
	}
	for _, it := range items {
		if it.ItemType == "note" && it.ID.String() == id {
			return true
		}
	}
	return false
}

// cleanupOrphanNotes deletes calendar notes with the E2E_TEST_ prefix left
// over from prior failed runs in this month and the next one, the months the
// test writes to.
func cleanupOrphanNotes(t *testing.T, client *garminapi.Client) {
	t.Helper()
	now := time.Now()
	for _, day := range []time.Time{now, now.AddDate(0, 1, 1-now.Day())} {
		items, err := monthItems(t, client, day)
		if err != nil {
			t.Logf("WARNING: could not read the calendar month for orphan cleanup: %v", err)
			continue
		}
		for _, it := range items {
			if it.ItemType != "note" || !strings.HasPrefix(it.Title, e2eNotePrefix) || it.ID == "" {
				continue
			}
			if _, err := client.Call(context.Background(), "calendar.note.delete", map[string]string{"noteID": it.ID.String()}, nil); err != nil && !isNotFound(err) {
				t.Logf("WARNING: failed to delete orphaned calendar note: %v", err)
			} else {
				t.Log("Cleaned up orphaned calendar note")
			}
		}
	}
}
