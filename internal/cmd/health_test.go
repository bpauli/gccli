package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/outfmt"
)

func overrideNowFn(t *testing.T, fixed time.Time) {
	t.Helper()
	orig := nowFn
	nowFn = func() time.Time { return fixed }
	t.Cleanup(func() { nowFn = orig })
}

func TestResolveDate(t *testing.T) {
	fixed := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	overrideNowFn(t, fixed)

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty defaults to today", "", "2024-06-15", false},
		{"today", "today", "2024-06-15", false},
		{"Today (case insensitive)", "Today", "2024-06-15", false},
		{"yesterday", "yesterday", "2024-06-14", false},
		{"Yesterday (case insensitive)", "Yesterday", "2024-06-14", false},
		{"3d relative", "3d", "2024-06-12", false},
		{"7d relative", "7d", "2024-06-08", false},
		{"1d relative", "1d", "2024-06-14", false},
		{"0d relative", "0d", "2024-06-15", false},
		{"+7d future", "+7d", "2024-06-22", false},
		{"+365d future", "+365d", "2025-06-15", false},
		{"+0d future", "+0d", "2024-06-15", false},
		{"explicit date", "2024-01-01", "2024-01-01", false},
		{"whitespace trimmed", "  2024-06-15  ", "2024-06-15", false},
		{"invalid date", "not-a-date", "", true},
		{"invalid format", "15/06/2024", "", true},
		{"partial date", "2024-06", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveDate(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveDate(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveDate_ErrorMessage(t *testing.T) {
	_, err := resolveDate("bad-date")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Errorf("expected error to mention YYYY-MM-DD format, got: %v", err)
	}
	if !strings.Contains(err.Error(), "3d") {
		t.Errorf("expected error to mention relative date example, got: %v", err)
	}
}

func TestWriteHealthJSON_NilData(t *testing.T) {
	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	err := writeHealthJSON(g, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
