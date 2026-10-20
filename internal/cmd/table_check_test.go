package cmd

import (
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/garminapi"
)

// fixtureLeaf returns the fixture leaf at path, or a zero Leaf for a path
// outside the fixture tree.
func fixtureLeaf(path string) Leaf {
	var found Leaf
	walkNodes(fixtureTree(), "", func(p string, n Node) {
		if l, ok := n.(Leaf); ok && p == path {
			found = l
		}
	})
	return found
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

func TestCommandTable(t *testing.T) {
	for _, err := range checkTable(commands) {
		t.Error(err)
	}
}

func TestEveryEndpointHasACaller(t *testing.T) {
	called := map[string]bool{}
	walkNodes(commands, "", func(_ string, n Node) {
		if l, ok := n.(Leaf); ok {
			called[l.Endpoint] = true
		}
	})
	fset := token.NewFileSet()
	for _, fn := range slices.Concat(parseFuncs(t, fset, "."), parseFuncs(t, fset, "../garminapi")) {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if name, ok := endpointName(call); ok {
					called[name] = true
				}
			}
			return true
		})
	}
	for _, name := range garminapi.Names() {
		if !called[name] {
			t.Errorf("row %s has no caller in internal/cmd or internal/garminapi: delete it or call it", name)
		}
	}
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
			"In names an unexported field", "goals list",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					status string `help:"Goal status." default:"active"`
				}{})
				l.Params = Params{"status": In("status")}
			},
			`goals list: Params["status"] binds In("status"), which is not a field of the args struct`,
		},
		{
			"In names no field at all", "goals list",
			func(l *Leaf) { l.Params = Params{"status": In("")} },
			`goals list: Params["status"] binds In(""), which names no field`,
		},
		{
			"In names a field Kong skips", "goals list",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					Status string `kong:"-" default:"active"`
				}{})
			},
			`goals list: Params["status"] binds In("Status"), which Kong skips (kong:"-")`,
		},
		{
			"In names a field the runner cannot send", "goals list",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					Status []string `help:"Goal status." default:"active"`
				}{})
			},
			`goals list: Params["status"] binds In("Status") of type []string: the runner sends only string, int, Date and ID`,
		},
		{
			"Kong enum allows a value the placeholder does not", "goals list",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					Status string `help:"Goal status." default:"active" enum:"active,future,past,done"`
				}{})
			},
			`goals list: Params["status"] binds In("Status"), whose enum allows "done", which {status} of goals.list does not: active, future, past`,
		},
		{
			"argsOf(nil)", "goals list",
			func(l *Leaf) { l.Spec = argsOf[struct{}](nil) },
			`goals list: argsOf[struct {}](nil): pass a pointer to a struct literal`,
		},
		{
			"argsOf of a non-struct", "goals list",
			func(l *Leaf) { l.Spec = argsOf(new(int)) },
			`goals list: argsOf(*int) is not a struct: pass a pointer to a struct literal`,
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
			"Rows has no Columns", "activities list",
			func(l *Leaf) { l.Output = Rows{Noun: "activities"} },
			`activities list: Rows has no Columns`,
		},
		{
			"a column has no Path", "activities list",
			func(l *Leaf) { l.Output = Rows{Noun: "activities", Columns: []Column{{Label: "ID"}}} },
			`activities list: Columns[0] (ID) has no Path`,
		},
		{
			"a column has an unknown Format", "activities list",
			func(l *Leaf) {
				l.Output = Rows{Noun: "activities", Columns: []Column{{Label: "ID", Path: "activityId", Format: Format(9)}}}
			},
			`activities list: Columns[0] (ID) has Format(9), which is not a Format`,
		},
		{
			"a column has the first Format past the last one", "activities list",
			func(l *Leaf) {
				l.Output = Rows{Noun: "activities", Columns: []Column{{Label: "ID", Path: "activityId", Format: Format(len(formatNames))}}}
			},
			fmt.Sprintf("activities list: Columns[0] (ID) has Format(%d), which is not a Format", len(formatNames)),
		},
		{
			"an int bound to a placeholder that may be left out", "activities list",
			func(l *Leaf) {
				l.Params = Params{"start": In("Offset"), "limit": In("Limit"), "activityType": In("Limit")}
			},
			`activities list: Params["activityType"] binds In("Limit") of type int, but {activityType} of activities.list may be left out: the runner sends an int even when it is 0, so use a string field`,
		},
		{
			"an int bound to a placeholder with a default", "goals list",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					Locale int `help:"Locale number." default:"1"`
				}{})
				l.Endpoint = "training.plans"
				l.Params = Params{"locale": In("Locale")}
			},
			`goals list: Params["locale"] binds In("Locale") of type int, but {locale} of training.plans may be left out: the runner sends an int even when it is 0, so use a string field`,
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
			"positional date of type string", "health sleep view",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					Date string `arg:"" optional:"" default:"today" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
				}{})
			},
			`health sleep view: Date is a date (by its name), so its type must be Date or DateTime, not string: Kong then checks it at parse time`,
		},
		{
			"date flag of type string", "body delete-weight",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					PK   string `arg:"" help:"Weigh-in version (primary key)."`
					Date string `help:"Date of the weigh-in (YYYY-MM-DD)." required:"" name:"date"`
				}{})
			},
			`body delete-weight: Date is a date (by its name), so its type must be Date or DateTime, not string: Kong then checks it at parse time`,
		},
		{
			"date and time flag of type string, help without a format", "body delete-weight",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					PK   string `arg:"" help:"Weigh-in version (primary key)."`
					Date string `help:"Start date and time (e.g. 2024-06-15T07:30:00). Defaults to now." name:"date"`
				}{})
			},
			`body delete-weight: Date is a date (by its name), so its type must be Date or DateTime, not string: Kong then checks it at parse time`,
		},
		{
			"date flag of type string whose name does not say date", "body delete-weight",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					PK  string `arg:"" help:"Weigh-in version (primary key)."`
					End string `help:"Last day (YYYY-MM-DD)." required:"" name:"end"`
				}{})
			},
			`body delete-weight: End is a date (its help names YYYY-MM-DD), so its type must be Date or DateTime, not string: Kong then checks it at parse time`,
		},
		{
			"JSONNull off the frozen list", "goals list",
			func(l *Leaf) { l.Output = JSONNull{} },
			`goals list: JSONNull is allowed only on the paths in v1NullJSON`,
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
		{
			"endpoint takes a body", "hydration log",
			func(l *Leaf) { *l = Leaf{Endpoint: "hydration.log", Output: Write{Done: "Logged hydration"}} },
			`hydration log: hydration.log takes a request body, which no Source gives: keep the command Custom`,
		},
		{
			"nil Output", "goals list",
			func(l *Leaf) { l.Output = nil },
			`goals list: Output is nil`,
		},
		{
			"Done names no field", "courses delete",
			func(l *Leaf) { l.Output = Write{Done: "Deleted course {Idd}", Confirm: "Delete course {ID}?"} },
			`courses delete: Done names {Idd}, which is not a field of the args struct`,
		},
		{
			"Confirm names no field", "courses delete",
			func(l *Leaf) { l.Output = Write{Done: "Deleted course {ID}", Confirm: "Delete course {Idd}?"} },
			`courses delete: Confirm names {Idd}, which is not a field of the args struct`,
		},
		{
			"Done names a field a template cannot take", "courses delete",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					ID   string `arg:"" help:"Course ID."`
					Deep bool   `help:"Deep."`
				}{})
				l.Output = Write{Done: "Deleted course {ID} {Deep}", Confirm: "Delete course {ID}?"}
			},
			`courses delete: Done names {Deep} of type bool: a template takes only string, int, Date and ID`,
		},
		{
			"Rows without Noun", "activities list",
			func(l *Leaf) { l.Output = Rows{Columns: []Column{{Label: "ID", Path: "activityId"}}} },
			`activities list: Rows has no Noun`,
		},
		{
			"required placeholder bound to an empty default", "goals list",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					Status string `help:"Goal status." default:""`
				}{})
			},
			`goals list: placeholder {status} of goals.list is required, but In("Status") may be empty: make the field required, give it a default, or add .Or(...)`,
		},
		{
			"required placeholder bound to a required field in an xor group", "goals list",
			func(l *Leaf) {
				l.Spec = argsOf(&struct {
					Status string `help:"Goal status." required:"" xor:"filter"`
					Other  string `help:"Other filter." required:"" xor:"filter"`
				}{})
			},
			`goals list: placeholder {status} of goals.list is required, but In("Status") may be empty: make the field required, give it a default, or add .Or(...)`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := fixtureLeaf(tc.path)
			tc.edit(&l)
			errs := checkTable(leafAt(tc.path, l))
			if !slices.ContainsFunc(errs, func(err error) bool { return err.Error() == tc.want }) {
				t.Errorf("checkTable = %v\nwant %s", errs, tc.want)
			}
		})
	}
}

func TestDateTypeError(t *testing.T) {
	cases := []struct {
		name, help string
		typ        reflect.Type
		want       string
	}{
		{"Date", "Start date and time. Defaults to now.", reflect.TypeFor[Date](), ""},
		{"Date", "Start date and time. Defaults to now.", reflect.TypeFor[DateTime](), ""},
		{"date", "Start date and time. Defaults to now.", reflect.TypeFor[string](), "date is a date (by its name), so its type must be Date or DateTime, not string: Kong then checks it at parse time"},
		{"StartDate", "Start.", reflect.TypeFor[string](), "StartDate is a date (by its name), so its type must be Date or DateTime, not string: Kong then checks it at parse time"},
		{"start-date", "Start.", reflect.TypeFor[string](), "start-date is a date (by its name), so its type must be Date or DateTime, not string: Kong then checks it at parse time"},
		{"End", "Last day (YYYY-MM-DD).", reflect.TypeFor[string](), "End is a date (its help names YYYY-MM-DD), so its type must be Date or DateTime, not string: Kong then checks it at parse time"},
		{"Date", "Start date and time: YYYY-MM-DDTHH:MM:SS or YYYY-MM-DD.", reflect.TypeFor[DateTime](), ""},
		{"Date", "Start date and time: YYYY-MM-DDTHH:MM:SS or YYYY-MM-DD.", reflect.TypeFor[string](), "Date is a date (by its name), so its type must be Date or DateTime, not string: Kong then checks it at parse time"},
		{"Time", "Start time (HH:MM).", reflect.TypeFor[Clock](), ""},
		{"time", "Start time (HH:MM).", reflect.TypeFor[string](), "time is a clock time (by its name), so its type must be Clock, not string: Kong then checks it at parse time"},
		{"StartTime", "Start.", reflect.TypeFor[string](), "StartTime is a clock time (by its name), so its type must be Clock, not string: Kong then checks it at parse time"},
		{"start-time", "Start.", reflect.TypeFor[string](), "start-time is a clock time (by its name), so its type must be Clock, not string: Kong then checks it at parse time"},
		{"Start", "Start of the nap (HH:MM).", reflect.TypeFor[string](), "Start is a clock time (its help names HH:MM), so its type must be Clock, not string: Kong then checks it at parse time"},
		{"Start", "Start of the nap (HH:MM).", reflect.TypeFor[Clock](), ""},
		{"Update", "Update the item.", reflect.TypeFor[bool](), ""},
		{"update", "Update the item.", reflect.TypeFor[bool](), ""},
	}
	for _, tc := range cases {
		if got := dateTypeError(tc.name, tc.help, tc.typ); got != tc.want {
			t.Errorf("dateTypeError(%q, %q, %s) = %q, want %q", tc.name, tc.help, tc.typ, got, tc.want)
		}
	}
}

func TestCheckTableRejectsBadCustom(t *testing.T) {
	cases := []struct {
		name string
		cmd  interface{ Run(*Globals) error }
		want string
	}{
		{"nil Cmd", nil, "completion: Custom needs a Cmd that points to a struct, got <nil>"},
		{"Cmd is not a struct", intCmd(0), "completion: Custom needs a Cmd that points to a struct, got cmd.intCmd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := checkTable([]Node{Custom{Name: "completion", Help: "Completion.", Cmd: tc.cmd}})
			if len(errs) != 1 || errs[0].Error() != tc.want {
				t.Errorf("checkTable = %v\nwant %s", errs, tc.want)
			}
		})
	}
}

type intCmd int

func (intCmd) Run(*Globals) error { return nil }

// The frozen lists must match the golden files that pin each quirk, so a
// new entry and a stale entry both fail here.
func TestFrozenListsMatchGoldens(t *testing.T) {
	t.Run("v1NullJSON", func(t *testing.T) {
		pinned := goldenLeaves(t, "@empty.golden", func(g golden) bool {
			return slices.IndexFunc(goldenModes, func(m goldenMode) bool {
				return strings.TrimSpace(g.sections[m.name+" stdout"]) != "null"
			}) < 0
		})
		compareFrozen(t, "v1NullJSON", v1NullJSON, pinned, "<leaf>@empty.golden prints null in every mode")
	})
}

// goldenLeaves returns the leaf path of every golden file with the suffix
// whose sections pass keep.
func goldenLeaves(t *testing.T, suffix string, keep func(golden) bool) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(goldenDir, "*"+suffix))
	if err != nil {
		t.Fatal(err)
	}
	var leaves []string
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		g, err := parseGolden(string(data))
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		if keep(g) {
			path, _, _ := goldenCommand(file)
			leaves = append(leaves, path)
		}
	}
	return leaves
}

func compareFrozen(t *testing.T, list string, entries, pinned []string, rule string) {
	t.Helper()
	for _, path := range entries {
		if !slices.Contains(pinned, path) {
			t.Errorf("%s names %q, but no %s: remove the entry", list, path, rule)
		}
	}
	for _, path := range pinned {
		if !slices.Contains(entries, path) {
			t.Errorf("%s does not name %q, but its %s: add the entry", list, path, rule)
		}
	}
}
