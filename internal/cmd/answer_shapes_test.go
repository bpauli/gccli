package cmd

import (
	"bytes"
	"fmt"
	"maps"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
)

// answerShapesFile is the answer shape registry: one line per Leaf row with
// a JSON output, with the shape of its Garmin answer and the evidence. The
// shape decides JSON.List, and TestAnswerShapes keeps the rows and the main
// goldens in step with it, so a hand-written golden body cannot make a row
// of the wrong shape pass.
const answerShapesFile = "testdata/answer_shapes.tsv"

const answerShapesHeader = `# The answer shape of every Leaf row with a JSON output, one line per
# command path: path, shape, source. The shape is array, object or unknown.
# A row is JSON{List: true} exactly when its shape is array, so an empty
# body prints []. An unknown shape prints {}. TestAnswerShapes checks the
# rows and the main goldens against this file. TestAnswerShapesLive
# refreshes it from Garmin, read-only, through a wrapper that runs gccli
# with the login of the test account (see its comment):
#   ANSWER_SHAPES_BIN=/path/to/wrapper go test -tags answershapes ./internal/cmd -run TestAnswerShapesLive -update
`

type answerShape struct {
	Shape  string
	Source string
}

var answerShapeNames = []string{"array", "object", "unknown"}

func readAnswerShapes(t *testing.T) map[string]answerShape {
	t.Helper()
	data, err := os.ReadFile(answerShapesFile)
	if err != nil {
		t.Fatal(err)
	}
	shapes := map[string]answerShape{}
	for i, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		switch {
		case len(f) != 3 || f[0] == "" || f[2] == "":
			t.Fatalf("%s:%d: want <path>\\t<shape>\\t<source>, got %q", answerShapesFile, i+1, line)
		case !slices.Contains(answerShapeNames, f[1]):
			t.Fatalf("%s:%d: shape %q is not array, object or unknown", answerShapesFile, i+1, f[1])
		}
		if _, dup := shapes[f[0]]; dup {
			t.Fatalf("%s:%d: %q appears twice", answerShapesFile, i+1, f[0])
		}
		shapes[f[0]] = answerShape{Shape: f[1], Source: f[2]}
	}
	return shapes
}

func formatAnswerShapes(shapes map[string]answerShape) []byte {
	var b bytes.Buffer
	b.WriteString(answerShapesHeader)
	for _, path := range slices.Sorted(maps.Keys(shapes)) {
		fmt.Fprintf(&b, "%s\t%s\t%s\n", path, shapes[path].Shape, shapes[path].Source)
	}
	return b.Bytes()
}

// jsonLeaves returns every Leaf row with a JSON output by its command path.
func jsonLeaves() map[string]Leaf {
	leaves := map[string]Leaf{}
	walkNodes(commands, "", func(path string, n Node) {
		if l, ok := n.(Leaf); ok {
			if _, ok := l.Output.(JSON); ok {
				leaves[path] = l
			}
		}
	})
	return leaves
}

// TestAnswerShapes checks every Leaf row with a JSON output against the
// registry: the row has a line, JSON.List is set exactly for an array, and
// the main golden answers the data route with a 200 body of that shape.
func TestAnswerShapes(t *testing.T) {
	shapes := readAnswerShapes(t)
	data, err := os.ReadFile(answerShapesFile)
	if err != nil {
		t.Fatal(err)
	}
	if want := formatAnswerShapes(shapes); !bytes.Equal(data, want) {
		t.Errorf("%s is not in its canonical form, the header of answer_shapes_test.go and one line per path in sorted order (- file, + canonical):\n%s", answerShapesFile, lineDiff(string(data), string(want)))
	}
	leaves := jsonLeaves()
	for _, path := range slices.Sorted(maps.Keys(shapes)) {
		if _, ok := leaves[path]; !ok {
			t.Errorf("%s names %q, which is not a Leaf row with a JSON output: remove the line", answerShapesFile, path)
		}
	}
	for _, path := range slices.Sorted(maps.Keys(leaves)) {
		out := leaves[path].Output.(JSON)
		s, ok := shapes[path]
		if !ok {
			t.Errorf("%s has no line for %q: add <path>\\t<shape>\\t<source>, with the shape of the Garmin answer (array, object or unknown) and where it was seen", answerShapesFile, path)
			continue
		}
		switch {
		case s.Shape == "array" && !out.List:
			t.Errorf("%s: %s says the answer is an array (%s), but the row is JSON{}: set JSON{List: true}, so an empty body prints []", path, answerShapesFile, s.Source)
		case s.Shape != "array" && out.List:
			t.Errorf("%s: the row is JSON{List: true}, but %s says the answer is %s (%s): make the row JSON{}, or correct the registry with its evidence", path, answerShapesFile, s.Shape, s.Source)
		}
		if s.Shape == "unknown" {
			continue
		}
		g, err := readGolden(goldenFile(path))
		if err != nil {
			t.Error(err)
			continue
		}
		_, in, err := g.input()
		if err != nil {
			t.Error(err)
			continue
		}
		route := dataRoute(g)
		if route == "" {
			t.Errorf("%s: the main golden sends no request, so nothing pins the answer shape", path)
			continue
		}
		reply := in.routes[route]
		if reply.status != 0 && reply.status != http.StatusOK {
			t.Errorf("%s: the main golden answers %s with status %d: give it a 200 and a body of the registry's shape (%s)", path, route, reply.status, s.Shape)
			continue
		}
		body := bytes.TrimSpace([]byte(reply.body))
		got := "object"
		if bytes.HasPrefix(body, []byte("[")) {
			got = "array"
		}
		if got != s.Shape {
			t.Errorf("%s: the main golden answers %s with %s %q, but %s says the answer is %s (%s): give the golden a body of that shape, then run go test ./internal/cmd -run TestGolden -update", path, route, got, body, answerShapesFile, s.Shape, s.Source)
		}
	}
}
