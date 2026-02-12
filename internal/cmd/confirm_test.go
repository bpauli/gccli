package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfirm_Force(t *testing.T) {
	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", true)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected true when force=true")
	}
	if buf.String() != "" {
		t.Fatalf("expected no prompt output with force, got: %q", buf.String())
	}
}

func TestConfirm_Yes(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("y\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected true for 'y' input")
	}
	if !strings.Contains(buf.String(), "Delete?") {
		t.Fatalf("expected prompt in output, got: %q", buf.String())
	}
}

func TestConfirm_YesFull(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("yes\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected true for 'yes' input")
	}
}

func TestConfirm_No(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("n\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ok {
		t.Fatal("expected false for 'n' input")
	}
}

func TestConfirm_EmptyInput(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ok {
		t.Fatal("expected false for empty input (default is no)")
	}
}

func TestConfirm_EOF(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ok {
		t.Fatal("expected false for EOF")
	}
}

func TestConfirm_CaseInsensitive(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("Y\n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected true for 'Y' input (case insensitive)")
	}
}

func TestConfirm_WhitespaceTrimmed(t *testing.T) {
	orig := confirmReader
	confirmReader = strings.NewReader("  yes  \n")
	t.Cleanup(func() { confirmReader = orig })

	var buf bytes.Buffer
	ok, err := confirm(&buf, "Delete?", false)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected true for '  yes  ' input (trimmed)")
	}
}
