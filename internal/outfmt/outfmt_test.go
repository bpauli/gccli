package outfmt

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestContextRoundTrip(t *testing.T) {
	for _, mode := range []Mode{Table, JSON, Plain} {
		ctx := NewContext(context.Background(), mode)
		got := ModeFromContext(ctx)
		if got != mode {
			t.Errorf("round-trip mode %d: got %d", mode, got)
		}
	}
}

func writeRows(t *testing.T, mode Mode, header []string, rows [][]string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := WriteRows(NewContext(context.Background(), mode), &buf, header, rows); err != nil {
		t.Fatalf("WriteRows error: %v", err)
	}
	return buf.String()
}

func TestWriteRows(t *testing.T) {
	header := []string{"ID", "NAME", "STATUS"}
	rows := [][]string{
		{"1", "alpha", "ok"},
		{"22", "beta", "fail"},
	}
	tests := []struct {
		mode   Mode
		header []string
		rows   [][]string
		want   string
	}{
		{Table, header, rows, "ID  NAME   STATUS\n1   alpha  ok\n22  beta   fail\n"},
		{Table, nil, rows, "1   alpha  ok\n22  beta   fail\n"},
		{Table, header, nil, "ID  NAME  STATUS\n"},
		{Plain, header, rows, "1\talpha\tok\n22\tbeta\tfail\n"},
		{Plain, header, nil, ""},
	}
	for _, tt := range tests {
		if got := writeRows(t, tt.mode, tt.header, tt.rows); got != tt.want {
			t.Errorf("WriteRows(mode %d, header %q, rows %q) = %q, want %q", tt.mode, tt.header, tt.rows, got, tt.want)
		}
	}
}
