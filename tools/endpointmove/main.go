// Command endpointmove moves the passthrough methods of one garminapi service
// file into that file's endpoint table.
//
// Usage, from the module root:
//
//	go run ./tools/endpointmove [-area name] internal/garminapi/<service>.go
//
// A passthrough method takes a context and string or int arguments and
// returns one ConnectAPI call whose path is a literal, a concatenation of
// literals and arguments, or a fmt.Sprintf with %s and %d verbs. An optional
// `if arg == "" { arg = "x" }` before it becomes the Default of that Param.
// An `if arg != "" { path += "&key=" + arg }` after the path becomes an
// Optional query pair. The body is nil, or bytes.NewReader of one
// json.RawMessage argument, which makes a Body row.
// Each method becomes a row of <area>Endpoints named <area>.<action>, where
// action is the method name in kebab case without Get and without the area
// word; a Get of only the area word is list (plural) or get (singular).
// Every call of the method in the module becomes a Client.Call with the same
// argument expressions; an int argument is passed through strconv.Itoa,
// which prints it as %d did, and the json.RawMessage argument is the body.
// The method is deleted, the table is registered in endpoint.go and the
// name leaves the passthroughMethods list.
//
// Methods of any other shape are left alone. A second run changes nothing.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func main() {
	area := flag.String("area", "", "table area; defaults to the file name without .go")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: go run ./tools/endpointmove [-area name] internal/garminapi/<service>.go")
		os.Exit(2)
	}
	changed, err := move(".", flag.Arg(0), *area)
	if err != nil {
		fmt.Fprintln(os.Stderr, "endpointmove:", err)
		os.Exit(1)
	}
	for _, f := range changed {
		fmt.Println(f)
	}
}

type arg struct {
	name string
	conv string
	body bool
}

type moved struct {
	method   string
	args     []arg
	endpoint string
	verb     string
	path     string
	defaults map[string]string
	optional map[string]bool
	body     bool
	decl     *ast.FuncDecl
}

type edit struct {
	start, end int
	text       string
}

func move(root, file, area string) ([]string, error) {
	modPath, err := modulePath(root)
	if err != nil {
		return nil, err
	}
	svcFile := filepath.Join(root, file)
	pkgDir := filepath.Dir(svcFile)
	if area == "" {
		area = strings.TrimSuffix(filepath.Base(file), ".go")
	}
	relPkg, err := filepath.Rel(root, pkgDir)
	if err != nil {
		return nil, err
	}
	pkgImport := modPath + "/" + filepath.ToSlash(relPkg)

	src, err := os.ReadFile(svcFile)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, svcFile, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var moves []*moved
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if m := passthrough(fn); m != nil {
			m.endpoint = area + "." + action(m.method, area)
			moves = append(moves, m)
		}
	}
	if len(moves) == 0 {
		return nil, nil
	}
	byName := map[string]*moved{}
	for _, m := range moves {
		byName[m.method] = m
	}

	out := map[string][]byte{}
	svc, err := rewriteService(fset, f, src, moves, area)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	out[svcFile] = svc

	epFile := filepath.Join(pkgDir, "endpoint.go")
	ep, err := register(epFile, area+"Endpoints")
	if err != nil {
		return nil, err
	}
	out[epFile] = ep

	ptFile := filepath.Join(pkgDir, "passthrough_test.go")
	pt, err := unlist(ptFile, byName)
	if err != nil {
		return nil, err
	}
	out[ptFile] = pt

	files, err := goFiles(root)
	if err != nil {
		return nil, err
	}
	callerDirs := map[string]bool{pkgDir: true}
	for _, path := range files {
		imp, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		if slices.ContainsFunc(imp.Imports, func(s *ast.ImportSpec) bool { return s.Path.Value == strconv.Quote(pkgImport) }) {
			callerDirs[filepath.Dir(path)] = true
		}
	}
	for _, path := range files {
		if !callerDirs[filepath.Dir(path)] {
			continue
		}
		cur, ok := out[path]
		if !ok {
			if cur, err = os.ReadFile(path); err != nil {
				return nil, err
			}
		}
		next, err := rewriteCallers(path, cur, byName, modPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if next != nil {
			out[path] = next
		}
	}

	var changed []string
	for _, path := range slices.Sorted(maps.Keys(out)) {
		formatted, err := format.Source(out[path])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		old, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(old, formatted) {
			continue
		}
		if err := os.WriteFile(path, formatted, 0o644); err != nil {
			return nil, err
		}
		changed = append(changed, path)
	}
	return changed, nil
}

func goFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() && path != root && (strings.HasPrefix(name, ".") || name == "testdata" || name == "vendor" || name == "tools" || name == "bin") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(name, ".go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func modulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", errors.New("go.mod has no module line")
}

func passthrough(fn *ast.FuncDecl) *moved {
	if fn.Recv == nil || !fn.Name.IsExported() || fn.Body == nil || len(fn.Recv.List[0].Names) != 1 {
		return nil
	}
	star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
	if !ok || !isIdent(star.X, "Client") {
		return nil
	}
	recv := fn.Recv.List[0].Names[0].Name
	if !returnsRawJSON(fn.Type.Results) {
		return nil
	}

	m := &moved{method: fn.Name.Name, decl: fn, defaults: map[string]string{}, optional: map[string]bool{}}
	types := map[string]string{}
	var ctx string
	for i, field := range fn.Type.Params.List {
		typ := exprString(field.Type)
		for _, n := range field.Names {
			switch {
			case i == 0 && typ == "context.Context" && ctx == "":
				ctx = n.Name
			case typ == "string":
				m.args = append(m.args, arg{name: n.Name})
			case typ == "int":
				m.args = append(m.args, arg{name: n.Name, conv: "strconv.Itoa"})
			case typ == "json.RawMessage" && !m.body:
				m.args = append(m.args, arg{name: n.Name, body: true})
				m.body = true
			default:
				return nil
			}
			types[n.Name] = typ
		}
	}
	if ctx == "" {
		return nil
	}

	stmts := fn.Body.List
	for len(stmts) > 0 {
		name, val, ok := defaultStmt(stmts[0])
		if !ok || types[name] != "string" {
			break
		}
		m.defaults[name] = val
		stmts = stmts[1:]
	}
	var pathVar string
	var pathExpr ast.Expr
	if as, ok := stmts[0].(*ast.AssignStmt); ok && len(stmts) > 1 {
		if as.Tok != token.DEFINE || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return nil
		}
		id, ok := as.Lhs[0].(*ast.Ident)
		if !ok {
			return nil
		}
		pathVar, pathExpr = id.Name, as.Rhs[0]
		stmts = stmts[1:]
	}
	var suffixes []suffix
	for pathVar != "" && len(stmts) > 1 {
		sf, ok := optionalStmt(stmts[0], pathVar)
		if !ok || types[sf.name] != "string" || m.defaults[sf.name] != "" || m.optional[sf.name] {
			return nil
		}
		m.optional[sf.name] = true
		suffixes = append(suffixes, sf)
		stmts = stmts[1:]
	}
	if len(stmts) != 1 {
		return nil
	}
	ret, ok := stmts[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return nil
	}
	call, ok := ret.Results[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 4 {
		return nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !isIdent(sel.X, recv) || sel.Sel.Name != "ConnectAPI" || !isIdent(call.Args[0], ctx) || !bodyArg(call.Args[3], m) {
		return nil
	}
	verb, ok := call.Args[1].(*ast.SelectorExpr)
	if !ok || !isIdent(verb.X, "http") || !strings.HasPrefix(verb.Sel.Name, "Method") {
		return nil
	}
	m.verb = "http." + verb.Sel.Name
	if pathVar != "" {
		if !isIdent(call.Args[2], pathVar) {
			return nil
		}
	} else {
		pathExpr = call.Args[2]
	}

	used := map[string]bool{}
	path, ok := template(pathExpr, types, used)
	if !ok {
		return nil
	}
	for _, sf := range suffixes {
		if used[sf.name] {
			return nil
		}
		used[sf.name] = true
		path += sf.lit + "{" + sf.name + "}"
	}
	if !validTemplate(path) {
		return nil
	}
	for _, a := range m.args {
		if !a.body && !used[a.name] {
			return nil
		}
	}
	m.path = path
	return m
}

func bodyArg(e ast.Expr, m *moved) bool {
	if isIdent(e, "nil") {
		return !m.body
	}
	call, ok := e.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || exprString(call.Fun) != "bytes.NewReader" {
		return false
	}
	for _, a := range m.args {
		if a.body {
			return isIdent(call.Args[0], a.name)
		}
	}
	return false
}

type suffix struct {
	lit, name string
}

func optionalStmt(s ast.Stmt, pathVar string) (suffix, bool) {
	is, ok := s.(*ast.IfStmt)
	if !ok || is.Init != nil || is.Else != nil || len(is.Body.List) != 1 {
		return suffix{}, false
	}
	cond, ok := is.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.NEQ || !isString(cond.Y, "") {
		return suffix{}, false
	}
	id, ok := cond.X.(*ast.Ident)
	if !ok {
		return suffix{}, false
	}
	as, ok := is.Body.List[0].(*ast.AssignStmt)
	if !ok || as.Tok != token.ADD_ASSIGN || len(as.Lhs) != 1 || len(as.Rhs) != 1 || !isIdent(as.Lhs[0], pathVar) {
		return suffix{}, false
	}
	add, ok := as.Rhs[0].(*ast.BinaryExpr)
	if !ok || add.Op != token.ADD || !isIdent(add.Y, id.Name) {
		return suffix{}, false
	}
	lit, ok := add.X.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return suffix{}, false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil || len(v) < 3 || (v[0] != '&' && v[0] != '?') || !strings.HasSuffix(v, "=") {
		return suffix{}, false
	}
	return suffix{lit: v, name: id.Name}, true
}

func returnsRawJSON(res *ast.FieldList) bool {
	return res != nil && len(res.List) == 2 && exprString(res.List[0].Type) == "json.RawMessage" && exprString(res.List[1].Type) == "error"
}

func defaultStmt(s ast.Stmt) (name, val string, ok bool) {
	is, isIf := s.(*ast.IfStmt)
	if !isIf || is.Init != nil || is.Else != nil || len(is.Body.List) != 1 {
		return "", "", false
	}
	cond, isBin := is.Cond.(*ast.BinaryExpr)
	if !isBin || cond.Op != token.EQL || !isString(cond.Y, "") {
		return "", "", false
	}
	id, isID := cond.X.(*ast.Ident)
	if !isID {
		return "", "", false
	}
	as, isAssign := is.Body.List[0].(*ast.AssignStmt)
	if !isAssign || as.Tok != token.ASSIGN || len(as.Lhs) != 1 || len(as.Rhs) != 1 || !isIdent(as.Lhs[0], id.Name) {
		return "", "", false
	}
	lit, isLit := as.Rhs[0].(*ast.BasicLit)
	if !isLit || lit.Kind != token.STRING {
		return "", "", false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil || v == "" {
		return "", "", false
	}
	return id.Name, v, true
}

func template(e ast.Expr, types map[string]string, used map[string]bool) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		s, err := strconv.Unquote(e.Value)
		return s, err == nil && e.Kind == token.STRING && !strings.ContainsAny(s, "{}")
	case *ast.Ident:
		if types[e.Name] != "string" {
			return "", false
		}
		used[e.Name] = true
		return "{" + e.Name + "}", true
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		l, ok := template(e.X, types, used)
		if !ok {
			return "", false
		}
		r, ok := template(e.Y, types, used)
		return l + r, ok
	case *ast.CallExpr:
		sel, ok := e.Fun.(*ast.SelectorExpr)
		if !ok || !isIdent(sel.X, "fmt") || sel.Sel.Name != "Sprintf" || len(e.Args) < 1 || e.Ellipsis.IsValid() {
			return "", false
		}
		lit, ok := e.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return "", false
		}
		f, err := strconv.Unquote(lit.Value)
		if err != nil || strings.ContainsAny(f, "{}") {
			return "", false
		}
		args := e.Args[1:]
		var b strings.Builder
		for i := 0; i < len(f); i++ {
			if f[i] != '%' {
				b.WriteByte(f[i])
				continue
			}
			if i+1 == len(f) {
				return "", false
			}
			i++
			v := f[i]
			if v == '%' {
				b.WriteByte('%')
				continue
			}
			if (v != 's' && v != 'd') || len(args) == 0 {
				return "", false
			}
			id, ok := args[0].(*ast.Ident)
			args = args[1:]
			want := map[byte]string{'s': "string", 'd': "int"}[v]
			if !ok || types[id.Name] != want {
				return "", false
			}
			used[id.Name] = true
			b.WriteString("{" + id.Name + "}")
		}
		return b.String(), len(args) == 0
	}
	return "", false
}

var (
	pathTextRe  = regexp.MustCompile(`^[A-Za-z0-9._~/-]*$`)
	queryTextRe = regexp.MustCompile(`^[A-Za-z0-9._~-]*$`)
	holeRe      = regexp.MustCompile(`\{[a-zA-Z][a-zA-Z0-9]*\}`)
)

func validTemplate(p string) bool {
	pathPart, query, hasQuery := strings.Cut(p, "?")
	if !strings.HasPrefix(pathPart, "/") || !pathTextRe.MatchString(holeRe.ReplaceAllString(pathPart, "")) {
		return false
	}
	if !hasQuery {
		return true
	}
	for _, kv := range strings.Split(query, "&") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" || !queryTextRe.MatchString(k) {
			return false
		}
		wholeHole := v != "" && holeRe.FindString(v) == v
		if !wholeHole && !queryTextRe.MatchString(v) {
			return false
		}
	}
	return true
}

func action(method, area string) string {
	words := splitCamel(method)
	get := words[0] == "Get" && len(words) > 1
	if get {
		words = words[1:]
	}
	var rest []string
	plural := false
	for _, w := range words {
		if singular(w) == singular(area) {
			plural = plural || singular(w) != strings.ToLower(w)
			continue
		}
		rest = append(rest, w)
	}
	switch {
	case len(rest) > 0:
		return strings.ToLower(strings.Join(rest, "-"))
	case get && plural:
		return "list"
	case get:
		return "get"
	}
	return strings.ToLower(strings.Join(words, "-"))
}

func singular(word string) string {
	w := strings.ToLower(word)
	if s, ok := strings.CutSuffix(w, "ies"); ok {
		return s + "y"
	}
	return strings.TrimSuffix(w, "s")
}

func splitCamel(s string) []string {
	var words []string
	r := []rune(s)
	start := 0
	for i := 1; i < len(r); i++ {
		lowerBefore := unicode.IsLower(r[i-1]) || unicode.IsDigit(r[i-1])
		acronymEnd := unicode.IsUpper(r[i-1]) && i+1 < len(r) && unicode.IsLower(r[i+1])
		if unicode.IsUpper(r[i]) && (lowerBefore || acronymEnd) {
			words = append(words, string(r[start:i]))
			start = i
		}
	}
	return append(words, string(r[start:]))
}

func rowText(m *moved) string {
	body := ""
	if m.body {
		body = ", Body: true"
	}
	var params []string
	for _, a := range m.args {
		if v, ok := m.defaults[a.name]; ok {
			params = append(params, fmt.Sprintf("{Name: %q, Default: %q}", a.name, v))
		}
		if m.optional[a.name] {
			params = append(params, fmt.Sprintf("{Name: %q, Optional: true}", a.name))
		}
	}
	if len(params) == 0 {
		return fmt.Sprintf("\t{Name: %q, Method: %s, Host: ConnectAPI, Path: %q%s},\n", m.endpoint, m.verb, m.path, body)
	}
	return fmt.Sprintf("\t{\n\t\tName: %q, Method: %s, Host: ConnectAPI,\n\t\tPath:   %q,\n\t\tParams: []Param{%s},%s\n\t},\n",
		m.endpoint, m.verb, m.path, strings.Join(params, ", "), body)
}

func rewriteService(fset *token.FileSet, f *ast.File, src []byte, moves []*moved, area string) ([]byte, error) {
	var edits []edit
	for _, m := range moves {
		start := m.decl.Pos()
		if m.decl.Doc != nil {
			start = m.decl.Doc.Pos()
		}
		end := fset.Position(m.decl.End()).Offset
		for end < len(src) && src[end] == '\n' {
			end++
		}
		edits = append(edits, edit{start: fset.Position(start).Offset, end: end})
	}
	var rows strings.Builder
	for _, m := range moves {
		rows.WriteString(rowText(m))
	}

	tableName := area + "Endpoints"
	var table *ast.CompositeLit
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, s := range gd.Specs {
			vs := s.(*ast.ValueSpec)
			if len(vs.Names) == 1 && vs.Names[0].Name == tableName && len(vs.Values) == 1 {
				table, _ = vs.Values[0].(*ast.CompositeLit)
			}
		}
	}
	if table != nil {
		at := fset.Position(table.Rbrace).Offset
		text := rows.String()
		if src[at-1] != '\n' {
			text = "\n" + text
		}
		edits = append(edits, edit{start: at, end: at, text: text})
	} else {
		at := fset.Position(f.Name.End()).Offset
		for _, d := range f.Decls {
			if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
				at = fset.Position(gd.End()).Offset
			}
		}
		edits = append(edits, edit{start: at, end: at, text: "\n\nvar " + tableName + " = []Endpoint{\n" + rows.String() + "}\n"})
	}
	out := apply(src, edits)
	return fixImports(out, []string{"net/http"}, "")
}

func apply(src []byte, edits []edit) []byte {
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := slices.Clone(src)
	for _, e := range edits {
		out = slices.Concat(out[:e.start], []byte(e.text), out[e.end:])
	}
	return out
}

func register(file, table string) ([]byte, error) {
	src, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var idx *ast.CallExpr
	ast.Inspect(f, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok && isIdent(c.Fun, "mustIndex") {
			idx = c
		}
		return idx == nil
	})
	if idx == nil {
		return nil, fmt.Errorf("%s: no mustIndex call", file)
	}
	for _, a := range idx.Args {
		if isIdent(a, table) {
			return src, nil
		}
	}
	at := fset.Position(idx.Rparen).Offset
	return apply(src, []edit{{start: at, end: at, text: "\t" + table + ",\n"}}), nil
}

func unlist(file string, moves map[string]*moved) ([]byte, error) {
	src, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var list *ast.CompositeLit
	ast.Inspect(f, func(n ast.Node) bool {
		if vs, ok := n.(*ast.ValueSpec); ok && len(vs.Names) == 1 && vs.Names[0].Name == "passthroughMethods" && len(vs.Values) == 1 {
			list, _ = vs.Values[0].(*ast.CompositeLit)
		}
		return list == nil
	})
	if list == nil {
		return nil, fmt.Errorf("%s: no passthroughMethods list", file)
	}
	var keep []string
	for _, el := range list.Elts {
		lit, ok := el.(*ast.BasicLit)
		if !ok {
			return nil, fmt.Errorf("%s: passthroughMethods holds a non-literal", file)
		}
		name, err := strconv.Unquote(lit.Value)
		if err != nil {
			return nil, err
		}
		if moves[name] == nil {
			keep = append(keep, lit.Value)
		}
	}
	if len(keep) == len(list.Elts) {
		return src, nil
	}
	const width = 110
	var b strings.Builder
	line := ""
	for _, q := range keep {
		item := q + ","
		if line != "" && len(line)+1+len(item) > width {
			b.WriteString(line + "\n")
			line = ""
		}
		if line == "" {
			line = "\t" + item
		} else {
			line += " " + item
		}
	}
	if line != "" {
		b.WriteString(line + "\n")
	}
	start := fset.Position(list.Lbrace).Offset + 1
	end := fset.Position(list.Rbrace).Offset
	return apply(src, []edit{{start: start, end: end, text: "\n" + b.String()}}), nil
}

func rewriteCallers(path string, src []byte, moves map[string]*moved, modPath string) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	text := func(n ast.Node) string {
		return string(src[fset.Position(n.Pos()).Offset:fset.Position(n.End()).Offset])
	}
	var edits []edit
	needStrconv := false
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		m := moves[sel.Sel.Name]
		if m == nil || len(call.Args) != 1+len(m.args) || call.Ellipsis.IsValid() {
			return true
		}
		values, body := "nil", "nil"
		var kv []string
		for i, a := range m.args {
			v := text(call.Args[i+1])
			switch {
			case a.body:
				body = v
				continue
			case a.conv != "" && decimalRe.MatchString(v):
				v = strconv.Quote(v)
			case a.conv != "":
				v = a.conv + "(" + v + ")"
				needStrconv = true
			}
			kv = append(kv, strconv.Quote(a.name)+": "+v)
		}
		if len(kv) > 0 {
			values = "map[string]string{" + strings.Join(kv, ", ") + "}"
		}
		edits = append(edits, edit{
			start: fset.Position(call.Pos()).Offset,
			end:   fset.Position(call.End()).Offset,
			text:  fmt.Sprintf("%s.Call(%s, %q, %s, %s)", text(sel.X), text(call.Args[0]), m.endpoint, values, body),
		})
		return false
	})
	if len(edits) == 0 {
		return nil, nil
	}
	var need []string
	if needStrconv {
		need = append(need, "strconv")
	}
	return fixImports(apply(src, edits), need, modPath)
}

func fixImports(src []byte, need []string, modPath string) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	used := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				used[id.Name] = true
			}
		}
		return true
	})
	type spec struct{ name, path string }
	var specs []spec
	changed := false
	for _, s := range f.Imports {
		p, _ := strconv.Unquote(s.Path.Value)
		name := ""
		if s.Name != nil {
			name = s.Name.Name
		}
		std := !strings.Contains(strings.SplitN(p, "/", 2)[0], ".")
		if std && name == "" && !used[pkgName(p)] {
			changed = true
			continue
		}
		specs = append(specs, spec{name, p})
	}
	for _, p := range need {
		if !slices.ContainsFunc(specs, func(s spec) bool { return s.path == p }) {
			specs = append(specs, spec{"", p})
			changed = true
		}
	}
	if !changed {
		return src, nil
	}

	group := func(p string) int {
		switch {
		case modPath != "" && (p == modPath || strings.HasPrefix(p, modPath+"/")):
			return 2
		case strings.Contains(strings.SplitN(p, "/", 2)[0], "."):
			return 1
		}
		return 0
	}
	sort.SliceStable(specs, func(i, j int) bool {
		gi, gj := group(specs[i].path), group(specs[j].path)
		if gi != gj {
			return gi < gj
		}
		return specs[i].path < specs[j].path
	})
	var b strings.Builder
	switch {
	case len(specs) == 1 && specs[0].name == "":
		b.WriteString("import " + strconv.Quote(specs[0].path))
	case len(specs) > 0:
		b.WriteString("import (\n")
		for i, s := range specs {
			if i > 0 && group(specs[i-1].path) != group(s.path) {
				b.WriteString("\n")
			}
			b.WriteString("\t")
			if s.name != "" {
				b.WriteString(s.name + " ")
			}
			b.WriteString(strconv.Quote(s.path) + "\n")
		}
		b.WriteString(")")
	}

	var decls []*ast.GenDecl
	for _, d := range f.Decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			decls = append(decls, gd)
		}
	}
	if len(decls) == 0 {
		at := fset.Position(f.Name.End()).Offset
		return apply(src, []edit{{start: at, end: at, text: "\n\n" + b.String()}}), nil
	}
	start := fset.Position(decls[0].Pos()).Offset
	end := fset.Position(decls[len(decls)-1].End()).Offset
	return apply(src, []edit{{start: start, end: end, text: b.String()}}), nil
}

var (
	versionRe = regexp.MustCompile(`^v[0-9]+$`)
	decimalRe = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
)

func pkgName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 1 && versionRe.MatchString(parts[len(parts)-1]) {
		return parts[len(parts)-2]
	}
	return parts[len(parts)-1]
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

func isString(e ast.Expr, want string) bool {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return false
	}
	s, err := strconv.Unquote(lit.Value)
	return err == nil && s == want
}

func exprString(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	}
	return ""
}
