package output

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
)

// Stdout and Stderr expose the underlying writers; the test just asserts
// the pointer identity round-trips.
func TestStdoutStderrAccessors(t *testing.T) {
	var out, err bytes.Buffer
	p := New(&out, &err)
	if p.Stdout() != &out {
		t.Error("Stdout() did not return the constructor's out writer")
	}
	if p.Stderr() != &err {
		t.Error("Stderr() did not return the constructor's err writer")
	}
}

// JSON of nil is a no-op (returns nil, writes nothing).
func TestJSONNilSilent(t *testing.T) {
	var out bytes.Buffer
	p := New(&out, io.Discard)
	if err := p.JSON(nil); err != nil {
		t.Errorf("JSON(nil): want nil error, got %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("JSON(nil): expected no output, got %q", out.String())
	}
}

// JSON returns a wrapped error when MarshalIndent fails. Channels are
// unmarshalable in encoding/json (the canonical "this can't be JSON"
// value).
func TestJSONMarshalError(t *testing.T) {
	p := New(io.Discard, io.Discard)
	err := p.JSON(make(chan int))
	if err == nil {
		t.Fatal("JSON(chan int): want error, got nil")
	}
	if msg := err.Error(); msg[:7] != "output:" {
		t.Errorf("expected wrapped 'output:' prefix, got %q", msg)
	}
}

// failingWriter always fails. Used to exercise the write-error path in JSON
// (fmt.Fprintln failure → returned error wrapped with `output: write`).
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("synthetic write fail")
}

func TestJSONWriteError(t *testing.T) {
	p := New(failingWriter{}, io.Discard)
	if err := p.JSON(map[string]int{"k": 1}); err == nil || err.Error()[:14] != "output: write:" {
		t.Errorf("expected wrapped 'output: write:' error, got %v", err)
	}
}

// Color: when Error writer is a TTY → true; when it's a buffer → false.
// We can directly inspect the path through New / NewWithColor.
func TestNewWithColorOverride(t *testing.T) {
	var b bytes.Buffer
	if p := NewWithColor(&b, &b, true); !p.Color() {
		t.Error("NewWithColor(true): want Color=true")
	}
	if p := NewWithColor(&b, &b, false); p.Color() {
		t.Error("NewWithColor(false): want Color=false")
	}
}

// isTerminal: non-*os.File → false; *os.File at /dev/null → false; *os.File
// at a real TTY → true (only if we can find one).
func TestIsTerminalNonFile(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Error("isTerminal(*bytes.Buffer): want false")
	}
}

func TestIsTerminalDevNull(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Skipf("can't open /dev/null: %v", err)
	}
	defer func() { _ = f.Close() }()
	if isTerminal(f) {
		t.Error("isTerminal(/dev/null): want false (not a TTY)")
	}
}

// Error with a TTY-colour-enabled Printer wraps the message in ANSI codes.
func TestErrorColorisedPath(t *testing.T) {
	var errBuf bytes.Buffer
	p := NewWithColor(io.Discard, &errBuf, true)
	p.Error(errors.New("boom"))
	out := errBuf.String()
	if !contains(out, "\x1b[31m") || !contains(out, "\x1b[1m") || !contains(out, "\x1b[0m") {
		t.Errorf("expected ANSI codes in colourised output, got %q", out)
	}
}

// Error(nil) is a no-op — no output, no panic.
func TestErrorNilSilent(t *testing.T) {
	var errBuf bytes.Buffer
	p := New(io.Discard, &errBuf)
	p.Error(nil)
	if errBuf.Len() != 0 {
		t.Errorf("Error(nil): expected no output, got %q", errBuf.String())
	}
}

// Println + Printf cover the success-write helpers.
func TestPrintlnAndPrintf(t *testing.T) {
	var out bytes.Buffer
	p := New(&out, io.Discard)
	p.Println("hello")
	p.Printf("world=%d\n", 42)
	got := out.String()
	if got != "hello\nworld=42\n" {
		t.Errorf("unexpected output: %q", got)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
