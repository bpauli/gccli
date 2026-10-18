package cmd

import (
	"fmt"
	"maps"
	"net/http"
	"reflect"
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

	fields := argsFields(l.spec().fields(), nil)
	for _, name := range l.LateDates {
		if f, ok := fields[name]; !ok || f.typ != reflect.TypeFor[string]() {
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
			if _, ok := fields[a.field]; a.field != "" && !ok {
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

	w, isWrite := l.Output.(Write)
	switch {
	case e.Method == http.MethodGet && isWrite:
		fail("Output is Write, but %s is a GET", l.Endpoint)
	case e.Method != http.MethodGet && !isWrite:
		fail("%s is a %s: Output must be Write", l.Endpoint, e.Method)
	case isWrite && w.Done == "":
		fail("Write has no Done")
	case e.Method == http.MethodDelete && w.Confirm == "":
		fail("%s is a DELETE: Write needs Confirm", l.Endpoint)
	}
	return errs
}

// A LateDates field is never empty: resolveDate gives today for "".
func (s Source) sure(fields map[string]argsField, late []string) bool {
	for _, a := range s.alts {
		if a.lookup != noLookup || fields[a.field].alwaysSet || slices.Contains(late, a.field) {
			return true
		}
	}
	return false
}

type argsField struct {
	typ       reflect.Type
	alwaysSet bool
}

func argsFields(t reflect.Type, out map[string]argsField) map[string]argsField {
	if out == nil {
		out = map[string]argsField{}
	}
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if _, embed := f.Tag.Lookup("embed"); embed || f.Anonymous {
			argsFields(f.Type, out)
			continue
		}
		_, arg := f.Tag.Lookup("arg")
		_, optional := f.Tag.Lookup("optional")
		_, required := f.Tag.Lookup("required")
		_, def := f.Tag.Lookup("default")
		out[f.Name] = argsField{typ: f.Type, alwaysSet: (arg && !optional) || required || def}
	}
	return out
}
