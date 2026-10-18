package cmd

import (
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"regexp"
	"slices"

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
	"health floors",
	"health hill-score",
	"health hr",
	"health hrv",
	"health intensity-minutes view",
	"health lifestyle",
	"health max-metrics",
	"health respiration",
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

// checkTable reports every Leaf that the runner could not serve as
// written. Kong rejects an enum with no default and a flag that collides
// with a root flag itself, so the check does not repeat those.
func checkTable(nodes []Node) []error {
	var errs []error
	leaves := map[string]Leaf{}
	walkLeaves(nodes, "", func(path string, l Leaf) {
		leaves[path] = l
		errs = append(errs, checkLeaf(path, l)...)
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

func walkLeaves(nodes []Node, prefix string, visit func(path string, l Leaf)) {
	for _, n := range nodes {
		path := prefix + n.head().Name
		switch n := n.(type) {
		case Group:
			walkLeaves(n.Children, path+" ", visit)
		case Leaf:
			visit(path, n)
		}
	}
}

func checkLeaf(path string, l Leaf) []error {
	var errs []error
	fail := func(format string, a ...any) {
		errs = append(errs, fmt.Errorf(path+": "+format, a...))
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
			if _, ok := argsField(fields, a.field); a.field != "" && !ok {
				fail("Params[%q] binds %s, which is not a field of the args struct", key, a)
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
		if p.Optional || p.Default != "" {
			continue
		}
		s, bound := l.Params[p.Name]
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
			for _, m := range templateField.FindAllStringSubmatch(tpl.text, -1) {
				if _, ok := argsField(fields, m[1]); !ok {
					fail("%s names {%s}, which is not a field of the args struct", tpl.name, m[1])
				}
			}
		}
	case Rows:
		if out.Noun == "" {
			fail("Rows has no Noun")
		}
	}
	if _, isWrite := l.Output.(Write); !isWrite && l.Output != nil && e.Method != http.MethodGet {
		fail("%s is a %s: Output must be Write", l.Endpoint, e.Method)
	}
	return errs
}

// templateField matches one {Field} of a Done or Confirm template.
var templateField = regexp.MustCompile(`\{(\w+)\}`)

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

// alwaysSet reports whether Kong never leaves the field empty. A required
// field in an xor group may stay empty when the other member is given.
func alwaysSet(f reflect.StructField) bool {
	_, arg := f.Tag.Lookup("arg")
	_, optional := f.Tag.Lookup("optional")
	_, required := f.Tag.Lookup("required")
	_, xor := f.Tag.Lookup("xor")
	return (arg && !optional) || (required && !xor) || f.Tag.Get("default") != ""
}
