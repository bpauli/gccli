package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bpauli/gccli/internal/outfmt"
)

func TestListItems(t *testing.T) {
	cases := []struct {
		name string
		keys []string
		data string
		want int
		err  string
	}{
		{"empty answer is no items", nil, "", 0, ""},
		{"empty answer under a key is no items", []string{"k"}, "", 0, ""},
		{"root array", nil, `[{"a":1},{"a":2}]`, 2, ""},
		{"root array with whitespace", []string{"k"}, "\n [{\"a\":1}]", 1, ""},
		{"root array fails on an element that is not an object", nil, `[{"a":1},2]`, 0, "cannot unmarshal number"},
		{"wrapped list", []string{"k"}, `{"k":[{"a":1}],"n":1}`, 1, ""},
		{"wrapped list skips an element that is not an object", []string{"k"}, `{"k":[{"a":1},2,"x"]}`, 1, ""},
		{"second key holds the list", []string{"k", "r"}, `{"r":[{"a":1},{"a":2}]}`, 2, ""},
		{"first key wins", []string{"k", "r"}, `{"k":[{"a":1}],"r":[{"a":2},{"a":3}]}`, 1, ""},
		{"object without the key is no items", []string{"k"}, `{"other":[{"a":1}]}`, 0, ""},
		{"object with a key that is not a list is no items", []string{"k"}, `{"k":{"a":1}}`, 0, ""},
		{"object answer to a root array list is no items", nil, `{"message":"none"}`, 0, ""},
		{"null answer is no items", []string{"k"}, `null`, 0, ""},
		{"not JSON fails", []string{"k"}, `not json`, 0, "invalid character"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items, err := list{keys: tc.keys}.items(json.RawMessage(tc.data))
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("err = %v, want one that contains %q", err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatalf("items: %v", err)
			}
			if len(items) != tc.want {
				t.Errorf("got %d items, want %d", len(items), tc.want)
			}
		})
	}
}

// Every list read prints the same thing for an empty answer: [] in json
// mode, the header in table mode and with --header, nothing in plain mode.
func TestListPrintEmptyAnswer(t *testing.T) {
	l := list{noun: "things", keys: []string{"things"}, header: []string{"ID", "NAME"}, rows: func(items []map[string]any) [][]string {
		rows := make([][]string, 0, len(items))
		for _, m := range items {
			rows = append(rows, []string{textOf(m["id"]), textOf(m["name"])})
		}
		return rows
	}}
	cases := []struct {
		name string
		mode outfmt.Mode
		data string
		want string
	}{
		{"json empty body", outfmt.JSON, "", "[]\n"},
		{"json missing key prints the answer", outfmt.JSON, `{"n":0}`, "{\n  \"n\": 0\n}\n"},
		{"table empty body", outfmt.Table, "", "ID  NAME\n"},
		{"table missing key", outfmt.Table, `{"n":0}`, "ID  NAME\n"},
		{"plain empty body", outfmt.Plain, "", ""},
		{"plain header empty body", outfmt.PlainHeader, "", "ID\tNAME\n"},
		{"table with rows", outfmt.Table, `{"things":[{"id":1,"name":"a"}]}`, "ID  NAME\n1   a\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			g := testGlobals(t, &buf, tc.mode, "test@example.com")
			g.Stdout = &buf
			if err := l.print(g, json.RawMessage(tc.data)); err != nil {
				t.Fatalf("print: %v", err)
			}
			if buf.String() != tc.want {
				t.Errorf("stdout %q, want %q", buf.String(), tc.want)
			}
		})
	}
}

func TestListPrintBadAnswer(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.Table, "test@example.com")
	g.Stdout = &buf
	err := list{noun: "things", rows: func([]map[string]any) [][]string { return nil }}.print(g, json.RawMessage(`[1]`))
	if err == nil || !strings.HasPrefix(err.Error(), "parse things: ") {
		t.Fatalf("err = %v, want a parse things error", err)
	}
	if buf.Len() != 0 {
		t.Errorf("stdout %q, want empty", buf.String())
	}
}

func TestWriteObject(t *testing.T) {
	for _, mode := range []outfmt.Mode{outfmt.Table, outfmt.Plain, outfmt.JSON} {
		var buf bytes.Buffer
		g := testGlobals(t, &buf, mode, "test@example.com")
		g.Stdout = &buf
		if err := writeObject(g, nil); err != nil {
			t.Fatalf("mode %d: %v", mode, err)
		}
		if buf.String() != "{}\n" {
			t.Errorf("mode %d: empty answer printed %q, want {}", mode, buf.String())
		}
		buf.Reset()
		if err := writeObject(g, json.RawMessage(`{"a":[1]}`)); err != nil {
			t.Fatalf("mode %d: %v", mode, err)
		}
		if buf.String() != "{\n  \"a\": [\n    1\n  ]\n}\n" {
			t.Errorf("mode %d: printed %q", mode, buf.String())
		}
	}
}
