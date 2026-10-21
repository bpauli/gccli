package garminapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestNoNewPassthroughMethods(t *testing.T) {
	fset := token.NewFileSet()
	got := map[string]token.Position{}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && isPassthrough(fn) {
				got[fn.Name.Name] = fset.Position(fn.Pos())
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(got)) {
		t.Errorf("%s: Client.%s only forwards one request: add a row to the service's endpoint table and call it with Client.Call", got[name], name)
	}
}

const maxPassthroughStatements = 3

func isPassthrough(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || !fn.Name.IsExported() || fn.Body == nil || len(fn.Body.List) > maxPassthroughStatements {
		return false
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok || len(fn.Recv.List[0].Names) == 0 {
		return false
	}
	if id, ok := star.X.(*ast.Ident); !ok || id.Name != "Client" {
		return false
	}
	recv := fn.Recv.List[0].Names[0].Name
	loopSends := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ForStmt:
			loopSends = loopSends || slices.ContainsFunc(recvCalls(n.Body, recv), isRequest)
		case *ast.RangeStmt:
			loopSends = loopSends || slices.ContainsFunc(recvCalls(n.Body, recv), isRequest)
		}
		return true
	})
	onRecv := recvCalls(fn.Body, recv)
	return !loopSends && len(onRecv) == 1 && isRequest(onRecv[0])
}

func recvCalls(n ast.Node, recv string) []string {
	var methods []string
	ast.Inspect(n, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == recv {
					methods = append(methods, sel.Sel.Name)
				}
			}
		}
		return true
	})
	return methods
}

func isRequest(method string) bool {
	return method == "ConnectAPI" || method == "Call"
}

func TestIsPassthrough(t *testing.T) {
	src := `package garminapi
func (c *Client) Forward(ctx context.Context) (json.RawMessage, error) {
	return c.Call(ctx, "a.b", nil, nil)
}
func (c *Client) Pages(ctx context.Context) error {
	for range 3 {
		if _, err := c.Call(ctx, "a.b", nil, nil); err != nil {
			return err
		}
	}
	return nil
}
func (c *Client) DeadLoop(ctx context.Context) (json.RawMessage, error) {
	for range 0 {
	}
	return c.Call(ctx, "a.b", nil, nil)
}
func (c *Client) Options(ctx context.Context, opts map[string]string) (json.RawMessage, error) {
	args := map[string]string{}
	for k, v := range opts {
		args[k] = v
	}
	return c.Call(ctx, "a.b", args, nil)
}`
	f, err := parser.ParseFile(token.NewFileSet(), "src.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, d := range f.Decls {
		fn := d.(*ast.FuncDecl)
		got[fn.Name.Name] = isPassthrough(fn)
	}
	if want := map[string]bool{"Forward": true, "Pages": false, "DeadLoop": true, "Options": true}; !maps.Equal(got, want) {
		t.Errorf("isPassthrough = %v, want %v: only a Call in a loop body sends more than one request", got, want)
	}
}
