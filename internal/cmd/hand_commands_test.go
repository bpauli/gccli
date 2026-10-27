package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

var handPassthroughAllowed = map[string]string{
	"activity comments likes remove": "a DELETE Leaf row must confirm, and a removed like comes back with one add, so the command does not prompt",
	"activity details":               "pending Leaf conversion",
	"activity exercise-sets show":    "pending Leaf conversion",
	"activity gear":                  "pending Leaf conversion",
	"activity split-summaries":       "pending Leaf conversion",
	"activity splits":                "pending Leaf conversion",
	"activity typed-splits":          "pending Leaf conversion",
	"activity weather":               "pending Leaf conversion",
	"courses detail":                 "pending Leaf conversion",
	"devices set-primary-tracker":    "pending Leaf conversion",
	"workouts detail":                "pending Leaf conversion",
}

func TestHandCommandsHaveLogic(t *testing.T) {
	fset := token.NewFileSet()
	runs := map[string]*ast.FuncDecl{}
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
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "Run" && fn.Recv != nil {
				if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
					if id, ok := star.X.(*ast.Ident); ok {
						runs[id.Name] = fn
					}
				}
			}
		}
	}
	flagged := map[string]bool{}
	walkNodes(commands, "", func(path string, n Node) {
		c, ok := n.(Custom)
		if !ok {
			return
		}
		typ := reflect.TypeOf(c.Cmd).Elem().Name()
		fn := runs[typ]
		if fn == nil {
			t.Errorf("gccli %s: no Run method on *%s in a non-test file of internal/cmd", path, typ)
			return
		}
		if !isHandPassthrough(fn) {
			return
		}
		flagged[path] = true
		if _, ok := handPassthroughAllowed[path]; !ok {
			t.Errorf("%s: gccli %s is a Custom row whose Run only sends one client.Call with a nil body and prints the answer: make it a Leaf row (CLAUDE.md, Command table)", fset.Position(fn.Pos()), path)
		}
	})
	for _, path := range slices.Sorted(maps.Keys(handPassthroughAllowed)) {
		if !flagged[path] {
			t.Errorf("handPassthroughAllowed[%q]: the row is gone or has logic now: delete the entry", path)
		}
	}
}

func isHandPassthrough(fn *ast.FuncDecl) bool {
	stmts := fn.Body.List
	if len(stmts) == 0 {
		return false
	}
	ret, ok := stmts[len(stmts)-1].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 || !isPrintCall(ret.Results[0]) {
		return false
	}
	calls := 0
	for _, s := range stmts[:len(stmts)-1] {
		switch s := s.(type) {
		case *ast.IfStmt:
			if !isErrCheck(s) {
				return false
			}
		case *ast.AssignStmt:
			if len(s.Rhs) != 1 {
				return false
			}
			call, ok := s.Rhs[0].(*ast.CallExpr)
			switch {
			case ok && isNilBodyCall(call):
				calls++
			case ok && isLookupCall(call):
			default:
				return false
			}
		default:
			return false
		}
	}
	return calls == 1 && onlyLeafCalls(fn.Body)
}

var leafHelperCalls = []string{"string", "fmt.Errorf", "strconv.Itoa"}

func onlyLeafCalls(body *ast.BlockStmt) bool {
	ok := true
	ast.Inspect(body, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall {
			return true
		}
		ok = ok && (isNilBodyCall(call) || isLookupCall(call) || isPrintCall(call) || slices.Contains(leafHelperCalls, callName(call)))
		return true
	})
	return ok
}

func callName(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		if x, ok := fun.X.(*ast.Ident); ok {
			return x.Name + "." + fun.Sel.Name
		}
	}
	return ""
}

func isErrCheck(s *ast.IfStmt) bool {
	cond, ok := s.Cond.(*ast.BinaryExpr)
	if s.Init != nil || s.Else != nil || !ok || cond.Op != token.NEQ || len(s.Body.List) != 1 {
		return false
	}
	_, isReturn := s.Body.List[0].(*ast.ReturnStmt)
	return isReturn && isIdent(cond.X, "err") && isIdent(cond.Y, "nil")
}

func isNilBodyCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Call" && len(call.Args) == 4 && isIdent(call.Args[3], "nil")
}

func isLookupCall(call *ast.CallExpr) bool {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name == "resolveClient" || fun.Name == "resolveDisplayName"
	case *ast.SelectorExpr:
		return fun.Sel.Name == "GetProfilePK"
	}
	return false
}

func isPrintCall(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return slices.Contains([]string{"writeAnswer", "writeObject", "writeResponseJSON", "reportWrite"}, fun.Name)
	case *ast.SelectorExpr:
		return fun.Sel.Name == "print" && len(call.Args) == 3 && isIdent(call.Args[1], "client") || isIdent(fun.X, "outfmt") && fun.Sel.Name == "WriteJSON"
	}
	return false
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

func TestIsHandPassthrough(t *testing.T) {
	src := `package cmd
func (c *Forward) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "a.b", map[string]string{"id": string(c.ID)}, nil)
	if err != nil {
		return fmt.Errorf("get b: %w", err)
	}
	return writeObject(g, data)
}
func (c *Named) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	name, err := resolveDisplayName(g, client)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "a.b", map[string]string{"displayName": name}, nil)
	if err != nil {
		return fmt.Errorf("get b: %w", err)
	}
	return outfmt.WriteJSON(g.Stdout, data)
}
func (c *Columns) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "a.b", nil, nil)
	if err != nil {
		return fmt.Errorf("list b: %w", err)
	}
	return bRows.print(g, client, data)
}
func (c *Sorted) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "a.b", nil, nil)
	if err != nil {
		return fmt.Errorf("list b: %w", err)
	}
	return list{noun: "b", rows: sortedRows}.print(g, data)
}
func (c *Computed) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "a.b", map[string]string{"date": c.Date.AddDays(1).String()}, nil)
	if err != nil {
		return fmt.Errorf("get b: %w", err)
	}
	return writeObject(g, data)
}
func (c *Body) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "a.b", nil, map[string]any{"x": 1})
	if err != nil {
		return fmt.Errorf("set b: %w", err)
	}
	return reportWrite(g, data, "Set b")
}
func (c *Twice) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	a, err := client.Call(g.Context, "a.a", nil, nil)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "a.b", map[string]string{"id": string(a)}, nil)
	if err != nil {
		return err
	}
	return writeObject(g, data)
}
func (c *Branch) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, "a.b", nil, nil)
	if err != nil {
		return err
	}
	if outfmt.IsJSON(g.Context) {
		return writeObject(g, data)
	}
	return printB(g, data)
}
func (c *Method) Run(g *Globals) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	data, err := client.InProgressBadges(g.Context)
	if err != nil {
		return err
	}
	return writeAnswer(g, data, "[]")
}`
	f, err := parser.ParseFile(token.NewFileSet(), "src.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, d := range f.Decls {
		fn := d.(*ast.FuncDecl)
		got[fn.Recv.List[0].Type.(*ast.StarExpr).X.(*ast.Ident).Name] = isHandPassthrough(fn)
	}
	want := map[string]bool{"Forward": true, "Named": true, "Columns": true, "Sorted": false, "Computed": false, "Body": false, "Twice": false, "Branch": false, "Method": false}
	if !maps.Equal(got, want) {
		t.Errorf("isHandPassthrough = %v, want %v: only one nil-body Call printed by a Leaf output helper is a passthrough", got, want)
	}
}
