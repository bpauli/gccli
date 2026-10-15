//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/garminapi"
)

const e2eCoursePrefix = "E2E_TEST_"

// TestImportAndDeleteCourse imports a GPX file as a new course, verifies it
// appears in the listing, and deletes it.
func TestImportAndDeleteCourse(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	// Safety-net: clean up orphaned E2E courses from prior runs.
	cleanupOrphanCourses(t, client)

	gpxPath := stubPath(t, "route.gpx")
	courseName := fmt.Sprintf("%sCOURSE_%d", e2eCoursePrefix, time.Now().UnixNano())

	// Step 1: Import GPX.
	parsed, err := client.ImportCourseGPX(ctx, gpxPath)
	if err != nil {
		t.Fatalf("ImportCourseGPX failed: %v", err)
	}

	var course map[string]any
	if err := json.Unmarshal(parsed, &course); err != nil {
		t.Fatalf("unmarshal imported course: %v", err)
	}

	geoPoints, ok := course["geoPoints"].([]any)
	if !ok || len(geoPoints) == 0 {
		t.Fatal("imported course has no geo points")
	}
	t.Logf("Imported GPX with %d geo points", len(geoPoints))

	// Step 2: Enrich elevation.
	elevInput := make([][]any, 0, len(geoPoints))
	for _, p := range geoPoints {
		pt, ok := p.(map[string]any)
		if !ok {
			continue
		}
		lat := pt["latitude"]
		lon := pt["longitude"]
		if lat == nil || lon == nil {
			// Fall back to short keys used by some responses.
			lat = pt["lat"]
			lon = pt["lon"]
		}
		if lat == nil || lon == nil {
			continue
		}
		elevInput = append(elevInput, []any{lat, lon, nil})
	}

	elevJSON, err := json.Marshal(elevInput)
	if err != nil {
		t.Fatalf("marshal elevation input: %v", err)
	}

	elevData, err := client.Call(ctx, "courses.elevation", nil, elevJSON)
	if err != nil {
		t.Fatalf("GetCourseElevation failed: %v", err)
	}

	var elevPoints [][]any
	if err := json.Unmarshal(elevData, &elevPoints); err != nil {
		t.Fatalf("unmarshal elevation: %v", err)
	}

	for i, ep := range elevPoints {
		if i >= len(geoPoints) {
			break
		}
		if pt, ok := geoPoints[i].(map[string]any); ok && len(ep) >= 3 {
			pt["elevation"] = ep[2]
		}
	}
	course["geoPoints"] = geoPoints
	t.Logf("Enriched %d points with elevation data", len(elevPoints))

	// Set E2E test name, activity type (cycling = 2), and defaults.
	course["courseName"] = courseName
	course["activityTypePk"] = 2
	if course["coordinateSystem"] == nil {
		course["coordinateSystem"] = "WGS84"
	}
	if course["sourceTypeId"] == nil {
		course["sourceTypeId"] = 3
	}
	if course["startPoint"] == nil && len(geoPoints) > 0 {
		course["startPoint"] = geoPoints[0]
	}

	// Filter to only the fields accepted by the save endpoint.
	save := filterSaveFields(course)
	save["coursePrivacy"] = 2
	if save["rulePK"] == nil {
		save["rulePK"] = 2
	}

	payload, err := json.Marshal(save)
	if err != nil {
		t.Fatalf("marshal course: %v", err)
	}

	// Step 3: Save course.
	saved, err := client.Call(ctx, "courses.save", nil, payload)
	if err != nil {
		t.Fatalf("SaveCourse failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(saved, &result); err != nil {
		t.Fatalf("unmarshal saved course: %v", err)
	}

	courseID := formatID(result["courseId"])
	if courseID == "" || courseID == "0" {
		t.Fatalf("expected valid courseId, got %q", courseID)
	}
	t.Logf("Created course %s with name %q", courseID, courseName)

	// Register cleanup to delete the course — runs even on test failure.
	RegisterCleanup(t, func() {
		if _, delErr := client.Call(context.Background(), "courses.delete", map[string]string{"courseID": courseID}, nil); delErr != nil {
			t.Logf("WARNING: failed to clean up course %s: %v", courseID, delErr)
		} else {
			t.Logf("Cleaned up course %s", courseID)
		}
	})

	// Brief pause to avoid rate limiting on subsequent calls.
	time.Sleep(2 * time.Second)

	// Verify the course appears in the listing.
	found := findCourseByID(t, client, courseID)
	if !found {
		t.Errorf("created course %s not found in GetCourses listing", courseID)
	}

	// Verify detail endpoint returns the course.
	detail, err := client.Call(ctx, "courses.get", map[string]string{"courseID": courseID}, nil)
	if err != nil {
		t.Fatalf("GetCourse(%s) failed: %v", courseID, err)
	}

	var detailMap map[string]any
	if err := json.Unmarshal(detail, &detailMap); err != nil {
		t.Fatalf("unmarshal course detail: %v", err)
	}
	if name, _ := detailMap["courseName"].(string); name != courseName {
		t.Errorf("course name = %q, want %q", name, courseName)
	}

	// Brief pause to avoid rate limiting on delete.
	time.Sleep(2 * time.Second)

	// Delete the course explicitly (cleanup is the safety net).
	if _, err := client.Call(ctx, "courses.delete", map[string]string{"courseID": courseID}, nil); err != nil {
		t.Fatalf("DeleteCourse(%s) failed: %v", courseID, err)
	}
	t.Logf("Deleted course %s", courseID)

	// Verify it no longer appears in the listing.
	if findCourseByID(t, client, courseID) {
		t.Errorf("course %s still appears in listing after delete", courseID)
	}
}

// findCourseByID searches courses for one with the given ID.
func findCourseByID(t *testing.T, client *garminapi.Client, courseID string) bool {
	t.Helper()

	data, err := client.Call(context.Background(), "courses.list", nil, nil)
	if err != nil {
		t.Fatalf("GetCourses failed while searching for course %s: %v", courseID, err)
	}

	var wrapper map[string]any
	if err := json.Unmarshal(data, &wrapper); err != nil {
		t.Fatalf("unmarshal courses: %v", err)
	}

	arr, ok := wrapper["coursesForUser"].([]any)
	if !ok {
		return false
	}

	for _, item := range arr {
		if c, ok := item.(map[string]any); ok {
			if formatID(c["courseId"]) == courseID {
				return true
			}
		}
	}
	return false
}

// cleanupOrphanCourses deletes any courses with the E2E_TEST_ prefix
// left over from prior failed runs.
func cleanupOrphanCourses(t *testing.T, client *garminapi.Client) {
	t.Helper()
	ctx := context.Background()

	data, err := client.Call(ctx, "courses.list", nil, nil)
	if err != nil {
		t.Logf("WARNING: could not list courses for orphan cleanup: %v", err)
		return
	}

	var wrapper map[string]any
	if err := json.Unmarshal(data, &wrapper); err != nil {
		t.Logf("WARNING: could not parse courses for orphan cleanup: %v", err)
		return
	}

	arr, ok := wrapper["coursesForUser"].([]any)
	if !ok {
		return
	}

	for _, item := range arr {
		c, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := c["courseName"].(string)
		if strings.HasPrefix(name, e2eCoursePrefix) {
			id := formatID(c["courseId"])
			if _, delErr := client.Call(ctx, "courses.delete", map[string]string{"courseID": id}, nil); delErr != nil {
				t.Logf("WARNING: failed to delete orphaned course %s (%s): %v", id, name, delErr)
			} else {
				t.Logf("Cleaned up orphaned course %s (%s)", id, name)
			}
		}
	}
}

// courseSaveFields are the fields accepted by the course save endpoint.
var courseSaveFields = []string{
	"activityTypePk", "boundingBox", "coordinateSystem", "courseLines",
	"courseName", "coursePoints", "distanceMeter", "elapsedSeconds",
	"elevationGainMeter", "elevationLossMeter", "favorite", "geoPoints",
	"hasPaceBand", "hasPowerGuide", "hasTurnDetectionDisabled", "includeLaps",
	"matchedToSegments", "openStreetMap", "rulePK", "sourceTypeId",
	"speedMeterPerSecond", "startPoint", "userProfilePk",
}

// filterSaveFields returns a new map containing only the non-nil fields
// accepted by the course save endpoint.
func filterSaveFields(course map[string]any) map[string]any {
	save := make(map[string]any, len(courseSaveFields))
	for _, key := range courseSaveFields {
		if v, ok := course[key]; ok && v != nil {
			save[key] = v
		}
	}
	return save
}

// stubPath returns the absolute path to a file in the stubs directory.
func stubPath(t *testing.T, name string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(filename), "stubs", name)
}
