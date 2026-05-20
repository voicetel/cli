package output

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// JSON output via Printer with color enabled embeds the cyan/green/yellow/
// magenta ANSI codes for keys / strings / numbers / literals respectively.
func TestJSONColorEnabledEmitsANSI(t *testing.T) {
	var out bytes.Buffer
	p := NewWithColor(&out, io.Discard, true)
	if err := p.JSON(map[string]any{
		"name":    "Acme",
		"count":   42,
		"active":  true,
		"deleted": nil,
		"price":   1.5,
	}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for label, want := range map[string]string{
		"key (cyan)":        ansiKey,
		"string (green)":    ansiString,
		"number (yellow)":   ansiNumber,
		"literal (magenta)": ansiLiteral,
		"reset":             ansiReset,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s code in output: %q", label, got)
		}
	}
}

// JSON output via Printer with color disabled emits NO ANSI codes.
func TestJSONColorDisabledNoANSI(t *testing.T) {
	var out bytes.Buffer
	p := NewWithColor(&out, io.Discard, false)
	if err := p.JSON(map[string]any{"name": "Acme"}); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); strings.Contains(got, "\x1b[") {
		t.Errorf("expected no ANSI codes, got %q", got)
	}
}

// colorizeJSON exhaustively: every supported token type, plus arrays and
// nested objects. Asserts both that the right colour is applied AND that
// the original characters are preserved.
func TestColorizeJSONAllTokenTypes(t *testing.T) {
	in := []byte(`{
  "name": "Acme",
  "count": 42,
  "rate": -3.14e10,
  "active": true,
  "muted": false,
  "deleted": null,
  "tags": ["a", "b"]
}`)
	got := colorizeJSON(in)

	cases := []struct {
		piece string
		color string
		why   string
	}{
		{`"name"`, ansiKey, "object key"},
		{`"Acme"`, ansiString, "string value"},
		{`42`, ansiNumber, "integer"},
		{`-3.14e10`, ansiNumber, "negative scientific notation"},
		{`true`, ansiLiteral, "literal true"},
		{`false`, ansiLiteral, "literal false"},
		{`null`, ansiLiteral, "literal null"},
		{`"a"`, ansiString, "array string element"},
	}
	for _, tc := range cases {
		want := tc.color + tc.piece + ansiReset
		if !strings.Contains(got, want) {
			t.Errorf("%s: expected coloured token %q in output", tc.why, want)
		}
	}
	// Structural characters (braces, brackets, commas, colons) must survive.
	for _, want := range []string{"{", "}", "[", "]", ":", ","} {
		if !strings.Contains(got, want) {
			t.Errorf("structural character %q missing from output", want)
		}
	}
}

// Backslash-escaped quotes inside a string must NOT terminate the string
// scanner early — the colorizer should treat \" as part of the string.
func TestColorizeJSONEscapedQuotes(t *testing.T) {
	in := []byte(`{"key": "value with \"escaped\" quotes"}`)
	got := colorizeJSON(in)
	want := ansiString + `"value with \"escaped\" quotes"` + ansiReset
	if !strings.Contains(got, want) {
		t.Errorf("escaped-quote string didn't round-trip; got %q", got)
	}
}

// Strings that happen to contain `true`, `null`, etc. must not get those
// keywords highlighted. The whole string must be wrapped exactly once.
func TestColorizeJSONStringContainingLiterals(t *testing.T) {
	in := []byte(`{"key": "the answer is true, not null"}`)
	got := colorizeJSON(in)
	wantStr := ansiString + `"the answer is true, not null"` + ansiReset
	if !strings.Contains(got, wantStr) {
		t.Errorf("string containing keywords wasn't a single coloured token: %q", got)
	}
	// And confirm `true`/`null` substrings did NOT get magenta wraps.
	if strings.Contains(got, ansiLiteral+"true") {
		t.Error("colorizer wrapped 'true' inside a string with literal colour")
	}
	if strings.Contains(got, ansiLiteral+"null") {
		t.Error("colorizer wrapped 'null' inside a string with literal colour")
	}
}

// Numbers at end-of-input (no trailing whitespace or comma) must still be
// fully consumed.
func TestColorizeJSONNumberAtEnd(t *testing.T) {
	in := []byte(`42`)
	got := colorizeJSON(in)
	want := ansiNumber + "42" + ansiReset
	if got != want {
		t.Errorf("number-at-end: got %q, want %q", got, want)
	}
}

// hasPrefix is internal; cover the out-of-bounds branch and the mismatch
// branch directly.
func TestHasPrefix(t *testing.T) {
	if hasPrefix([]byte("abc"), 0, "abcd") {
		t.Error("hasPrefix should reject when needle is longer than remainder")
	}
	if hasPrefix([]byte("abcd"), 0, "abe") {
		t.Error("hasPrefix should reject on character mismatch")
	}
	if !hasPrefix([]byte("abcd"), 1, "bc") {
		t.Error("hasPrefix should match at non-zero offset")
	}
}

// Verify a deeply nested structure still colourises cleanly — protects
// against off-by-one bugs in the state machine.
func TestColorizeJSONNested(t *testing.T) {
	in := []byte(`{
  "outer": {
    "inner": {
      "leaf": "value"
    }
  }
}`)
	got := colorizeJSON(in)
	if !strings.Contains(got, ansiKey+`"outer"`+ansiReset) ||
		!strings.Contains(got, ansiKey+`"inner"`+ansiReset) ||
		!strings.Contains(got, ansiKey+`"leaf"`+ansiReset) ||
		!strings.Contains(got, ansiString+`"value"`+ansiReset) {
		t.Errorf("nested keys/values not coloured correctly: %q", got)
	}
}
