package repl

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Parsed represents a tokenised REPL line.
//
// Tokens are split on whitespace except where escaped by single or
// double quotes. Quotes are stripped. We intentionally do NOT handle
// shell-style escapes ($VARS, backticks, etc.) — this is an interactive
// REPL, not a shell.
//
// JSONTail is the unparsed remainder of the line, used by commands that
// accept a JSON body argument. The convention is: the first N tokens
// (the verb chain) are consumed by the dispatcher; the rest of the line,
// untrimmed of internal whitespace, is offered as JSONTail.
type Parsed struct {
	Tokens []string
	Raw    string
}

// ErrEmpty is returned by Parse when the line contains only whitespace
// or is a comment (starts with `#`).
var ErrEmpty = errors.New("repl: empty line")

// Parse tokenises a single REPL input line.
func Parse(line string) (*Parsed, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil, ErrEmpty
	}
	tokens, err := tokenize(trimmed)
	if err != nil {
		return nil, err
	}
	return &Parsed{Tokens: tokens, Raw: trimmed}, nil
}

// TailFrom returns the substring of Raw after the first n tokens. Used
// for commands that take a free-form JSON body — we don't want to lose
// internal whitespace inside the body just because we tokenized it.
func (p *Parsed) TailFrom(n int) string {
	if n <= 0 {
		return p.Raw
	}
	// Walk Raw character-by-character, skipping n whitespace-separated
	// fields, then return what's left.
	s := p.Raw
	i := 0
	fields := 0
	for i < len(s) && fields < n {
		// Skip leading whitespace.
		for i < len(s) && unicode.IsSpace(rune(s[i])) {
			i++
		}
		if i >= len(s) {
			break
		}
		// Consume one token, respecting quotes.
		inQ := byte(0)
		for i < len(s) {
			ch := s[i]
			if inQ != 0 {
				if ch == inQ {
					inQ = 0
				}
				i++
				continue
			}
			if ch == '"' || ch == '\'' {
				inQ = ch
				i++
				continue
			}
			if unicode.IsSpace(rune(ch)) {
				break
			}
			i++
		}
		fields++
	}
	// Skip trailing whitespace before the body.
	for i < len(s) && unicode.IsSpace(rune(s[i])) {
		i++
	}
	return s[i:]
}

// tokenize splits a line into whitespace-separated fields, honouring
// single and double quotes.
func tokenize(line string) ([]string, error) {
	var (
		out   []string
		cur   strings.Builder
		inQ   byte
		piece bool
	)
	for i := 0; i < len(line); i++ {
		ch := line[i]
		switch {
		case inQ != 0:
			if ch == inQ {
				inQ = 0
				continue
			}
			cur.WriteByte(ch)
			piece = true
		case ch == '"' || ch == '\'':
			inQ = ch
			piece = true
		case unicode.IsSpace(rune(ch)):
			if piece {
				out = append(out, cur.String())
				cur.Reset()
				piece = false
			}
		default:
			cur.WriteByte(ch)
			piece = true
		}
	}
	if inQ != 0 {
		return nil, fmt.Errorf("repl: unterminated %c-quoted string", inQ)
	}
	if piece {
		out = append(out, cur.String())
	}
	return out, nil
}
