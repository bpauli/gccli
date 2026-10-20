package cmd

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/bpauli/gccli/internal/garminapi"
)

func (s *specLeafG[A]) Run(g *Globals) error {
	force := s.Force != nil && s.Force.(*forceArgs).Force
	return s.leaf.run(g, reflect.ValueOf(s.Args), force)
}

func (l Leaf) run(g *Globals, args reflect.Value, force bool) error {
	client, err := resolveClient(g)
	if err != nil {
		return err
	}
	in := l.inputs(args)
	if w, ok := l.Output.(Write); ok && w.Confirm != "" {
		ok, err := confirm(g.Stderr, expand(w.Confirm, in), force)
		if err != nil {
			return err
		}
		if !ok {
			g.UI.Infof("Cancelled")
			return nil
		}
	}
	params, err := l.params(g, client, in)
	if err != nil {
		return err
	}
	data, err := client.Call(g.Context, l.Endpoint, params, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", l.Verb, err)
	}
	switch out := l.Output.(type) {
	case JSON:
		return writeAnswer(g, data, out.empty())
	case Rows:
		return out.print(g, client, data)
	case Write:
		return reportWrite(g, data, "%s", expand(out.Done, in))
	}
	return fmt.Errorf("%T output is not wired: keep the command Custom", l.Output)
}

func (l Leaf) inputs(args reflect.Value) map[string]string {
	in := map[string]string{}
	read := func(name string) {
		if f, ok := argsField(args.Type(), name); ok {
			in[name], _ = requestValue(args.FieldByIndex(f.Index))
		}
	}
	for _, s := range l.Params {
		for _, a := range s.alts {
			read(a.field)
		}
	}
	if w, ok := l.Output.(Write); ok {
		for _, name := range slices.Concat(templateFields(w.Done), templateFields(w.Confirm)) {
			read(name)
		}
	}
	return in
}

func (l Leaf) params(g *Globals, client *garminapi.Client, in map[string]string) (map[string]string, error) {
	params := map[string]string{}
	for _, key := range slices.Sorted(maps.Keys(l.Params)) {
		for _, a := range l.Params[key].alts {
			v, err := a.value(g, client, in)
			if err != nil {
				return nil, err
			}
			if v != "" {
				params[key] = v
				break
			}
		}
	}
	return params, nil
}

func (a alt) value(g *Globals, client *garminapi.Client, in map[string]string) (string, error) {
	switch a.lookup {
	case lookupDisplayName:
		return resolveDisplayName(g, client)
	case lookupProfilePK:
		return client.GetProfilePK(g.Context)
	case lookupToday:
		return nowFn().Format("2006-01-02"), nil
	}
	return in[a.field], nil
}

func expand(template string, in map[string]string) string {
	var pairs []string
	for _, name := range templateFields(template) {
		pairs = append(pairs, "{"+name+"}", in[name])
	}
	return strings.NewReplacer(pairs...).Replace(template)
}
