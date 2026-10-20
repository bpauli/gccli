package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/bpauli/gccli/internal/outfmt"
)

// Garmin's answer to a read is a json.RawMessage, nil for a 204 or an
// empty body. An empty answer is not an error: a detail read prints {}
// in every mode, and a list read prints [] in --json mode and no rows
// otherwise. This file holds that rule.

// unmarshalAnswer parses an answer into v. An empty answer leaves v as
// it is.
func unmarshalAnswer(data json.RawMessage, v any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// writeAnswer prints an answer as indented JSON, or empty for an empty
// answer.
func writeAnswer(g *Globals, data json.RawMessage, empty string) error {
	if len(data) == 0 {
		data = json.RawMessage(empty)
	}
	return outfmt.WriteJSON(g.Stdout, data)
}

// writeObject prints the answer of a detail read as JSON in every mode.
func writeObject(g *Globals, data json.RawMessage) error {
	return writeAnswer(g, data, "{}")
}

// list is what a list read prints: the answer in --json mode, and the
// rows of its items otherwise. A root array is the items. An object holds
// them under the first of keys that holds a list, and an object without
// such a key has no items.
type list struct {
	// noun names the list in a parse error: "parse <noun>: <err>".
	noun   string
	keys   []string
	header []string
	rows   func(items []map[string]any) [][]string
}

func (l list) print(g *Globals, data json.RawMessage) error {
	if outfmt.IsJSON(g.Context) {
		return writeAnswer(g, data, "[]")
	}
	items, err := l.items(data)
	if err != nil {
		return fmt.Errorf("parse %s: %w", l.noun, err)
	}
	return outfmt.WriteRows(g.Context, g.Stdout, l.header, l.rows(items))
}

// items keeps the element rules of the hand-written parsers it replaced:
// a root array fails on an element that is not an object, and a wrapped
// list skips it.
func (l list) items(data json.RawMessage) ([]map[string]any, error) {
	var items []map[string]any
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("[")) {
		err := unmarshalAnswer(data, &items)
		return items, err
	}
	var wrapper map[string]any
	if err := unmarshalAnswer(data, &wrapper); err != nil {
		return nil, err
	}
	for _, key := range l.keys {
		list, ok := wrapper[key].([]any)
		if !ok {
			continue
		}
		for _, v := range list {
			if m, ok := v.(map[string]any); ok {
				items = append(items, m)
			}
		}
		return items, nil
	}
	return nil, nil
}
