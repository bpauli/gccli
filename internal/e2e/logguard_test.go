package e2e

// This file has no build tag, so `go test ./...` runs the guard without
// Garmin credentials.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// allowedLogFieldReads lists the log lines that may print a field read,
// because the field holds no account data.
var allowedLogFieldReads = map[string]string{
	"Exercise %d: %s/%s": "exercise catalog names are public",
}

// TestNoAccountDataLogged fails when a t.Log or t.Logf call in the e2e suite
// prints a value read from a response (a map index or a struct field). The
// e2e log is public, and a response field can hold account data such as a
// display name or the name of a real activity. Log that a value is present,
// or its length, instead.
func TestNoAccountDataLogged(t *testing.T) {
	fset := token.NewFileSet()
	calls := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		bad, n := leakingLogCalls(fset, f)
		calls += n
		for _, pos := range bad {
			t.Errorf("%s: log prints a field read from a response; log its presence or length instead", pos)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls == 0 {
		t.Fatal("found no t.Log or t.Logf calls; the guard checks nothing")
	}
}

func TestLeakingLogCallsDetectsFieldReads(t *testing.T) {
	src := `package e2e
func f(t T, m map[string]any, p P, n int) {
	t.Logf("profile for: %v", m["displayName"])
	t.Logf("activity: %s", p.Name)
	t.Log(m["courseName"])
	t.Logf("got %d items", len(m["items"].([]any)))
	t.Logf("got %d bytes", n)
	t.Logf("Exercise %d: %s/%s", n, p.Category, p.Name)
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	bad, calls := leakingLogCalls(fset, f)
	if calls != 6 {
		t.Errorf("calls = %d, want 6", calls)
	}
	want := []string{"x.go:3:2", "x.go:4:2", "x.go:5:2"}
	if strings.Join(bad, " ") != strings.Join(want, " ") {
		t.Errorf("leaking calls = %v, want %v", bad, want)
	}
}

// leakingLogCalls returns the positions of the t.Log and t.Logf calls in f
// that print a map index or a field read outside len(), and the number of log
// calls it saw. A response value copied into a local variable first is not
// caught.
func leakingLogCalls(fset *token.FileSet, f *ast.File) (bad []string, calls int) {
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "Log" && sel.Sel.Name != "Logf") {
			return true
		}
		calls++
		args := call.Args
		if sel.Sel.Name == "Logf" && len(args) > 0 {
			if lit, ok := args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if format, err := strconv.Unquote(lit.Value); err == nil {
					if _, ok := allowedLogFieldReads[format]; ok {
						return true
					}
				}
			}
			args = args[1:]
		}
		for _, arg := range args {
			if readsField(arg) {
				bad = append(bad, fset.Position(call.Pos()).String())
				break
			}
		}
		return true
	})
	return bad, calls
}

// readsField reports whether e reads a map index or a field outside a len()
// call.
func readsField(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "len" {
				return false
			}
		case *ast.IndexExpr, *ast.SelectorExpr:
			found = true
		}
		return !found
	})
	return found
}
