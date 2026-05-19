package repl

import "testing"

func TestBuildCompleterSmoke(t *testing.T) {
	c := BuildCompleter(
		[]string{"help", "exit", "login"},
		[]string{"help", "exit", "numbers"},
		map[string][]string{
			"numbers": {"list", "get", "add"},
		},
	)
	if c == nil {
		t.Fatal("nil completer")
	}
}
