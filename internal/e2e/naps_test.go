//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

// e2eNapLength is the length of the test nap. A nap has no name, so the
// orphan sweep knows a test nap by this length.
const e2eNapLength = 13 * time.Minute

const napTimestamp = "2006-01-02T15:04:05.000"

// TestAddDeleteNap adds a nap that ends a minute ago in the user's Garmin
// time zone, reads it back and deletes it by its date and start.
func TestAddDeleteNap(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	loc, err := client.GetTimeZone(ctx)
	if err != nil {
		t.Fatalf("GetTimeZone failed: %v", err)
	}
	now := time.Now().In(loc)
	end := now.Truncate(time.Minute).Add(-time.Minute)
	start := end.Add(-e2eNapLength)
	if start.Day() != now.Day() {
		t.Skip("too early in the day: Garmin takes only naps that end today")
	}
	date := now.Format("2006-01-02")
	cleanupOrphanNaps(t, client, date)

	pk, err := client.GetProfilePK(ctx)
	if err != nil {
		t.Fatalf("GetProfilePK failed: %v", err)
	}
	body, err := json.Marshal(map[string]any{
		"calendarDate":         date,
		"napStartTimestampGMT": start.UTC().Format(napTimestamp),
		"napEndTimestampGMT":   end.UTC().Format(napTimestamp),
		"userProfilePK":        json.Number(pk),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Call(ctx, "health.add-nap", nil, body); err != nil {
		t.Fatalf("health.add-nap failed: %v", err)
	}
	t.Log("Added nap")

	params := map[string]string{"date": date, "startMillis": strconv.FormatInt(start.UnixMilli(), 10)}
	RegisterCleanup(t, func() {
		if _, err := client.Call(context.Background(), "health.delete-nap", params, nil); err != nil && !isNotFound(err) {
			t.Logf("WARNING: failed to clean up nap: %v", err)
		}
	})

	if !hasNapAt(readNaps(t, client, date), start) {
		t.Error("nap list does not hold the added nap")
	}

	time.Sleep(2 * time.Second)
	if _, err := client.Call(ctx, "health.delete-nap", params, nil); err != nil {
		t.Fatalf("health.delete-nap failed: %v", err)
	}
	t.Log("Deleted nap")
	if hasNapAt(readNaps(t, client, date), start) {
		t.Error("nap list still holds the nap after the delete")
	}
}

type nap struct {
	Start   string `json:"napStartTimestampGMT"`
	TimeSec int    `json:"napTimeSec"`
}

func (n nap) start() (time.Time, bool) {
	t, err := time.ParseInLocation("2006-01-02T15:04:05", n.Start, time.UTC)
	return t, err == nil
}

func readNaps(t *testing.T, client *garminapi.Client, date string) []nap {
	t.Helper()
	data, err := client.Call(context.Background(), "health.sleep-naps", map[string]string{"date": date}, nil)
	if err != nil {
		t.Fatalf("health.sleep-naps failed: %v", err)
	}
	var naps []nap
	if err := json.Unmarshal(data, &naps); err != nil {
		t.Fatalf("unmarshal naps: %v", err)
	}
	return naps
}

func hasNapAt(naps []nap, start time.Time) bool {
	for _, n := range naps {
		if s, ok := n.start(); ok && s.Equal(start) {
			return true
		}
	}
	return false
}

// cleanupOrphanNaps deletes the naps of date that have the length of a test
// nap, left over from prior failed runs.
func cleanupOrphanNaps(t *testing.T, client *garminapi.Client, date string) {
	t.Helper()
	for _, n := range readNaps(t, client, date) {
		s, ok := n.start()
		if !ok || n.TimeSec != int(e2eNapLength.Seconds()) {
			continue
		}
		params := map[string]string{"date": date, "startMillis": strconv.FormatInt(s.UnixMilli(), 10)}
		if _, err := client.Call(context.Background(), "health.delete-nap", params, nil); err != nil && !isNotFound(err) {
			t.Logf("WARNING: failed to delete orphaned nap: %v", err)
		} else {
			t.Log("Cleaned up orphaned nap")
		}
	}
}
