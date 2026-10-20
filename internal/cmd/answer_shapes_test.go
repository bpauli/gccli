package cmd

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// answerShapesFile is the answer shape registry: one line per read that
// prints the Garmin answer as JSON, with the shape of that answer and the
// evidence. The shape decides what an empty body prints, and
// TestAnswerShapes keeps the rows and the main goldens in step with it, so
// a hand-written golden body cannot make a row of the wrong shape pass.
const answerShapesFile = "testdata/answer_shapes.tsv"

const answerShapesHeader = `# The answer shape of every read that prints the Garmin answer as JSON (a
# Leaf row with a JSON output, or a Custom whose Run calls writeObject or
# writeAnswer), one line per command path: path, shape, source. The shape
# is array, object or unknown. A Leaf row is JSON{List: true} exactly when
# its shape is array, so an empty body prints []. An unknown shape prints
# {}. TestAnswerShapes checks the rows, the main goldens and the empty
# output of each Custom against this file. TestAnswerShapesLive refreshes
# it from the raw Garmin answers, read-only, with the stored login of the
# test account (see its comment):
#   ANSWER_SHAPES_CONFIG=<XDG_CONFIG_HOME of that login> go test -tags answershapes ./internal/cmd -run TestAnswerShapesLive -update
`

type answerShape struct {
	Shape  string
	Source string
}

var answerShapeNames = []string{"array", "object", "unknown"}

// bodyShape classifies a raw Garmin answer. A 204 or a body without a
// value is "empty", which is no evidence of a shape. Anything that is not
// an array or an object is "".
func bodyShape(status int, body []byte) string {
	b := bytes.TrimSpace(body)
	switch {
	case status == http.StatusNoContent || len(b) == 0:
		return "empty"
	case b[0] == '[':
		return "array"
	case b[0] == '{':
		return "object"
	}
	return ""
}

func TestBodyShape(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"204", http.StatusNoContent, "", "empty"},
		{"204 with a body", http.StatusNoContent, "[]", "empty"},
		{"empty 200", http.StatusOK, "", "empty"},
		{"whitespace", http.StatusOK, " \n\t", "empty"},
		{"empty array", http.StatusOK, "[]", "array"},
		{"array with whitespace", http.StatusOK, "\n [1]\n", "array"},
		{"empty object", http.StatusOK, "{}", "object"},
		{"null", http.StatusOK, "null", ""},
		{"number", http.StatusOK, "42", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bodyShape(tt.status, []byte(tt.body)); got != tt.want {
				t.Errorf("bodyShape(%d, %q) = %q, want %q", tt.status, tt.body, got, tt.want)
			}
		})
	}
}

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

// answerRead is a command that prints the Garmin answer as JSON: a Leaf
// row with a JSON output, or a Custom whose Run calls writeObject or
// writeAnswer. A Custom has no Leaf.
type answerRead struct {
	leaf   Leaf
	custom bool
}

// answerReads returns every read that prints the Garmin answer, by command
// path.
func answerReads(t *testing.T) map[string]answerRead {
	t.Helper()
	printers := runPrinters(t)
	reads := map[string]answerRead{}
	walkNodes(commands, "", func(path string, n Node) {
		switch n := n.(type) {
		case Leaf:
			if _, ok := n.Output.(JSON); ok {
				reads[path] = answerRead{leaf: n}
			}
		case Custom:
			if printers[reflect.TypeOf(n.Cmd).Elem().Name()] {
				reads[path] = answerRead{custom: true}
			}
		}
	})
	return reads
}

// runPrinters returns the receiver types of the package whose Run method
// calls writeObject or writeAnswer, from the source files.
func runPrinters(t *testing.T) map[string]bool {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	printers := map[string]bool{}
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Run" || fn.Recv == nil {
				continue
			}
			star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			recv, ok := star.X.(*ast.Ident)
			if !ok {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if id, ok := call.Fun.(*ast.Ident); ok && (id.Name == "writeObject" || id.Name == "writeAnswer") {
						printers[recv.Name] = true
					}
				}
				return true
			})
		}
	}
	return printers
}

// emptyValue is what a read of the shape prints for an empty body.
func emptyValue(shape string) string {
	if shape == "array" {
		return "[]"
	}
	return "{}"
}

// TestAnswerShapes checks every read that prints the Garmin answer against
// the registry: the read has a line, a Leaf row has JSON.List set exactly
// for an array, a Custom prints the empty value of its shape for an empty
// body, and the main golden answers the data route with a 200 body of the
// registry's shape.
func TestAnswerShapes(t *testing.T) {
	shapes := readAnswerShapes(t)
	data, err := os.ReadFile(answerShapesFile)
	if err != nil {
		t.Fatal(err)
	}
	if want := formatAnswerShapes(shapes); !bytes.Equal(data, want) {
		t.Errorf("%s is not in its canonical form, the header of answer_shapes_test.go and one line per path in sorted order (- file, + canonical):\n%s", answerShapesFile, lineDiff(string(data), string(want)))
	}
	reads := answerReads(t)
	for _, path := range slices.Sorted(maps.Keys(shapes)) {
		if _, ok := reads[path]; !ok {
			t.Errorf("%s names %q, which is not a read that prints the Garmin answer: remove the line", answerShapesFile, path)
		}
	}
	for _, path := range slices.Sorted(maps.Keys(reads)) {
		r := reads[path]
		s, ok := shapes[path]
		if !ok {
			t.Errorf("%s has no line for %q: add <path>\\t<shape>\\t<source>, with the shape of the Garmin answer (array, object or unknown) and where it was seen", answerShapesFile, path)
			continue
		}
		if r.custom {
			checkCustomEmpty(t, path, s)
		} else {
			out := r.leaf.Output.(JSON)
			switch {
			case s.Shape == "array" && !out.List:
				t.Errorf("%s: %s says the answer is an array (%s), but the row is JSON{}: set JSON{List: true}, so an empty body prints []", path, answerShapesFile, s.Source)
			case s.Shape != "array" && out.List:
				t.Errorf("%s: the row is JSON{List: true}, but %s says the answer is %s (%s): make the row JSON{}, or correct the registry with its evidence", path, answerShapesFile, s.Shape, s.Source)
			}
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
		if got := bodyShape(http.StatusOK, []byte(reply.body)); got != s.Shape {
			t.Errorf("%s: the main golden answers %s with %s %q, but %s says the answer is %s (%s): give the golden a body of that shape, then run go test ./internal/cmd -run TestGolden -update", path, route, got, reply.body, answerShapesFile, s.Shape, s.Source)
		}
	}
}

// checkCustomEmpty runs the main golden of a Custom read with an empty 200
// body on every data route, in --json mode, and requires the empty value
// of the registry's shape and exit 0.
func checkCustomEmpty(t *testing.T, path string, s answerShape) {
	t.Helper()
	t.Run(path+"/empty body", func(t *testing.T) {
		args, in, err := goldenInput(goldenFile(path))
		if err != nil {
			t.Fatal(err)
		}
		for route := range in.routes {
			if !isLookup(route) {
				in.routes[route] = mockReply{}
			}
		}
		h := newHarness(t, in)
		stdout, stderr, code := h.run([]string{"--json"}, args)
		if want := emptyValue(s.Shape) + "\n"; code != 0 || stdout != want || stderr != "" {
			t.Errorf("an empty body gives exit %d, stdout %q and stderr %q; %s says the answer is %s (%s), so want exit 0 and %q", code, stdout, stderr, answerShapesFile, s.Shape, s.Source, want)
		}
	})
}
