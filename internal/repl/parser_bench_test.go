package repl

import "testing"

// BenchmarkParseSimple covers the common case — a top-level command followed
// by a sub-command (e.g. `numbers list`). This is the hot path for every
// REPL line and every `-x '<cmd>'` invocation.
func BenchmarkParseSimple(b *testing.B) {
	const line = "numbers list"
	b.ReportAllocs()
	for range b.N {
		if _, err := Parse(line); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseQuoted exercises the tokenizer's quote-handling path —
// strings with embedded whitespace, the slowest tokenization case.
func BenchmarkParseQuoted(b *testing.B) {
	const line = `login 1000000001 "hunter two"`
	b.ReportAllocs()
	for range b.N {
		if _, err := Parse(line); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseJSONBody covers the worst-case payload — a typical
// `account update` invocation with a JSON object body that the
// tokenizer splits-on-whitespace-outside-quotes through.
func BenchmarkParseJSONBody(b *testing.B) {
	const line = `account update {"timezone":"America/Chicago","name":"Acme Co","callerId":"2015551234"}`
	b.ReportAllocs()
	for range b.N {
		if _, err := Parse(line); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTailFrom isolates the rejoin path used by commands that consume
// the unparsed tail (e.g. `account update <json>`).
func BenchmarkTailFrom(b *testing.B) {
	p, err := Parse(`account update {"timezone":"America/Chicago","name":"Acme Co"}`)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = p.TailFrom(2)
	}
}
