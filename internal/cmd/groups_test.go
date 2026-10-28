package cmd

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/garminapi"
)

func TestGroupsCreateImageEnumMatchesTheImageList(t *testing.T) {
	f, ok := reflect.TypeFor[GroupsCreateCmd]().FieldByName("Image")
	if !ok {
		t.Fatal("GroupsCreateCmd has no Image field")
	}
	got := strings.Split(f.Tag.Get("enum"), ",")
	if !slices.Equal(got, garminapi.GroupImages) {
		t.Errorf("--image enum = %v, want %v", got, garminapi.GroupImages)
	}
}
