package cmd

import (
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
)

// v1LateDates lists the paths whose positional date is checked after
// login (exit 1 or 3) instead of at parse time (exit 2), as their
// <leaf>@bad-date.golden files pin. The major release that moves the dates
// to parse time deletes it; until then the list only shrinks.
var v1LateDates = []string{
	"body composition",
	"body daily-weigh-ins",
	"health body-battery view",
	"health endurance-score",
	"health events",
	"health fitness-age",
	"health floors view",
	"health hill-score",
	"health hr view",
	"health hrv",
	"health intensity-minutes view",
	"health lifestyle",
	"health max-metrics",
	"health respiration view",
	"health rhr",
	"health sleep",
	"health spo2",
	"health steps view",
	"health stress view",
	"health summary",
	"health training-readiness",
	"health training-status",
	"hydration view",
	"nutrition food-log",
	"nutrition meals",
	"nutrition settings",
	"reload",
}

// v1NullJSON lists the paths that print null on an empty body, as their
// <leaf>@empty.golden files pin. The major release that prints {}
// everywhere deletes it; until then the list only shrinks.
var v1NullJSON = []string{
	"activity details",
	"activity exercise-sets show",
	"activity gear",
	"activity hr-zones",
	"activity power-zones",
	"activity split-summaries",
	"activity splits",
	"activity typed-splits",
	"activity weather",
	"courses detail",
	"workouts detail",
}

// checkTable reports every defect of the command tree: a Custom without
// a struct Cmd; a Leaf whose endpoint, placeholders, In fields, enums,
// required bindings, int bindings, frozen-list paths, Output type or
// Rows columns and Write templates do not fit. Each fail message in
// checkLeaf states one rule.
func checkTable(nodes []Node) []error {
	var errs []error
	leaves := map[string]Leaf{}
	walkNodes(nodes, "", func(path string, n Node) {
		switch n := n.(type) {
		case Custom:
			if n.Cmd == nil || reflect.Indirect(reflect.ValueOf(n.Cmd)).Kind() != reflect.Struct {
				errs = append(errs, fmt.Errorf("%s: Custom needs a Cmd that points to a struct, got %T", path, n.Cmd))
			}
		case Leaf:
			leaves[path] = n
			errs = append(errs, checkLeaf(path, n)...)
		}
	})
	for _, path := range v1LateDates {
		if l, ok := leaves[path]; ok && len(l.LateDates) == 0 {
			errs = append(errs, fmt.Errorf("%s: v1LateDates names this path, but the row has no LateDates: remove the entry", path))
		}
	}
	for _, path := range v1NullJSON {
		if l, ok := leaves[path]; ok && !isNullJSON(l) {
			errs = append(errs, fmt.Errorf("%s: v1NullJSON names this path, but the row's Output is not JSONNull: remove the entry", path))
		}
	}
	return errs
}

func isNullJSON(l Leaf) bool {
	_, ok := l.Output.(JSONNull)
	return ok
}

func walkNodes(nodes []Node, prefix string, visit func(path string, n Node)) {
	for _, n := range nodes {
		path := prefix + n.head().Name
		visit(path, n)
		if g, ok := n.(Group); ok {
			walkNodes(g.Children, path+" ", visit)
		}
	}
}

func checkLeaf(path string, l Leaf) []error {
	var errs []error
	fail := func(format string, a ...any) {
		errs = append(errs, fmt.Errorf(path+": "+format, a...))
	}

	if err := l.spec().check(); err != nil {
		fail("%s", err)
		return errs
	}
	fields := l.spec().fields()
	for _, name := range l.LateDates {
		if f, ok := argsField(fields, name); !ok || f.Type != reflect.TypeFor[string]() {
			fail("LateDates names %q, which is not a string field of the args struct", name)
		}
	}
	if len(l.LateDates) > 0 && !slices.Contains(v1LateDates, path) {
		fail("LateDates is allowed only on the paths in v1LateDates")
	}
	if isNullJSON(l) && !slices.Contains(v1NullJSON, path) {
		fail("JSONNull is allowed only on the paths in v1NullJSON")
	}
	for _, key := range slices.Sorted(maps.Keys(l.Params)) {
		for _, a := range l.Params[key].alts {
			if a.lookup != noLookup {
				continue
			}
			f, ok := argsField(fields, a.field)
			switch {
			case a.field == "":
				fail("Params[%q] binds In(\"\"), which names no field", key)
			case !ok:
				fail("Params[%q] binds %s, which is not a field of the args struct", key, a)
			case f.Tag.Get("kong") == "-":
				fail("Params[%q] binds %s, which Kong skips (kong:\"-\")", key, a)
			default:
				if _, sendable := requestValue(reflect.Zero(f.Type)); !sendable {
					fail("Params[%q] binds %s of type %s: the runner sends only string, int and Date", key, a, f.Type)
				}
			}
		}
	}

	e, ok := garminapi.Describe(l.Endpoint)
	if !ok {
		fail("endpoint %q is not in the endpoint table", l.Endpoint)
		return errs
	}
	placeholder := func(name string) bool {
		return slices.ContainsFunc(e.Params, func(p garminapi.Param) bool { return p.Name == name })
	}
	for _, key := range slices.Sorted(maps.Keys(l.Params)) {
		if !placeholder(key) {
			fail("Params[%q] is not a placeholder of %s", key, l.Endpoint)
		}
	}
	for _, p := range e.Params {
		s, bound := l.Params[p.Name]
		for _, a := range s.alts {
			f, ok := argsField(fields, a.field)
			if !ok {
				continue
			}
			if (p.Optional || p.Default != "") && f.Type == reflect.TypeFor[int]() {
				fail("Params[%q] binds %s of type int, but {%s} of %s may be left out: the runner sends an int even when it is 0, so use a string field", p.Name, a, p.Name, l.Endpoint)
			}
			if p.Enum == nil {
				continue
			}
			for _, v := range kongEnum(f) {
				if !slices.Contains(p.Enum, v) {
					fail("Params[%q] binds %s, whose enum allows %q, which {%s} of %s does not: %s", p.Name, a, v, p.Name, l.Endpoint, strings.Join(p.Enum, ", "))
				}
			}
		}
		if p.Optional || p.Default != "" {
			continue
		}
		switch {
		case !bound:
			fail("placeholder {%s} of %s is required, but Params does not bind it", p.Name, l.Endpoint)
		case !s.sure(fields, l.LateDates):
			fail("placeholder {%s} of %s is required, but %s may be empty: make the field required, give it a default, or add .Or(...)", p.Name, l.Endpoint, s)
		}
	}

	if e.Body {
		fail("%s takes a request body, which no Source gives: keep the command Custom", l.Endpoint)
	}
	switch out := l.Output.(type) {
	case nil:
		fail("Output is nil")
	case Write:
		switch {
		case e.Method == http.MethodGet:
			fail("Output is Write, but %s is a GET", l.Endpoint)
		case out.Done == "":
			fail("Write has no Done")
		case e.Method == http.MethodDelete && out.Confirm == "":
			fail("%s is a DELETE: Write needs Confirm", l.Endpoint)
		}
		for _, tpl := range []struct{ name, text string }{{"Done", out.Done}, {"Confirm", out.Confirm}} {
			for _, name := range templateFields(tpl.text) {
				f, ok := argsField(fields, name)
				if !ok {
					fail("%s names {%s}, which is not a field of the args struct", tpl.name, name)
				} else if _, sendable := requestValue(reflect.Zero(f.Type)); !sendable {
					fail("%s names {%s} of type %s: a template takes only string, int and Date", tpl.name, name, f.Type)
				}
			}
		}
	case Rows:
		if out.Noun == "" {
			fail("Rows has no Noun")
		}
		if len(out.Columns) == 0 {
			fail("Rows has no Columns")
		}
		for i, c := range out.Columns {
			switch {
			case c.Path == "":
				fail("Columns[%d] (%s) has no Path", i, c.Label)
			case int(c.Format) >= len(formatNames):
				fail("Columns[%d] (%s) has %s, which is not a Format", i, c.Label, c.Format)
			}
		}
	}
	if _, isWrite := l.Output.(Write); !isWrite && l.Output != nil && e.Method != http.MethodGet {
		fail("%s is a %s: Output must be Write", l.Endpoint, e.Method)
	}
	return errs
}

func templateFields(text string) []string {
	var names []string
	for {
		open := strings.IndexByte(text, '{')
		if open < 0 {
			return names
		}
		length := strings.IndexByte(text[open:], '}')
		if length < 0 {
			return names
		}
		names = append(names, text[open+1:open+length])
		text = text[open+length+1:]
	}
}

func requestValue(v reflect.Value) (value string, ok bool) {
	switch x := v.Interface().(type) {
	case string:
		return x, true
	case Date:
		return string(x), true
	case int:
		return strconv.Itoa(x), true
	}
	return "", false
}

func kongEnum(f reflect.StructField) []string {
	tag, ok := f.Tag.Lookup("enum")
	if !ok {
		return nil
	}
	values := strings.Split(tag, ",")
	for i, v := range values {
		values[i] = strings.TrimSpace(v)
	}
	return values
}

// A LateDates field is never empty: resolveDate gives today for "".
func (s Source) sure(fields reflect.Type, late []string) bool {
	for _, a := range s.alts {
		if a.lookup != noLookup || slices.Contains(late, a.field) {
			return true
		}
		if f, ok := argsField(fields, a.field); ok && alwaysSet(f) {
			return true
		}
	}
	return false
}

// argsField finds the args field that In(name) reads, by Go's promotion
// rules: an outer field shadows an embedded one, and a named embed:""
// field is not promoted. The runner reads the parsed value with
// FieldByIndex(f.Index), so the check and the runner agree on the field.
func argsField(t reflect.Type, name string) (reflect.StructField, bool) {
	f, ok := t.FieldByName(name)
	return f, ok && f.IsExported()
}

// Kong lets a required flag in an xor group stay empty when another member
// of the group is given.
func alwaysSet(f reflect.StructField) bool {
	_, arg := f.Tag.Lookup("arg")
	_, optional := f.Tag.Lookup("optional")
	_, required := f.Tag.Lookup("required")
	_, xor := f.Tag.Lookup("xor")
	return (arg && !optional) || (required && !xor) || f.Tag.Get("default") != ""
}
