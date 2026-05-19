package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestJSONPrettyPrint(t *testing.T) {
	out := &bytes.Buffer{}
	err := &bytes.Buffer{}
	p := NewWithColor(out, err, false)
	if err := p.JSON(map[string]any{"a": 1, "b": "x"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "\"a\": 1") {
		t.Errorf("expected indented JSON, got %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("expected trailing newline, got %q", got)
	}
}

func TestJSONNilSkipped(t *testing.T) {
	out := &bytes.Buffer{}
	p := NewWithColor(out, &bytes.Buffer{}, false)
	if err := p.JSON(nil); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("expected nothing, got %q", out.String())
	}
}

func TestErrorNoColor(t *testing.T) {
	errBuf := &bytes.Buffer{}
	p := NewWithColor(&bytes.Buffer{}, errBuf, false)
	p.Error(errors.New("boom"))
	got := errBuf.String()
	if !strings.HasPrefix(got, "Error: boom") {
		t.Errorf("got %q", got)
	}
	if strings.Contains(got, "\x1b[") {
		t.Errorf("unexpected ANSI codes in non-tty output: %q", got)
	}
}

func TestErrorWithColor(t *testing.T) {
	errBuf := &bytes.Buffer{}
	p := NewWithColor(&bytes.Buffer{}, errBuf, true)
	p.Error(errors.New("oops"))
	got := errBuf.String()
	if !strings.Contains(got, "\x1b[31m") {
		t.Errorf("expected red ANSI code, got %q", got)
	}
	if !strings.Contains(got, "oops") {
		t.Errorf("expected message, got %q", got)
	}
}

func TestErrorNilIsNoop(t *testing.T) {
	errBuf := &bytes.Buffer{}
	p := NewWithColor(&bytes.Buffer{}, errBuf, true)
	p.Error(nil)
	if errBuf.Len() != 0 {
		t.Errorf("expected nothing on nil err, got %q", errBuf.String())
	}
}

func TestErrorf(t *testing.T) {
	errBuf := &bytes.Buffer{}
	p := NewWithColor(&bytes.Buffer{}, errBuf, false)
	p.Errorf("bad %s", "value")
	if !strings.Contains(errBuf.String(), "bad value") {
		t.Errorf("got %q", errBuf.String())
	}
}

func TestPrintln(t *testing.T) {
	out := &bytes.Buffer{}
	p := NewWithColor(out, &bytes.Buffer{}, false)
	p.Println("hi")
	if out.String() != "hi\n" {
		t.Errorf("got %q", out.String())
	}
}

func TestPrintf(t *testing.T) {
	out := &bytes.Buffer{}
	p := NewWithColor(out, &bytes.Buffer{}, false)
	p.Printf("%s=%d", "x", 1)
	if out.String() != "x=1" {
		t.Errorf("got %q", out.String())
	}
}

func TestNewDetectsNonTTY(t *testing.T) {
	out := &bytes.Buffer{}
	p := New(out, out)
	if p.Color() {
		t.Error("expected non-TTY buffer to be non-colored")
	}
}
