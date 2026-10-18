package cmd

import (
	"reflect"
	"slices"
	"strconv"

	"github.com/alecthomas/kong"
)

// Node is one entry of the command tree: a Group, a Custom or a Leaf.
type Node interface{ head() head }

type head struct {
	Name    string
	Help    string
	Default bool
}

// Group is a command with children and no Run of its own.
type Group struct {
	Name     string
	Help     string
	Children []Node
}

// Custom is a leaf whose Kong struct and Run are hand-written. Cmd is a
// pointer to that struct. The table only places it in the tree.
type Custom struct {
	Name    string
	Help    string
	Default bool
	Cmd     interface{ Run(*Globals) error }
}

// Leaf is a leaf described as data: its CLI inputs (Spec), the request
// (Endpoint, Params), the error wrap (Verb) and what it prints (Output).
type Leaf struct {
	Name    string
	Help    string
	Default bool

	// Spec is the args struct with Kong tags, from argsOf. nil means no inputs.
	Spec     argsSpec
	Endpoint string
	Params   Params
	// Verb wraps a request error: "get sleep: <err>".
	Verb   string
	Output Output
	// LateDates names string fields that Run resolves with resolveDate after
	// resolveClient, so a bad date exits 1 or 3 instead of 2. Only the paths
	// in v1LateDates may set it.
	LateDates []string
}

func (g Group) head() head  { return head{Name: g.Name, Help: g.Help} }
func (c Custom) head() head { return head{Name: c.Name, Help: c.Help, Default: c.Default} }
func (l Leaf) head() head   { return head{Name: l.Name, Help: l.Help, Default: l.Default} }

func (l Leaf) spec() argsSpec {
	if l.Spec == nil {
		return argsOf(&struct{}{})
	}
	return l.Spec
}

func (l Leaf) confirms() bool {
	w, ok := l.Output.(Write)
	return ok && w.Confirm != ""
}

// Params binds each placeholder of the endpoint to the source of its value.
// The key is the placeholder name as the endpoint row writes it.
type Params map[string]Source

// Source is where one request value comes from: In(field), DisplayName,
// ProfilePK or Today. Or adds a fallback that is used when every source
// before it is empty.
type Source struct{ alts []alt }

type alt struct {
	field  string
	lookup lookup
}

type lookup uint8

const (
	noLookup lookup = iota
	lookupDisplayName
	lookupProfilePK
	lookupToday
)

// In reads the args field with this Go name, found by Go's promotion
// rules: a field of an anonymous embedded struct counts, a field of a
// named embed:"" struct does not.
func In(field string) Source { return Source{[]alt{{field: field}}} }

var (
	// DisplayName is the profile display name, from one GET of profile.get.
	DisplayName = Source{[]alt{{lookup: lookupDisplayName}}}
	// ProfilePK is the profile PK, as GetProfilePK resolves it.
	ProfilePK = Source{[]alt{{lookup: lookupProfilePK}}}
	// Today is the clock date, YYYY-MM-DD, from nowFn.
	Today = Source{[]alt{{lookup: lookupToday}}}
)

func (s Source) Or(fallback Source) Source {
	return Source{slices.Concat(s.alts, fallback.alts)}
}

func (s Source) String() string {
	if len(s.alts) == 0 {
		return "Source{}"
	}
	out := s.alts[0].String()
	for _, a := range s.alts[1:] {
		out += ".Or(" + a.String() + ")"
	}
	return out
}

func (a alt) String() string {
	switch a.lookup {
	case lookupDisplayName:
		return "DisplayName"
	case lookupProfilePK:
		return "ProfilePK"
	case lookupToday:
		return "Today"
	}
	return "In(" + strconv.Quote(a.field) + ")"
}

// Output is what a Leaf prints: JSON, JSONNull, Rows or Write.
type Output interface{ output() }

// JSON prints the response as indented JSON in every mode and {} on an
// empty body.
type JSON struct{}

// JSONNull is JSON that prints null on an empty body. Only the paths in
// v1NullJSON may use it.
type JSONNull struct{}

// Rows prints the response as JSON in --json mode and as columns otherwise.
// Items names the wrapper key of the list, or "" for a root array. A
// missing key gives no rows. SortBy names a numeric path to sort by,
// ascending and stable; "" keeps Garmin's order. Noun names the list in
// the parse error: "parse <noun>: <err>".
type Rows struct {
	Noun    string
	Items   string
	SortBy  string
	Columns []Column
}

// Column is one table column. Path is a dotted JSON path.
type Column struct {
	Label string
	Path  string
}

// Write is the write rule: --json prints the response, other modes print
// Done on stderr. Done and Confirm are templates where {Field} is the raw
// value of that args field. A non-empty Confirm adds --force/-f to the
// leaf and asks before any request.
type Write struct {
	Done    string
	Confirm string
}

func (JSON) output()     {}
func (JSONNull) output() {}
func (Rows) output()     {}
func (Write) output()    {}

type argsSpec interface {
	target(force bool) any
	fields() reflect.Type
}

// argsOf wraps an args struct literal with Kong tags as a Leaf Spec. The
// generic target keeps the struct concrete, so Kong finds every hook on it
// and on its embedded types, and copies it by value, so Kong never writes
// into the table.
func argsOf[A any](args *A) argsSpec { return argsOfT[A]{args} }

type argsOfT[A any] struct{ args *A }

func (s argsOfT[A]) target(force bool) any {
	t := specLeafG[A]{Args: *s.args}
	if force {
		t.Force = &forceArgs{}
	}
	return t
}

func (s argsOfT[A]) fields() reflect.Type { return reflect.TypeFor[A]() }

// specLeafG is the Kong target of every Leaf. Force is nil or *forceArgs;
// Kong skips a nil interface embed.
type specLeafG[A any] struct {
	Args  A   `embed:""`
	Force any `embed:""`
}

type forceArgs struct {
	Force bool `help:"Skip confirmation prompt." short:"f"`
}

func attach(nodes []Node) kong.Plugins {
	var plugins kong.Plugins
	for _, n := range nodes {
		plugins = append(plugins, holder(n))
	}
	return plugins
}

type group struct{ kong.Plugins }

// The node's value is held by value, because Kong replaces a pointer-typed
// cmd field with a fresh zero value.
func holder(n Node) any {
	var cmd reflect.Value
	switch n := n.(type) {
	case Group:
		cmd = reflect.ValueOf(group{attach(n.Children)})
	case Custom:
		cmd = reflect.Indirect(reflect.ValueOf(n.Cmd))
	case Leaf:
		cmd = reflect.ValueOf(n.spec().target(n.confirms()))
	}
	h := n.head()
	tag := `cmd:"" name:` + strconv.Quote(h.Name) + ` help:` + strconv.Quote(h.Help)
	if h.Default {
		tag += ` default:"withargs"`
	}
	v := reflect.New(reflect.StructOf([]reflect.StructField{{Name: "Cmd", Type: cmd.Type(), Tag: reflect.StructTag(tag)}}))
	v.Elem().Field(0).Set(cmd)
	return v.Interface()
}
