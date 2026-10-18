package cmd

import (
	"encoding/json"
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
	in, err := l.inputs(args)
	if err != nil {
		return err
	}
	emit, err := l.printer(g, in)
	if err != nil {
		return err
	}
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
	return emit(data)
}

func (l Leaf) printer(g *Globals, in map[string]string) (func(json.RawMessage) error, error) {
	switch out := l.Output.(type) {
	case JSON:
		return func(data json.RawMessage) error { return writeHealthJSON(g, data) }, nil
	case Write:
		return func(data json.RawMessage) error { return reportWrite(g, data, "%s", expand(out.Done, in)) }, nil
	}
	return nil, fmt.Errorf("%T output is not wired yet: keep the command Custom", l.Output)
}

func (l Leaf) inputs(args reflect.Value) (map[string]string, error) {
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
	for _, name := range l.LateDates {
		date, err := resolveDate(in[name])
		if err != nil {
			return nil, err
		}
		in[name] = date
	}
	return in, nil
}

func (l Leaf) params(g *Globals, client *garminapi.Client, in map[string]string) (map[string]string, error) {
	params := map[string]string{}
	memo := map[lookup]string{}
	for _, key := range slices.Sorted(maps.Keys(l.Params)) {
		for _, a := range l.Params[key].alts {
			v, err := a.value(g, client, in, memo)
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

func (a alt) value(g *Globals, client *garminapi.Client, in map[string]string, memo map[lookup]string) (string, error) {
	if a.lookup == noLookup {
		return in[a.field], nil
	}
	if v, ok := memo[a.lookup]; ok {
		return v, nil
	}
	var v string
	var err error
	switch a.lookup {
	case lookupDisplayName:
		v, err = resolveDisplayName(g, client)
	case lookupProfilePK:
		v, err = client.GetProfilePK(g.Context)
	case lookupToday:
		v = nowFn().Format("2006-01-02")
	}
	if err != nil {
		return "", err
	}
	memo[a.lookup] = v
	return v, nil
}

func expand(template string, in map[string]string) string {
	var pairs []string
	for _, name := range templateFields(template) {
		pairs = append(pairs, "{"+name+"}", in[name])
	}
	return strings.NewReplacer(pairs...).Replace(template)
}
