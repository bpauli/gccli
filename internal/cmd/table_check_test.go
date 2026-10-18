package cmd

import (
	"slices"
	"strings"
	"testing"
)

func fixtureLeaf(t *testing.T, path string) Leaf {
	t.Helper()
	var found *Leaf
	walkLeaves(fixtureTree(), "", func(p string, l Leaf) {
		if p == path {
			found = &l
		}
	})
	if found == nil {
		t.Fatalf("fixture tree has no leaf %q", path)
	}
	return *found
}

func leafAt(path string, l Leaf) []Node {
	words := strings.Fields(path)
	l.Name = words[len(words)-1]
	node := Node(l)
	for i := len(words) - 2; i >= 0; i-- {
		node = Group{Name: words[i], Children: []Node{node}}
	}
	return []Node{node}
}

func TestCheckTableAcceptsFixture(t *testing.T) {
	if errs := checkTable(fixtureTree()); len(errs) > 0 {
		t.Errorf("checkTable = %v", errs)
	}
}

func TestCheckTableRejects(t *testing.T) {
	cases := []struct {
		name string
		path string
		edit func(l *Leaf)
		want string
	}{
		{
			"unknown endpoint", "goals list",
			func(l *Leaf) { l.Endpoint = "goals.lsit" },
			`goals list: endpoint "goals.lsit" is not in the endpoint table`,
		},
		{
			"In names no field", "goals list",
			func(l *Leaf) { l.Params = Params{"status": In("Stat")} },
			`goals list: Params["status"] binds In("Stat"), which is not a field of the args struct`,
		},
		{
			"In reads the outer field that shadows an embedded one", "activities list",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					Limit      string `help:"Cap." name:"cap"`
					Pagination `embed:"" set:"items=activities"`
					Type       string `help:"Filter by activity type." short:"t"`
				}{})
			},
			`activities list: placeholder {limit} of activities.list is required, but In("Limit") may be empty: make the field required, give it a default, or add .Or(...)`,
		},
		{
			"In names a field of a named embed", "activities list",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					Page Pagination `embed:"" set:"items=activities"`
					Type string     `help:"Filter by activity type." short:"t"`
				}{})
			},
			`activities list: Params["limit"] binds In("Limit"), which is not a field of the args struct`,
		},
		{
			"LateDates names a Date field", "health sleep",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					Date Date `arg:"" optional:""`
				}{})
			},
			`health sleep: LateDates names "Date", which is not a string field of the args struct`,
		},
		{
			"LateDates off the frozen list", "goals list",
			func(l *Leaf) { l.LateDates = []string{"Status"} },
			`goals list: LateDates is allowed only on the paths in v1LateDates`,
		},
		{
			"JSONNull off the frozen list", "goals list",
			func(l *Leaf) { l.Output = JSONNull{} },
			`goals list: JSONNull is allowed only on the paths in v1NullJSON`,
		},
		{
			"stale v1LateDates entry", "health sleep",
			func(l *Leaf) {
				l.LateDates = nil
				l.Spec = argsOf(&struct {
					Date Date `arg:"" optional:"" default:"today"`
				}{})
			},
			`health sleep: v1LateDates names this path, but the row has no LateDates: remove the entry`,
		},
		{
			"stale v1NullJSON entry", "courses detail",
			func(l *Leaf) {
				*l = Leaf{
					Spec: argsOf(&struct {
						ID string `arg:""`
					}{}),
					Endpoint: "courses.get", Params: Params{"courseID": In("ID")}, Output: JSON{},
				}
			},
			`courses detail: v1NullJSON names this path, but the row's Output is not JSONNull: remove the entry`,
		},
		{
			"Params key is no placeholder", "goals list",
			func(l *Leaf) { l.Params = Params{"state": In("Status")} },
			`goals list: Params["state"] is not a placeholder of goals.list`,
		},
		{
			"required placeholder unbound", "goals list",
			func(l *Leaf) { l.Params = nil },
			`goals list: placeholder {status} of goals.list is required, but Params does not bind it`,
		},
		{
			"required placeholder may be empty", "devices solar",
			func(l *Leaf) { l.Params["startDate"] = In("StartDate") },
			`devices solar: placeholder {startDate} of devices.solar is required, but In("StartDate") may be empty: make the field required, give it a default, or add .Or(...)`,
		},
		{
			"Write on a GET", "goals list",
			func(l *Leaf) { l.Output = Write{Done: "Listed goals"} },
			`goals list: Output is Write, but goals.list is a GET`,
		},
		{
			"read output on a write", "courses delete",
			func(l *Leaf) { l.Output = JSON{} },
			`courses delete: courses.delete is a DELETE: Output must be Write`,
		},
		{
			"Write without Done", "courses delete",
			func(l *Leaf) { l.Output = Write{Confirm: "Delete course {ID}?"} },
			`courses delete: Write has no Done`,
		},
		{
			"DELETE without Confirm", "courses delete",
			func(l *Leaf) { l.Output = Write{Done: "Deleted course {ID}"} },
			`courses delete: courses.delete is a DELETE: Write needs Confirm`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var l Leaf
			if tc.name != "stale v1NullJSON entry" {
				l = fixtureLeaf(t, tc.path)
			}
			tc.edit(&l)
			errs := checkTable(leafAt(tc.path, l))
			if !slices.ContainsFunc(errs, func(err error) bool { return err.Error() == tc.want }) {
				t.Errorf("checkTable = %v\nwant %s", errs, tc.want)
			}
		})
	}
}
