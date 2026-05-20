package repl

import "testing"

// TailFrom: every path through the quote-handling state machine. Walks
// well-formed lines with single quotes, double quotes, and tabs as
// separators; out-of-range n values; consecutive whitespace.
func TestTailFromExhaustive(t *testing.T) {
	cases := []struct {
		name, raw string
		n         int
		want      string
	}{
		{"empty raw", "", 0, ""},
		{"n<0 returns full raw", "alpha bravo", -1, "alpha bravo"},
		{"single token n=1 empty tail", "alpha", 1, ""},
		{"three tokens n=2 single tail", "a b c", 2, "c"},
		{"double quoted preserves spaces", `cmd "a b" rest`, 2, "rest"},
		{"single quoted preserves spaces", `cmd 'a b' rest`, 2, "rest"},
		{"unmatched quote consumes to eol", `cmd "no close`, 2, ""},
		{"tabs as separators", "a\tb\tc", 1, "b\tc"},
		{"mixed whitespace runs", "a   b   c", 2, "c"},
		{"json body preserved as tail", `account update {"k":"v"}`, 2, `{"k":"v"}`},
		{"json body with internal spaces", `acc update {"a": "b c"}`, 2, `{"a": "b c"}`},
		{"n equal to token count yields empty", "alpha bravo charlie", 3, ""},
		{"n past token count yields empty", "alpha bravo", 50, ""},
		{"leading whitespace before tail", "   alpha   bravo", 1, "bravo"},
		{"quote followed by whitespace", `cmd "a"   tail`, 2, "tail"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &Parsed{Raw: tc.raw}
			got := p.TailFrom(tc.n)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// FlagSet.Parse: a few more paths to push from 96.9% → 100%.
func TestFlagSetParseEdges(t *testing.T) {
	// Bool flag with value=true literal works.
	fs := NewFlagSet()
	b := fs.RegisterBool("force")
	if err := fs.Parse([]string{"--force=true"}); err != nil {
		t.Fatal(err)
	}
	if !*b {
		t.Error("--force=true didn't set bool")
	}

	// Positional args accumulate in order.
	fs = NewFlagSet()
	if err := fs.Parse([]string{"alpha", "bravo", "charlie"}); err != nil {
		t.Fatal(err)
	}
	if len(fs.Positional) != 3 || fs.Positional[0] != "alpha" || fs.Positional[2] != "charlie" {
		t.Errorf("positional = %v", fs.Positional)
	}

	// Bool flag without value defaults to true.
	fs = NewFlagSet()
	b = fs.RegisterBool("verbose")
	if err := fs.Parse([]string{"--verbose"}); err != nil {
		t.Fatal(err)
	}
	if !*b {
		t.Error("--verbose alone didn't set bool to true")
	}

	// Flag immediately followed by non-flag positional.
	fs = NewFlagSet()
	s := fs.RegisterString("name")
	if err := fs.Parse([]string{"--name=Alice", "extra1", "extra2"}); err != nil {
		t.Fatal(err)
	}
	if *s != "Alice" {
		t.Errorf("name = %q", *s)
	}
	if len(fs.Positional) != 2 {
		t.Errorf("positional = %v", fs.Positional)
	}
}
