//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

// e2eMenstrualLogDate is a past day before the test account set up cycle
// tracking, so it holds no log of the owner.
const e2eMenstrualLogDate = "2025-03-11"

// TestSetDeleteMenstrualLog sets a menstrual daily log, changes one field and
// checks that the other fields stay, then deletes the log.
func TestSetDeleteMenstrualLog(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()
	date := e2eMenstrualLogDate
	notes := e2eNotePrefix + "MENSTRUAL_LOG"

	if log, ok := readMenstrualLog(t, client, date); ok && !strings.HasPrefix(log.Notes, e2eNotePrefix) {
		t.Skipf("%s holds a menstrual log that the test did not write", date)
	}
	params := map[string]string{"date": date}
	RegisterCleanup(t, func() {
		if _, err := client.Call(context.Background(), "wellness.delete-menstrual-log", params, nil); err != nil {
			t.Logf("WARNING: failed to clean up menstrual log: %v", err)
		}
	})

	flow := "LIGHT"
	symptoms := []string{"CRAMPS", "HEADACHE"}
	if _, err := client.SetMenstrualLog(ctx, date, garminapi.MenstrualLogEdit{Flow: &flow, Symptoms: &symptoms, Notes: &notes}, time.Now()); err != nil {
		t.Fatalf("SetMenstrualLog failed: %v", err)
	}
	t.Log("Set menstrual log")
	log, ok := readMenstrualLog(t, client, date)
	if !ok || log.Flow != flow || !slices.Equal(log.Symptoms, symptoms) || log.Notes != notes {
		t.Fatalf("menstrual log GET does not match the set log: %+v", log)
	}

	time.Sleep(2 * time.Second)
	moods := []string{"HAPPY"}
	if _, err := client.SetMenstrualLog(ctx, date, garminapi.MenstrualLogEdit{Moods: &moods}, time.Now()); err != nil {
		t.Fatalf("SetMenstrualLog of the moods failed: %v", err)
	}
	t.Log("Set moods")
	log, ok = readMenstrualLog(t, client, date)
	if !ok || !slices.Equal(log.Moods, moods) {
		t.Errorf("menstrual log does not hold the new moods: %+v", log)
	}
	if log.Flow != flow || !slices.Equal(log.Symptoms, symptoms) || log.Notes != notes {
		t.Errorf("the change of the moods lost other fields: %+v", log)
	}

	if _, err := client.Call(ctx, "wellness.delete-menstrual-log", params, nil); err != nil {
		t.Fatalf("wellness.delete-menstrual-log failed: %v", err)
	}
	t.Log("Deleted menstrual log")
	if _, ok := readMenstrualLog(t, client, date); ok {
		t.Error("menstrual log still exists after the delete")
	}
}

type menstrualLog struct {
	Flow     string   `json:"flow"`
	Symptoms []string `json:"symptoms"`
	Moods    []string `json:"moods"`
	Notes    string   `json:"notes"`
}

// readMenstrualLog returns the log of date, and false when Garmin answers
// 204 because the day has no log.
func readMenstrualLog(t *testing.T, client *garminapi.Client, date string) (menstrualLog, bool) {
	t.Helper()
	data, err := client.Call(context.Background(), "wellness.menstrual-log", map[string]string{"date": date}, nil)
	if err != nil {
		t.Fatalf("wellness.menstrual-log failed: %v", err)
	}
	var log menstrualLog
	if data == nil {
		return log, false
	}
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatalf("unmarshal menstrual log: %v", err)
	}
	return log, true
}
