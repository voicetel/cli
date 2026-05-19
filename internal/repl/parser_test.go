package repl

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseEmpty(t *testing.T) {
	for _, line := range []string{"", "   ", "# a comment", "  # also a comment"} {
		_, err := Parse(line)
		if !errors.Is(err, ErrEmpty) {
			t.Errorf("Parse(%q): expected ErrEmpty, got %v", line, err)
		}
	}
}

func TestParseSimple(t *testing.T) {
	p, err := Parse("numbers list")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Tokens, []string{"numbers", "list"}) {
		t.Errorf("tokens = %#v", p.Tokens)
	}
}

func TestParseQuoted(t *testing.T) {
	p, err := Parse(`account update {"timezone":"America/Chicago","name":"Acme Co"}`)
	if err != nil {
		t.Fatal(err)
	}
	// JSON body is one token (a brace-block). Our tokenizer doesn't
	// special-case braces — it just splits on whitespace outside quotes.
	// So "account update" is the first two tokens, then the body is split.
	if len(p.Tokens) < 3 {
		t.Fatalf("expected at least 3 tokens, got %#v", p.Tokens)
	}
	if p.Tokens[0] != "account" || p.Tokens[1] != "update" {
		t.Errorf("first two tokens = %v", p.Tokens[:2])
	}
}

func TestParseQuotedString(t *testing.T) {
	p, err := Parse(`login 100 "hunter two"`)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p.Tokens, []string{"login", "100", "hunter two"}) {
		t.Errorf("tokens = %#v", p.Tokens)
	}
}

func TestParseUnterminatedQuote(t *testing.T) {
	if _, err := Parse(`login 100 "oops`); err == nil {
		t.Fatal("expected error for unterminated quote")
	}
}

func TestTailFrom(t *testing.T) {
	p, err := Parse(`account update {"timezone":"America/Chicago"}`)
	if err != nil {
		t.Fatal(err)
	}
	tail := p.TailFrom(2)
	expect := `{"timezone":"America/Chicago"}`
	if tail != expect {
		t.Errorf("tail = %q, want %q", tail, expect)
	}
}

func TestTailFromZero(t *testing.T) {
	p, err := Parse(`hello world`)
	if err != nil {
		t.Fatal(err)
	}
	if got := p.TailFrom(0); got != "hello world" {
		t.Errorf("TailFrom(0) = %q", got)
	}
}

func TestFlagSetBasics(t *testing.T) {
	fs := NewFlagSet()
	state := fs.RegisterString("state")
	verbose := fs.RegisterBool("verbose")
	if err := fs.Parse([]string{"--state=NJ", "--verbose", "extra"}); err != nil {
		t.Fatal(err)
	}
	if *state != "NJ" {
		t.Errorf("state = %q", *state)
	}
	if !*verbose {
		t.Error("verbose not set")
	}
	if !reflect.DeepEqual(fs.Positional, []string{"extra"}) {
		t.Errorf("positional = %#v", fs.Positional)
	}
}

func TestFlagSetSeparateValue(t *testing.T) {
	fs := NewFlagSet()
	npa := fs.RegisterString("npa")
	if err := fs.Parse([]string{"--npa", "201"}); err != nil {
		t.Fatal(err)
	}
	if *npa != "201" {
		t.Errorf("npa = %q", *npa)
	}
}

func TestFlagSetMissingValue(t *testing.T) {
	fs := NewFlagSet()
	fs.RegisterString("npa")
	if err := fs.Parse([]string{"--npa"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestFlagSetUnknown(t *testing.T) {
	fs := NewFlagSet()
	if err := fs.Parse([]string{"--mystery"}); err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestFlagSetBoolWithValue(t *testing.T) {
	fs := NewFlagSet()
	b := fs.RegisterBool("force")
	if err := fs.Parse([]string{"--force=false"}); err != nil {
		t.Fatal(err)
	}
	if *b {
		t.Error("force should be false")
	}
	if err := fs.Parse([]string{"--force=garbage"}); err == nil {
		t.Fatal("expected error on bad bool value")
	}
}
