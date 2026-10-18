package e2e

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

// The e2e job runs with -v in a public repository, so every t.Log line of a
// passing run is public. A value read from a response by key can be account
// data, such as the display name or the name of a real activity.
func TestLogsPrintNoResponseFields(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (sel.Sel.Name != "Log" && sel.Sel.Name != "Logf") {
				return true
			}
			for _, arg := range call.Args {
				if idx, ok := arg.(*ast.IndexExpr); ok {
					if lit, ok := idx.Index.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						t.Errorf("%s: %s logs the response field %s: the e2e log is public, so log that the field is present, not its value", fset.Position(arg.Pos()), sel.Sel.Name, lit.Value)
					}
				}
			}
			return true
		})
	}
}
