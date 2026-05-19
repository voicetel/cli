// Package commands wires REPL verbs to SDK calls.
//
// The package exposes a single Registry that the REPL uses to look up,
// help-print, and dispatch commands. Each resource group registers its
// sub-commands in an init-style helper called from BuildRegistry.
package commands

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/voicetel/cli/internal/output"
	"github.com/voicetel/cli/internal/sdkclient"
)

// Context carries the dispatcher's collaborators into each handler so we
// don't have to thread them through individual arg lists.
type Context struct {
	Ctx     context.Context
	Client  sdkclient.Client
	Printer *output.Printer

	// OnConfigChanged is invoked when a command mutates connection
	// state (api key, base url). The REPL persists the change.
	OnConfigChanged func()
}

// Handler is the signature every command implements.
//
// args are the tokens after the verb chain (e.g. for "numbers get
// 2015551234", args is ["2015551234"]).
//
// jsonTail is the raw remainder of the line, suitable for commands that
// take a JSON body. It is "" when the command does not take a body.
type Handler func(c *Context, args []string, jsonTail string) error

// Command is one leaf in the verb tree.
type Command struct {
	// Names: the verb itself plus any aliases. Names[0] is the canonical form.
	Names []string
	// Synopsis: one-line description used in `help`.
	Synopsis string
	// Usage: detailed multi-line help string.
	Usage string
	// Run executes the command.
	Run Handler
}

// Group is a top-level resource (e.g. "numbers"). It groups sub-commands.
type Group struct {
	Name        string
	Description string
	Commands    []*Command
}

// Registry is the entire verb tree.
type Registry struct {
	Groups   []*Group
	Builtins []*Command // top-level commands not under a group (help, login, ...)
}

// NewRegistry builds an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// AddGroup appends a group to the registry.
func (r *Registry) AddGroup(g *Group) { r.Groups = append(r.Groups, g) }

// AddBuiltin adds a top-level command.
func (r *Registry) AddBuiltin(c *Command) { r.Builtins = append(r.Builtins, c) }

// FindGroup returns the group with a matching name, or nil.
func (r *Registry) FindGroup(name string) *Group {
	for _, g := range r.Groups {
		if g.Name == name {
			return g
		}
	}
	return nil
}

// FindBuiltin returns the builtin matching name or one of its aliases.
func (r *Registry) FindBuiltin(name string) *Command {
	for _, c := range r.Builtins {
		for _, n := range c.Names {
			if n == name {
				return c
			}
		}
	}
	return nil
}

// FindCommand returns the sub-command in g matching name (canonical or
// alias).
func (g *Group) FindCommand(name string) *Command {
	for _, c := range g.Commands {
		for _, n := range c.Names {
			if n == name {
				return c
			}
		}
	}
	return nil
}

// Dispatch resolves and executes a parsed line.
func (r *Registry) Dispatch(c *Context, tokens []string, raw string) error {
	if len(tokens) == 0 {
		return nil
	}
	head := tokens[0]
	// Top-level builtins first.
	if b := r.FindBuiltin(head); b != nil {
		return b.Run(c, tokens[1:], tailFromRaw(raw, 1))
	}
	// Resource group?
	g := r.FindGroup(head)
	if g == nil {
		return fmt.Errorf("unknown command %q — type `help` for the command list", head)
	}
	if len(tokens) < 2 {
		return fmt.Errorf("%s: missing sub-command — try `help %s`", head, head)
	}
	sub := tokens[1]
	cmd := g.FindCommand(sub)
	if cmd == nil {
		return fmt.Errorf("%s: unknown sub-command %q — try `help %s`", head, sub, head)
	}
	return cmd.Run(c, tokens[2:], tailFromRaw(raw, 2))
}

// tailFromRaw mirrors Parsed.TailFrom but operates on a raw string so
// the dispatcher doesn't need access to the parsed wrapper.
func tailFromRaw(raw string, n int) string {
	s := strings.TrimSpace(raw)
	for i := 0; i < n; i++ {
		s = strings.TrimSpace(s)
		// consume one whitespace-separated, quote-aware token
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
			if ch == ' ' || ch == '\t' {
				break
			}
			end++
		}
		s = s[end:]
	}
	return strings.TrimSpace(s)
}

// BuildRegistry returns the fully-populated default Registry.
func BuildRegistry() *Registry {
	r := NewRegistry()
	r.AddBuiltin(helpCommand(r))
	r.AddBuiltin(&Command{
		Names:    []string{"exit", "quit"},
		Synopsis: "Leave the REPL.",
		Usage:    "exit | quit\n  Exit the REPL. Ctrl-D works too.",
		Run: func(_ *Context, _ []string, _ string) error {
			return ErrExit
		},
	})
	r.AddBuiltin(&Command{
		Names:    []string{"clear"},
		Synopsis: "Clear the screen.",
		Usage:    "clear\n  Clear the terminal.",
		Run: func(c *Context, _ []string, _ string) error {
			c.Printer.Printf("\x1b[H\x1b[2J")
			return nil
		},
	})
	r.AddBuiltin(loginCommand())
	r.AddBuiltin(setCommand())
	r.AddBuiltin(whoamiCommand())

	registerAccount(r)
	registerACL(r)
	registerAuthentication(r)
	registerE911(r)
	registerGateways(r)
	registerINumbering(r)
	registerLookups(r)
	registerMessaging(r)
	registerNumbers(r)
	registerSupport(r)

	for _, g := range r.Groups {
		sort.SliceStable(g.Commands, func(i, j int) bool {
			return g.Commands[i].Names[0] < g.Commands[j].Names[0]
		})
	}
	sort.SliceStable(r.Groups, func(i, j int) bool {
		return r.Groups[i].Name < r.Groups[j].Name
	})
	return r
}

// ErrExit signals the REPL loop to terminate cleanly.
var ErrExit = errors.New("repl: exit requested")
