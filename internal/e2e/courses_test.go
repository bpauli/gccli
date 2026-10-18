//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
)

// TestGetCourses verifies that GetCourses returns a valid JSON object
// with a coursesForUser field.
func TestGetCourses(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	data, err := client.Call(ctx, "courses.list", nil, nil)
	if err != nil {
		t.Fatalf("GetCourses failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("expected JSON object from GetCourses, got: %v", err)
	}

	courses, ok := result["coursesForUser"]
	if !ok {
		t.Fatal("GetCourses response missing coursesForUser field")
	}

	arr, ok := courses.([]any)
	if !ok {
		t.Fatalf("coursesForUser is not an array, got %T", courses)
	}

	t.Logf("GetCourses returned %d courses", len(arr))
}

// TestGetCourseFavorites verifies that GetCourseFavorites returns a valid JSON array.
func TestGetCourseFavorites(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	data, err := client.Call(ctx, "courses.favorites", nil, nil)
	if err != nil {
		t.Fatalf("GetCourseFavorites failed: %v", err)
	}

	var favorites []json.RawMessage
	if err := json.Unmarshal(data, &favorites); err != nil {
		t.Fatalf("expected JSON array from GetCourseFavorites, got: %v", err)
	}

	t.Logf("GetCourseFavorites returned %d courses", len(favorites))
}

// TestGetCourseDetail fetches the first course from the list and verifies
// that GetCourse returns a response containing courseId and courseName fields.
func TestGetCourseDetail(t *testing.T) {
	client := AuthenticatedClient(t)
	ctx := context.Background()

	// Fetch courses to get an ID.
	data, err := client.Call(ctx, "courses.list", nil, nil)
	if err != nil {
		t.Fatalf("GetCourses failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal courses: %v", err)
	}

	arr, ok := result["coursesForUser"].([]any)
	if !ok || len(arr) == 0 {
		t.Skip("no courses in account, skipping detail test")
	}

	first, ok := arr[0].(map[string]any)
	if !ok {
		t.Fatal("first course is not a JSON object")
	}

	courseID := formatID(first["courseId"])
	if courseID == "" || courseID == "0" {
		t.Fatalf("expected valid courseId, got %q", courseID)
	}

	// Fetch the course detail.
	detail, err := client.Call(ctx, "courses.get", map[string]string{"courseID": courseID}, nil)
	if err != nil {
		t.Fatalf("GetCourse(%s) failed: %v", courseID, err)
	}

	var detailMap map[string]any
	if err := json.Unmarshal(detail, &detailMap); err != nil {
		t.Fatalf("unmarshal course detail: %v", err)
	}

	if _, ok := detailMap["courseId"]; !ok {
		t.Error("course detail response missing courseId field")
	}
	if _, ok := detailMap["courseName"]; !ok {
		t.Error("course detail response missing courseName field")
	}

	t.Logf("GetCourse(%s) returned a course with a name", courseID)
}
