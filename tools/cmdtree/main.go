// Command cmdtree moves the Kong command tree of internal/cmd from struct
// fields into one []Node value. It replaces each group struct (a struct whose
// fields all carry a cmd tag) with a Group literal in the same file, keeps
// RootFlags and Version in CLI and gives CLI kong.Plugins, and writes the
// top-level nodes as `var commands`. It fails on any tag the Node types
// cannot carry, so nothing is dropped unseen.
//
// usage: go run ./tools/cmdtree internal/cmd
package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

type field struct {
	name string
	help string
	def  bool
	typ  string
	doc  *ast.CommentGroup
	pos  token.Pos
	end  token.Pos
}

type tree struct {
	fset       *token.FileSet
	files      map[string]*ast.File
	src        map[string][]byte
	childrenOf map[string][]field
	decls      map[string]*ast.GenDecl
	holderOf   map[string]field
	root       []field
	cli        *ast.GenDecl
	edits      map[string][]edit
	errs       []string
}

type edit struct {
	start, end int
	text       string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: cmdtree <package dir>")
		os.Exit(2)
	}
	t := &tree{
		fset:       token.NewFileSet(),
		files:      map[string]*ast.File{},
		src:        map[string][]byte{},
		childrenOf: map[string][]field{},
		decls:      map[string]*ast.GenDecl{},
		holderOf:   map[string]field{},
		edits:      map[string][]edit{},
	}
	if err := t.load(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	t.collect()
	t.check()
	if len(t.errs) > 0 {
		for _, e := range t.errs {
			fmt.Fprintln(os.Stderr, e)
		}
		os.Exit(1)
	}
	t.rewrite()
	if err := t.write(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (t *tree) load(dir string) error {
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return err
	}
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		f, err := parser.ParseFile(t.fset, name, src, parser.ParseComments)
		if err != nil {
			return err
		}
		t.files[name] = f
		t.src[name] = src
	}
	return nil
}

func (t *tree) collect() {
	for _, f := range t.files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE || len(gd.Specs) != 1 {
				continue
			}
			ts := gd.Specs[0].(*ast.TypeSpec)
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			fields, all := t.cmdFields(st)
			switch {
			case ts.Name.Name == "CLI":
				t.cli = gd
				t.root = fields
			case len(fields) > 0 && all:
				t.childrenOf[ts.Name.Name] = fields
				t.decls[ts.Name.Name] = gd
			case len(fields) > 0:
				t.failf(gd.Pos(), "%s mixes cmd fields with other fields: extend Group before the move", ts.Name.Name)
			}
		}
	}
}

func (t *tree) cmdFields(st *ast.StructType) (fields []field, all bool) {
	all = true
	for _, f := range st.Fields.List {
		if f.Tag == nil || len(f.Names) != 1 {
			all = false
			continue
		}
		raw, err := strconv.Unquote(f.Tag.Value)
		if err != nil {
			t.failf(f.Pos(), "bad tag: %v", err)
			continue
		}
		tag := reflect.StructTag(raw)
		if _, ok := tag.Lookup("cmd"); !ok {
			all = false
			continue
		}
		fd := field{doc: f.Doc, pos: f.Pos(), end: f.End()}
		for _, key := range tagKeys(raw) {
			val := tag.Get(key)
			switch key {
			case "cmd":
			case "name":
				fd.name = val
			case "help":
				fd.help = val
			case "default":
				if val != "withargs" {
					t.failf(f.Pos(), "default:%q is not withargs: extend Custom before the move", val)
				}
				fd.def = true
			default:
				t.failf(f.Pos(), "tag key %q has no Node field: extend the Node types before the move", key)
			}
		}
		if fd.name == "" {
			fd.name = kongName(f.Names[0].Name)
		}
		id, ok := f.Type.(*ast.Ident)
		if !ok {
			t.failf(f.Pos(), "field type is not a local type")
			continue
		}
		fd.typ = id.Name
		fields = append(fields, fd)
	}
	return fields, all
}

func (t *tree) check() {
	if t.cli == nil {
		t.errs = append(t.errs, "no CLI struct")
	}
	uses := map[string]int{}
	var walk func(fields []field)
	walk = func(fields []field) {
		for _, f := range fields {
			if f.doc != nil {
				t.failf(f.pos, "field has a doc comment that no Node field keeps")
			}
			if children, ok := t.childrenOf[f.typ]; ok {
				uses[f.typ]++
				t.holderOf[f.typ] = f
				walk(children)
			}
		}
	}
	walk(t.root)
	for name := range t.childrenOf {
		if uses[name] != 1 {
			t.failf(t.decls[name].Pos(), "group %s is used by %d fields, want 1", name, uses[name])
		}
	}
	for _, f := range t.files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			typ := fd.Recv.List[0].Type
			if star, ok := typ.(*ast.StarExpr); ok {
				typ = star.X
			}
			if id, ok := typ.(*ast.Ident); ok && t.childrenOf[id.Name] != nil {
				t.failf(fd.Pos(), "group %s has a method %s: a Group has no hooks", id.Name, fd.Name.Name)
			}
		}
	}
}

func (t *tree) rewrite() {
	for name, gd := range t.decls {
		file := t.fset.File(gd.Pos()).Name()
		start := gd.Pos()
		if gd.Doc != nil {
			start = gd.Doc.Pos()
		}
		holder := t.holderOf[name]
		text := "var " + groupVar(name) + " = Group{Name: " + strconv.Quote(holder.name) + ", Help: " + strconv.Quote(holder.help) + ", Children: []Node{\n" + t.children(t.childrenOf[name]) + "}}"
		t.edit(file, start, gd.End(), text)
	}

	file := t.fset.File(t.cli.Pos()).Name()
	first, last := t.root[0], t.root[len(t.root)-1]
	t.edit(file, first.pos, last.end, "kong.Plugins")
	var b strings.Builder
	b.WriteString("\n\nvar commands = []Node{\n")
	for _, f := range t.root {
		b.WriteString(t.nodeLiteral(f) + ",\n")
	}
	b.WriteString("}")
	t.edit(file, t.cli.End(), t.cli.End(), b.String())
}

func (t *tree) children(fields []field) string {
	var b strings.Builder
	for _, f := range fields {
		b.WriteString(t.nodeLiteral(f) + ",\n")
	}
	return b.String()
}

func (t *tree) nodeLiteral(f field) string {
	if _, ok := t.childrenOf[f.typ]; ok {
		return groupVar(f.typ)
	}
	def := ""
	if f.def {
		def = ", Default: true"
	}
	return "Custom{Name: " + strconv.Quote(f.name) + ", Help: " + strconv.Quote(f.help) + def + ", Cmd: &" + f.typ + "{}}"
}

func (t *tree) edit(file string, start, end token.Pos, text string) {
	t.edits[file] = append(t.edits[file], edit{t.fset.Position(start).Offset, t.fset.Position(end).Offset, text})
}

func (t *tree) write() error {
	files := slices.Sorted(maps.Keys(t.edits))
	for _, file := range files {
		edits := t.edits[file]
		slices.SortFunc(edits, func(a, b edit) int { return b.start - a.start })
		src := t.src[file]
		for _, e := range edits {
			src = slices.Concat(src[:e.start], []byte(e.text), src[e.end:])
		}
		out, err := format.Source(src)
		if err != nil {
			return fmt.Errorf("%s: %w\n%s", file, err, src)
		}
		if err := os.WriteFile(file, out, 0o644); err != nil {
			return err
		}
	}
	leaves := 0
	var count func(fields []field)
	count = func(fields []field) {
		for _, f := range fields {
			if children, ok := t.childrenOf[f.typ]; ok {
				count(children)
			} else {
				leaves++
			}
		}
	}
	count(t.root)
	fmt.Printf("%d group structs replaced, %d leaves, %d files written\n", len(t.childrenOf), leaves, len(files))
	return nil
}

func (t *tree) failf(pos token.Pos, format string, a ...any) {
	t.errs = append(t.errs, t.fset.Position(pos).String()+": "+fmt.Sprintf(format, a...))
}

func groupVar(typ string) string {
	name := strings.TrimSuffix(strings.TrimSuffix(typ, "Cmd"), "Group")
	return strings.ToLower(name[:1]) + name[1:] + "Group"
}

func tagKeys(tag string) []string {
	var keys []string
	for tag != "" {
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		keys = append(keys, tag[:i])
		tag = tag[i+1:]
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		tag = tag[i+1:]
	}
	return keys
}

// kongName is Kong's camelCase split (kong/camelcase.go) joined with
// dashes in lower case: the name a field with no name tag gets.
func kongName(field string) string {
	class := func(r rune) int {
		switch {
		case unicode.IsLower(r):
			return 1
		case unicode.IsUpper(r):
			return 2
		case unicode.IsDigit(r):
			return 3
		}
		return 4
	}
	var runs [][]rune
	last := 0
	for _, r := range field {
		if c := class(r); c == last {
			runs[len(runs)-1] = append(runs[len(runs)-1], r)
		} else {
			runs = append(runs, []rune{r})
			last = c
		}
	}
	for i := 0; i < len(runs)-1; i++ {
		if unicode.IsUpper(runs[i][0]) && unicode.IsLower(runs[i+1][0]) {
			runs[i+1] = append([]rune{runs[i][len(runs[i])-1]}, runs[i+1]...)
			runs[i] = runs[i][:len(runs[i])-1]
		}
	}
	var words []string
	for _, run := range runs {
		if len(run) > 0 {
			words = append(words, string(run))
		}
	}
	return strings.ToLower(strings.Join(words, "-"))
}
