package outfmt

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestModeConstants(t *testing.T) {
	if Table != 0 {
		t.Errorf("Table = %d, want 0", Table)
	}
	if JSON != 1 {
		t.Errorf("JSON = %d, want 1", JSON)
	}
	if Plain != 2 {
		t.Errorf("Plain = %d, want 2", Plain)
	}
}

func TestNewContext(t *testing.T) {
	ctx := NewContext(context.Background(), JSON)
	got := ModeFromContext(ctx)
	if got != JSON {
		t.Errorf("ModeFromContext = %d, want JSON (%d)", got, JSON)
	}
}

func TestModeFromContextDefault(t *testing.T) {
	got := ModeFromContext(context.Background())
	if got != Table {
		t.Errorf("ModeFromContext(empty) = %d, want Table (%d)", got, Table)
	}
}

func TestIsJSON(t *testing.T) {
	ctx := NewContext(context.Background(), JSON)
	if !IsJSON(ctx) {
		t.Error("IsJSON = false, want true")
	}
	if IsJSON(context.Background()) {
		t.Error("IsJSON(empty ctx) = true, want false")
	}
}

func TestIsPlain(t *testing.T) {
	ctx := NewContext(context.Background(), Plain)
	if !IsPlain(ctx) {
		t.Error("IsPlain = false, want true")
	}
	if IsPlain(context.Background()) {
		t.Error("IsPlain(empty ctx) = true, want false")
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"name":  "test",
		"count": 42,
		"url":   "https://example.com/?q=1&r=2",
	}
	if err := WriteJSON(&buf, data); err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}
	got := buf.String()

	// Verify pretty-printed.
	if !strings.Contains(got, "  ") {
		t.Error("WriteJSON output not indented")
	}

	// Verify no HTML escaping (& should not become \u0026).
	if strings.Contains(got, `\u0026`) {
		t.Error("WriteJSON escaped HTML characters")
	}
	if !strings.Contains(got, "&") {
		t.Error("WriteJSON missing unescaped &")
	}

	// Verify valid JSON.
	var parsed map[string]any
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Errorf("WriteJSON produced invalid JSON: %v", err)
	}
}

func TestWriteTable(t *testing.T) {
	var buf bytes.Buffer
	header := []string{"ID", "NAME", "STATUS"}
	rows := [][]string{
		{"1", "alpha", "ok"},
		{"2", "beta", "fail"},
	}
	if err := WriteTable(&buf, header, rows); err != nil {
		t.Fatalf("WriteTable error: %v", err)
	}
	got := buf.String()

	// Header present.
	if !strings.Contains(got, "ID") || !strings.Contains(got, "NAME") || !strings.Contains(got, "STATUS") {
		t.Errorf("WriteTable missing header columns in: %q", got)
	}

	// Data rows present.
	if !strings.Contains(got, "alpha") || !strings.Contains(got, "beta") {
		t.Errorf("WriteTable missing data in: %q", got)
	}

	// Aligned with spaces (tabwriter uses spaces for padding).
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Errorf("WriteTable produced %d lines, want 3", len(lines))
	}
}

func TestWriteTableNoHeader(t *testing.T) {
	var buf bytes.Buffer
	rows := [][]string{
		{"a", "b"},
		{"c", "d"},
	}
	if err := WriteTable(&buf, nil, rows); err != nil {
		t.Fatalf("WriteTable error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("WriteTable with no header produced %d lines, want 2", len(lines))
	}
}

func TestWritePlain(t *testing.T) {
	var buf bytes.Buffer
	rows := [][]string{
		{"1", "alpha", "ok"},
		{"2", "beta", "fail"},
	}
	if err := WritePlain(&buf, rows); err != nil {
		t.Fatalf("WritePlain error: %v", err)
	}
	got := buf.String()
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Errorf("WritePlain produced %d lines, want 2", len(lines))
	}

	// Columns separated by tabs.
	if !strings.Contains(lines[0], "\t") {
		t.Errorf("WritePlain line not tab-separated: %q", lines[0])
	}
	parts := strings.Split(lines[0], "\t")
	if len(parts) != 3 {
		t.Errorf("WritePlain line has %d tab-separated parts, want 3", len(parts))
	}
	if parts[0] != "1" || parts[1] != "alpha" || parts[2] != "ok" {
		t.Errorf("WritePlain line = %v, want [1 alpha ok]", parts)
	}
}

func TestWritePlainEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WritePlain(&buf, nil); err != nil {
		t.Fatalf("WritePlain error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("WritePlain with no rows produced output: %q", buf.String())
	}
}

func TestNewTabWriter(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTabWriter(&buf)
	if tw == nil {
		t.Fatal("NewTabWriter returned nil")
	}
}

func TestContextRoundTrip(t *testing.T) {
	for _, mode := range []Mode{Table, JSON, Plain} {
		ctx := NewContext(context.Background(), mode)
		got := ModeFromContext(ctx)
		if got != mode {
			t.Errorf("round-trip mode %d: got %d", mode, got)
		}
	}
}

// captureStdout redirects os.Stdout to a pipe, runs fn, and returns the
// captured output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	origStdout := os.Stdout
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func TestWrite_JSON(t *testing.T) {
	ctx := NewContext(context.Background(), JSON)
	data := map[string]string{"key": "value"}

	got := captureStdout(t, func() {
		if err := Write(ctx, data, nil, nil); err != nil {
			t.Fatalf("Write JSON: %v", err)
		}
	})

	if !strings.Contains(got, `"key"`) || !strings.Contains(got, `"value"`) {
		t.Errorf("Write JSON output = %q, want key/value pair", got)
	}
}

func TestWrite_Plain(t *testing.T) {
	ctx := NewContext(context.Background(), Plain)
	rows := [][]string{{"a", "b"}, {"c", "d"}}

	got := captureStdout(t, func() {
		if err := Write(ctx, nil, nil, rows); err != nil {
			t.Fatalf("Write Plain: %v", err)
		}
	})

	if !strings.Contains(got, "a\tb") {
		t.Errorf("Write Plain output = %q, want tab-separated", got)
	}
}

func TestWrite_Table(t *testing.T) {
	ctx := NewContext(context.Background(), Table)
	header := []string{"COL1", "COL2"}
	rows := [][]string{{"x", "y"}}

	got := captureStdout(t, func() {
		if err := Write(ctx, nil, header, rows); err != nil {
			t.Fatalf("Write Table: %v", err)
		}
	})

	if !strings.Contains(got, "COL1") || !strings.Contains(got, "x") {
		t.Errorf("Write Table output = %q, want header and data", got)
	}
}

func TestWriteTable_SingleColumn(t *testing.T) {
	var buf bytes.Buffer
	header := []string{"NAME"}
	rows := [][]string{{"alice"}, {"bob"}}
	if err := WriteTable(&buf, header, rows); err != nil {
		t.Fatalf("WriteTable error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "NAME") || !strings.Contains(got, "alice") {
		t.Errorf("WriteTable single column = %q, want NAME and alice", got)
	}
}

func TestWritePlain_SingleColumn(t *testing.T) {
	var buf bytes.Buffer
	rows := [][]string{{"one"}, {"two"}}
	if err := WritePlain(&buf, rows); err != nil {
		t.Fatalf("WritePlain error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("WritePlain single column produced %d lines, want 2", len(lines))
	}
	if lines[0] != "one" {
		t.Errorf("first line = %q, want %q", lines[0], "one")
	}
}

func TestWriteTable_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTable(&buf, []string{"H"}, nil); err != nil {
		t.Fatalf("WriteTable error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "H") {
		t.Errorf("WriteTable empty rows = %q, want header", got)
	}
}
