package ui

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	for _, mode := range []string{"auto", "always", "never", ""} {
		u := New(mode)
		if u == nil {
			t.Errorf("New(%q) returned nil", mode)
		}
	}
}

func TestSuccessf(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, "never")
	u.Successf("hello %s", "world")
	got := buf.String()
	if !strings.Contains(got, "hello world") {
		t.Errorf("Successf output = %q, want to contain %q", got, "hello world")
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, "never")
	u.Error(errors.New("something failed"))
	got := buf.String()
	if !strings.Contains(got, "something failed") {
		t.Errorf("Error output = %q, want to contain %q", got, "something failed")
	}
}

func TestWarnf(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, "never")
	u.Warnf("warning: %d issues", 5)
	got := buf.String()
	if !strings.Contains(got, "warning: 5 issues") {
		t.Errorf("Warnf output = %q, want to contain %q", got, "warning: 5 issues")
	}
}

func TestInfof(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, "never")
	u.Infof("info: %s", "done")
	got := buf.String()
	if !strings.Contains(got, "info: done") {
		t.Errorf("Infof output = %q, want to contain %q", got, "info: done")
	}
}

func TestColorModeAlways(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, "always")
	u.Successf("colored")
	got := buf.String()
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("always mode output = %q, want ANSI escape codes", got)
	}
	if !strings.Contains(got, "colored") {
		t.Errorf("always mode output = %q, want to contain %q", got, "colored")
	}
}

func TestColorModeNever(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, "never")
	u.Successf("plain")
	got := buf.String()
	if strings.Contains(got, "\x1b[") {
		t.Errorf("never mode output = %q, want no ANSI escape codes", got)
	}
}

func TestAllMethodsWithColor(t *testing.T) {
	var buf bytes.Buffer
	u := NewWithWriter(&buf, "always")

	u.Successf("ok")
	u.Error(errors.New("fail"))
	u.Warnf("warn")
	u.Infof("note")

	got := buf.String()
	for _, want := range []string{"ok", "fail", "warn", "note"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q in %q", want, got)
		}
	}
}

func TestContext(t *testing.T) {
	u := New("never")
	ctx := NewContext(context.Background(), u)
	got := FromContext(ctx)
	if got != u {
		t.Error("FromContext did not return the stored UI")
	}
}

func TestFromContextDefault(t *testing.T) {
	got := FromContext(context.Background())
	if got == nil {
		t.Error("FromContext with no UI returned nil")
	}
}
