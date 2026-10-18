package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/alecthomas/kong"

	"github.com/bpauli/gccli/internal/units"
)

// fixtureTree is a small tree with one row of every shape the migration
// uses. handTree holds the same commands as the hand-written structs they
// copy, so a diff between the two is a defect in attach.
func fixtureTree() []Node {
	return []Node{
		Group{Name: "goals", Help: "View goals.", Children: []Node{
			Leaf{
				Name: "list", Help: "List goals.", Default: true,
				Spec: argsOf(&struct {
					Status string `help:"Goal status: active, future, past." default:"active" enum:"active,future,past"`
				}{}),
				Endpoint: "goals.list",
				Params:   Params{"status": In("Status")},
				Verb:     "get goals", Output: JSON{},
			},
		}},
		Group{Name: "health", Help: "View health data.", Children: []Node{
			Leaf{
				Name: "sleep", Help: "Show sleep data.",
				Spec: argsOf(&struct {
					Date string `arg:"" optional:"" help:"Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today."`
				}{}),
				Endpoint: "health.sleep",
				Params:   Params{"displayName": DisplayName, "date": In("Date")},
				Verb:     "get sleep", Output: JSON{},
				LateDates: []string{"Date"},
			},
		}},
		Group{Name: "devices", Help: "Manage devices.", Children: []Node{
			Leaf{
				Name: "solar", Help: "Show solar charging data for a device.",
				Spec: argsOf(&struct {
					DeviceID  string `arg:"" help:"Device ID."`
					StartDate Date   `help:"Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today." name:"start"`
					EndDate   Date   `help:"End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to start date." name:"end"`
				}{}),
				Endpoint: "devices.solar",
				Params: Params{
					"deviceID":  In("DeviceID"),
					"startDate": In("StartDate").Or(Today),
					"endDate":   In("EndDate").Or(In("StartDate")).Or(Today),
				},
				Verb: "get solar data", Output: JSON{},
			},
		}},
		Group{Name: "activities", Help: "List and search activities.", Children: []Node{
			Leaf{
				Name: "list", Help: "List recent activities.", Default: true,
				Spec: argsOf(&struct {
					Pagination `embed:"" set:"items=activities"`
					Type       string `help:"Filter by activity type (e.g. running, cycling)." short:"t"`
				}{}),
				Endpoint: "activities.list",
				Params:   Params{"start": In("Offset"), "limit": In("Limit"), "activityType": In("Type")},
				Verb:     "list activities",
				Output: Rows{Noun: "activities", Columns: []Column{
					{Label: "ID", Path: "activityId"},
					{Label: "NAME", Path: "activityName"},
				}},
			},
		}},
		Group{Name: "courses", Help: "Manage courses.", Children: []Node{
			Leaf{
				Name: "delete", Help: "Delete a course.",
				Spec: argsOf(&struct {
					ID string `arg:"" help:"Course ID."`
				}{}),
				Endpoint: "courses.delete",
				Params:   Params{"courseID": In("ID")},
				Verb:     "delete course",
				Output:   Write{Done: "Deleted course {ID}", Confirm: "Delete course {ID}?"},
			},
		}},
		Custom{Name: "completion", Help: "Generate shell completion script.", Cmd: &CompletionCmd{}},
	}
}

type tableCLI struct {
	RootFlags

	Version kong.VersionFlag `help:"Print version information." name:"version"`

	kong.Plugins
}

type handCLI struct {
	RootFlags

	Version kong.VersionFlag `help:"Print version information." name:"version"`

	Goals struct {
		List GoalsListCmd `cmd:"" default:"withargs" help:"List goals."`
	} `cmd:"" help:"View goals."`
	Health struct {
		Sleep HealthSleepCmd `cmd:"" help:"Show sleep data."`
	} `cmd:"" help:"View health data."`
	Devices struct {
		Solar DeviceSolarCmd `cmd:"" help:"Show solar charging data for a device."`
	} `cmd:"" help:"Manage devices."`
	Activities struct {
		List ActivitiesListCmd `cmd:"" default:"withargs" help:"List recent activities."`
	} `cmd:"" help:"List and search activities."`
	Courses struct {
		Delete CourseDeleteCmd `cmd:"" help:"Delete a course."`
	} `cmd:"" help:"Manage courses."`
	Completion CompletionCmd `cmd:"" help:"Generate shell completion script."`
}

// newParserFor builds a parser with the options execute uses, so help is
// printed as the binary prints it.
func newParserFor(t *testing.T, cli any, stdout, stderr io.Writer) *kong.Kong {
	t.Helper()
	parser, err := kong.New(cli,
		kong.Name("gccli"),
		kong.Description("Garmin Connect CLI"),
		kong.UsageOnError(),
		kong.Writers(stdout, stderr),
		kong.Exit(func(int) {}),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
		kong.Help(colorHelpPrinter),
		kong.Vars{"version": "test", "units": strings.Join(units.PreferenceNames(), ",")},
	)
	if err != nil {
		t.Fatalf("kong.New: %v", err)
	}
	return parser
}

func newTableParser(t *testing.T, nodes []Node, stdout, stderr io.Writer) *kong.Kong {
	t.Helper()
	return newParserFor(t, &tableCLI{Plugins: attach(nodes)}, stdout, stderr)
}

// dumpModel writes the Kong model fields that docs_gen.go, completion*.go
// and the golden harness read.
func dumpModel(w io.Writer, n *kong.Node) {
	for _, c := range n.Children {
		def := ""
		if n.DefaultCmd == c {
			def = " default=" + c.Tag.Default
		}
		_, _ = fmt.Fprintf(w, "%s | help=%q hidden=%v type=%v leaf=%v%s\n", c.Path(), c.Help, c.Hidden, c.Type, c.Leaf(), def)
		for _, p := range c.Positional {
			_, _ = fmt.Fprintf(w, "  arg %q help=%q required=%v enum=%q summary=%q\n", p.Name, p.Help, p.Required, p.Enum, p.Summary())
		}
		for _, f := range c.Flags {
			_, _ = fmt.Fprintf(w, "  flag %q short=%q help=%q enum=%q default=%q hasdefault=%v required=%v bool=%v placeholder=%q hidden=%v summary=%q string=%q\n",
				f.Name, string(f.Short), f.Help, f.Enum, f.Default, f.HasDefault, f.Required, f.IsBool(), f.PlaceHolder, f.Hidden, f.Summary(), f.String())
		}
		dumpModel(w, c)
	}
}

func TestAttachMatchesHandStructs(t *testing.T) {
	var hand, table bytes.Buffer
	dumpModel(&hand, newParserFor(t, &handCLI{}, io.Discard, io.Discard).Model.Node)
	dumpModel(&table, newTableParser(t, fixtureTree(), io.Discard, io.Discard).Model.Node)
	if hand.String() != table.String() {
		t.Errorf("Kong model differs\nhand:\n%s\ntable:\n%s", hand.String(), table.String())
	}
	if !strings.Contains(table.String(), "goals list | ") || !strings.Contains(table.String(), "completion | ") {
		t.Fatalf("fixture tree is missing commands:\n%s", table.String())
	}

	paths := [][]string{{}}
	for _, c := range collectCommands(newParserFor(t, &handCLI{}, io.Discard, io.Discard).Model.Node, nil) {
		paths = append(paths, c.Path)
	}
	for _, path := range paths {
		args := append(slices.Clone(path), "--help")
		var handOut, tableOut bytes.Buffer
		_, _ = newParserFor(t, &handCLI{}, &handOut, &handOut).Parse(args)
		_, _ = newTableParser(t, fixtureTree(), &tableOut, &tableOut).Parse(args)
		if handOut.Len() == 0 {
			t.Errorf("gccli %s printed nothing", strings.Join(args, " "))
		}
		if handOut.String() != tableOut.String() {
			t.Errorf("gccli %s differs\nhand:\n%s\ntable:\n%s", strings.Join(args, " "), handOut.String(), tableOut.String())
		}
	}
}

// hookArgs has every Kong hook and an embedded type with its own Validate.
// Kong must find each of them on the table leaf as it does on a hand struct.
type hookArgs struct {
	HookPages `embed:"" set:"items=hooks"`
	Name      string `arg:"" help:"Name."`
}

type HookPages struct {
	Limit int `help:"Maximum number of ${items} to return." default:"20"`
}

var hookLog []string

func (h *hookArgs) BeforeApply() error {
	hookLog = append(hookLog, "BeforeApply")
	return nil
}

func (h *hookArgs) AfterApply(ctx *kong.Context) error {
	hookLog = append(hookLog, "AfterApply "+ctx.Command())
	return nil
}

func (h *hookArgs) Validate() error {
	hookLog = append(hookLog, "Validate")
	if h.Name == "bad" {
		return errors.New("name must not be bad")
	}
	return nil
}

func (p *HookPages) Validate() error {
	hookLog = append(hookLog, "HookPages.Validate")
	if p.Limit < 1 {
		return fmt.Errorf("--limit must be at least 1, got %d", p.Limit)
	}
	return nil
}

func TestAttachRunsHooksNatively(t *testing.T) {
	type handHooks struct {
		Hooks struct {
			Demo hookArgs `cmd:"" help:"Hook demo."`
		} `cmd:"" help:"Hooks."`
	}
	tree := []Node{Group{Name: "hooks", Help: "Hooks.", Children: []Node{
		Leaf{Name: "demo", Help: "Hook demo.", Spec: argsOf(&hookArgs{}), Endpoint: "goals.list", Output: JSON{}},
	}}}
	cases := []struct {
		args []string
		log  []string
		err  string
	}{
		{[]string{"hooks", "demo", "x"}, []string{"BeforeApply", "Validate", "HookPages.Validate", "AfterApply hooks demo <name>"}, ""},
		{[]string{"hooks", "demo", "bad"}, []string{"BeforeApply", "Validate"}, "hooks demo: name must not be bad"},
		{[]string{"hooks", "demo", "x", "--limit", "0"}, []string{"BeforeApply", "Validate", "HookPages.Validate"}, "hooks demo: --limit must be at least 1, got 0"},
	}
	for _, tc := range cases {
		for _, side := range []struct {
			name string
			cli  any
		}{{"hand", &handHooks{}}, {"table", &tableCLI{Plugins: attach(tree)}}} {
			hookLog = nil
			_, err := newParserFor(t, side.cli, io.Discard, io.Discard).Parse(tc.args)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tc.err {
				t.Errorf("%s %v: err = %q, want %q", side.name, tc.args, got, tc.err)
			}
			if !slices.Equal(hookLog, tc.log) {
				t.Errorf("%s %v: hooks = %q, want %q", side.name, tc.args, hookLog, tc.log)
			}
		}
	}
}

type sleepArgs struct {
	Date string `arg:"" optional:"" help:"Date."`
}

func parsedDate(t *testing.T, ctx *kong.Context) string {
	t.Helper()
	leaf, ok := ctx.Selected().Target.Interface().(specLeafG[sleepArgs])
	if !ok {
		t.Fatalf("target is %T", ctx.Selected().Target.Interface())
	}
	return leaf.Args.Date
}

func TestAttachIsolatesLeafValues(t *testing.T) {
	shared := &sleepArgs{}
	spec := argsOf(shared)
	tree := []Node{Group{Name: "health", Help: "Health.", Children: []Node{
		Leaf{Name: "sleep", Help: "Sleep.", Spec: spec, Endpoint: "health.sleep", Output: JSON{}},
		Leaf{Name: "sleep2", Help: "Sleep again.", Spec: spec, Endpoint: "health.sleep", Output: JSON{}},
	}}}

	parser := newTableParser(t, tree, io.Discard, io.Discard)
	ctx, err := parser.Parse([]string{"health", "sleep", "2026-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	if got := parsedDate(t, ctx); got != "2026-01-01" {
		t.Errorf("sleep: Date = %q", got)
	}
	ctx, err = parser.Parse([]string{"health", "sleep2"})
	if err != nil {
		t.Fatal(err)
	}
	if got := parsedDate(t, ctx); got != "" {
		t.Errorf("sleep2: Date = %q, want empty", got)
	}
	if shared.Date != "" {
		t.Errorf("table literal: Date = %q, want empty", shared.Date)
	}

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			want := fmt.Sprintf("2026-03-%02d", i%28+1)
			ctx, err := newTableParser(t, tree, io.Discard, io.Discard).Parse([]string{"health", "sleep", want})
			if err != nil {
				t.Error(err)
				return
			}
			if got := parsedDate(t, ctx); got != want {
				t.Errorf("parser %d: Date = %q, want %q", i, got, want)
			}
		}()
	}
	wg.Wait()
	if shared.Date != "" {
		t.Errorf("table literal after 50 parsers: Date = %q, want empty", shared.Date)
	}
}
