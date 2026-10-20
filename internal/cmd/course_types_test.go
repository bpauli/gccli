package cmd

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestCourseActivityTypesMatchEnum pins that the enum of courses import
// --type, which Kong checks at parse time, names exactly the keys of
// courseActivityTypes, so no value can reach Run without a PK.
func TestCourseActivityTypesMatchEnum(t *testing.T) {
	f, ok := reflect.TypeFor[CourseImportCmd]().FieldByName("Type")
	if !ok {
		t.Fatal("CourseImportCmd has no Type field")
	}
	got := strings.Split(f.Tag.Get("enum"), ",")
	want := slices.Sorted(maps.Keys(courseActivityTypes))
	if !slices.Equal(got, want) {
		t.Errorf("enum of --type = %v, want the sorted keys of courseActivityTypes %v", got, want)
	}
	if courseActivityTypes["cycling"] != 2 || courseActivityTypes["gravel_cycling"] != 143 {
		t.Errorf("cycling = %d and gravel_cycling = %d, want 2 and 143", courseActivityTypes["cycling"], courseActivityTypes["gravel_cycling"])
	}
}
