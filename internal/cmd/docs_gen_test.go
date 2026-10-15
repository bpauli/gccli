package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
)

type refTestCLI struct {
	Things refTestThingsCmd `cmd:"" help:"Manage things."`
}

type refTestThingsCmd struct {
	Verbose bool `help:"Print more."`

	List refTestListCmd `cmd:"" default:"withargs" help:"List things."`
	Get  refTestGetCmd  `cmd:"" help:"Show one thing."`
}

type refTestListCmd struct {
	Date string `arg:"" optional:"" help:"Day to list."`
}

type refTestGetCmd struct {
	ID     string `arg:"" help:"Thing ID."`
	Since  string `required:"" placeholder:"DATE" help:"Start date."`
	Force  bool   `short:"f" help:"Skip the check."`
	Output string `default:"out.txt" help:"Output file."`
	Kind   string `enum:"plain,fancy" default:"plain" help:"Thing kind."`
	Mode   string `enum:"on,off" default:"on" help:"Turn it on or off."`
	Secret bool   `hidden:"" help:"Not shown."`
}

var refTestGroups = []referenceGroup{{"Things & Stuff", []string{"things"}}}

func refTestRoot(t *testing.T) *kong.Node {
	t.Helper()
	var cli refTestCLI
	parser, err := kong.New(&cli, kong.Name("gccli"))
	if err != nil {
		t.Fatal(err)
	}
	return parser.Model.Node
}

func TestRenderReference(t *testing.T) {
	groups, err := buildReference(refTestRoot(t), refTestGroups)
	if err != nil {
		t.Fatal(err)
	}

	wantMarkdown := `<!-- Generated from the command tree. Do not edit. Run: go test ./internal/cmd -run TestDocsUpToDate -update -->

## Command Reference

### Things & Stuff

- ` + "`gccli things list [<date>]`: List things. Also runs as `gccli things [<date>]`." + `
  - ` + "`[<date>]`: Day to list." + `
- ` + "`gccli things get --since=DATE <id> [flags]`: Show one thing." + `
  - ` + "`<id>`: Thing ID." + `
  - ` + "`--since=DATE`: Start date." + `
  - ` + "`-f, --force`: Skip the check." + `
  - ` + "`--output=\"out.txt\"`: Output file." + `
  - ` + "`--kind=\"plain\"`: Thing kind. One of: plain, fancy." + `
  - ` + "`--mode=\"on\"`: Turn it on or off." + `
`
	if got := renderMarkdownReference(groups); got != wantMarkdown {
		t.Errorf("markdown:\n%s\nwant:\n%s", got, wantMarkdown)
	}

	wantHTML := `          <!-- Generated from the command tree. Do not edit. Run: go test ./internal/cmd -run TestDocsUpToDate -update -->

          <div class="cmd-group is-open fade-in">
            <div class="cmd-header"><h3>Things &amp; Stuff</h3><span class="cmd-arrow">&#x25B6;</span></div>
            <div class="cmd-body">
              <div class="cmd-row"><div class="cmd">gccli things list [&lt;date&gt;]</div><div class="desc">List things. Also runs as <code>gccli things [&lt;date&gt;]</code>.<br><code>[&lt;date&gt;]</code> Day to list.</div></div>
              <div class="cmd-row"><div class="cmd">gccli things get --since=DATE &lt;id&gt; [flags]</div><div class="desc">Show one thing.<br><code>&lt;id&gt;</code> Thing ID.<br><code>--since=DATE</code> Start date.<br><code>-f, --force</code> Skip the check.<br><code>--output=&#34;out.txt&#34;</code> Output file.<br><code>--kind=&#34;plain&#34;</code> Thing kind. One of: plain, fancy.<br><code>--mode=&#34;on&#34;</code> Turn it on or off.</div></div>
            </div>
          </div>
`
	if got := renderHTMLReference(groups); got != wantHTML {
		t.Errorf("html:\n%s\nwant:\n%s", got, wantHTML)
	}
}

func TestRenderReferenceSecondGroupIsClosed(t *testing.T) {
	got := renderHTMLReference([]refGroup{{Title: "A"}, {Title: "B"}})
	if strings.Count(got, `class="cmd-group is-open fade-in"`) != 1 || strings.Count(got, `class="cmd-group fade-in"`) != 1 {
		t.Errorf("want one open group and one closed group, got:\n%s", got)
	}
}

func TestBuildReferenceGroupErrors(t *testing.T) {
	root := refTestRoot(t)
	if _, err := buildReference(root, nil); err == nil || !strings.Contains(err.Error(), `"things"`) {
		t.Errorf("ungrouped command: got %v, want an error naming \"things\"", err)
	}
	groups := append([]referenceGroup{{"Extra", []string{"nope"}}}, refTestGroups...)
	if _, err := buildReference(root, groups); err == nil || !strings.Contains(err.Error(), `"nope"`) {
		t.Errorf("unknown command: got %v, want an error naming \"nope\"", err)
	}
}

func TestSpliceReference(t *testing.T) {
	doc := "intro\n  <!-- gccli:commands:begin -->\nold\nlines\n  <!-- gccli:commands:end -->\noutro\n"
	want := "intro\n  <!-- gccli:commands:begin -->\nnew\n  <!-- gccli:commands:end -->\noutro\n"
	got, err := spliceReference(doc, "new\n")
	if err != nil || got != want {
		t.Fatalf("splice: got %q, %v; want %q", got, err, want)
	}
	again, err := spliceReference(got, "new\n")
	if err != nil || again != want {
		t.Errorf("second splice: got %q, %v; want %q", again, err, want)
	}

	for name, bad := range map[string]string{
		"no begin":   "<!-- gccli:commands:end -->\n",
		"no end":     "<!-- gccli:commands:begin -->\n",
		"two begins": "<!-- gccli:commands:begin -->\n<!-- gccli:commands:begin -->\n<!-- gccli:commands:end -->\n",
		"reversed":   "<!-- gccli:commands:end -->\n<!-- gccli:commands:begin -->\n",
		"same line":  "<!-- gccli:commands:begin --><!-- gccli:commands:end -->\n",
	} {
		if _, err := spliceReference(bad, "new\n"); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

var referenceTargets = []struct {
	path   string
	render func([]refGroup) string
}{
	{"README.md", renderMarkdownReference},
	{"skills/gccli/SKILL.md", renderMarkdownReference},
	{"docs/index.html", renderHTMLReference},
}

func TestDocsUpToDate(t *testing.T) {
	groups, err := buildReference(newTestParser(t).Model.Node, referenceGroups)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range referenceTargets {
		file := filepath.Join("..", "..", filepath.FromSlash(target.path))
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		want, err := spliceReference(string(data), target.render(groups))
		if err != nil {
			t.Fatalf("%s: %v", target.path, err)
		}
		if want == string(data) {
			continue
		}
		if *update {
			if err := os.WriteFile(file, []byte(want), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		t.Errorf("%s is out of date: run go test ./internal/cmd -run TestDocsUpToDate -update", target.path)
		line, haveLine, wantLine := firstDifference(string(data), want)
		t.Errorf("first difference at line %d:\nhave: %s\nwant: %s", line, haveLine, wantLine)
	}
}

func firstDifference(a, b string) (line int, lineA, lineB string) {
	as, bs := strings.Split(a, "\n"), strings.Split(b, "\n")
	i := 0
	for i < len(as) && i < len(bs) && as[i] == bs[i] {
		i++
	}
	at := func(lines []string) string {
		if i < len(lines) {
			return lines[i]
		}
		return "(end of file)"
	}
	return i + 1, at(as), at(bs)
}
