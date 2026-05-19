package commands

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// skipArgs trims n whitespace-separated tokens (quote-aware) from the
// front of tail and returns the remainder. Used by commands that take
// one or more positional arguments followed by a JSON body, where the
// body still lives inside `tail`.
//
// Example:
//
//	skipArgs(`2015551234 {"route":4}`, 1) -> `{"route":4}`
func skipArgs(tail string, n int) string {
	s := strings.TrimSpace(tail)
	for i := 0; i < n; i++ {
		s = strings.TrimSpace(s)
		end := 0
		inQ := byte(0)
		for end < len(s) {
			ch := s[end]
			if inQ != 0 {
				if ch == inQ {
					inQ = 0
				}
				end++
				continue
			}
			if ch == '"' || ch == '\'' {
				inQ = ch
				end++
				continue
			}
			if unicode.IsSpace(rune(ch)) {
				break
			}
			end++
		}
		s = s[end:]
	}
	return strings.TrimSpace(s)
}

// parseJSON unmarshals a JSON body argument into v. Returns a friendly
// error if the body is empty or malformed.
func parseJSON(label, body string, v any) error {
	if body == "" {
		return fmt.Errorf("%s: missing JSON body argument", label)
	}
	if err := json.Unmarshal([]byte(body), v); err != nil {
		return fmt.Errorf("%s: invalid JSON body: %w", label, err)
	}
	return nil
}

// requireArgs asserts at least n positional args were supplied.
func requireArgs(label string, args []string, n int, hint string) error {
	if len(args) < n {
		return fmt.Errorf("%s: expected %d argument(s) — %s", label, n, hint)
	}
	return nil
}

// argInt parses arg as an int, surfacing a clear error.
func argInt(label string, raw string) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: expected integer, got %q", label, raw)
	}
	return n, nil
}

// printResultG is sugar for "if err return err, else JSON-print v".
// Generic over T so callers can pass typed SDK return values without
// having to nil-unwrap explicitly. nil pointers are detected and
// printed as JSON null suppressed.
func printResultG[T any](c *Context, v *T, err error) error {
	if err != nil {
		return err
	}
	if v == nil {
		return nil
	}
	return c.Printer.JSON(v)
}
