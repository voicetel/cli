package commands

import (
	"fmt"
	"strings"
)

// helpCommand builds the dynamic help command. We pass the registry in
// rather than reaching for a package-level variable so tests can build
// alternate registries.
func helpCommand(r *Registry) *Command {
	return &Command{
		Names:    []string{"help", "?"},
		Synopsis: "Show help for the REPL or a specific command/topic.",
		Usage: "help [topic]\n" +
			"  With no argument, prints the top-level command tree.\n" +
			"  With one argument, prints details for that resource group or builtin.\n" +
			"  With two arguments, prints details for the sub-command (e.g. `help numbers list`).",
		Run: func(c *Context, args []string, _ string) error {
			switch len(args) {
			case 0:
				c.Printer.Println(renderRoot(r))
				return nil
			case 1:
				topic := args[0]
				if b := r.FindBuiltin(topic); b != nil {
					c.Printer.Println(renderCommand(topic, b))
					return nil
				}
				if g := r.FindGroup(topic); g != nil {
					c.Printer.Println(renderGroup(g))
					return nil
				}
				return fmt.Errorf("help: no such topic %q", topic)
			default:
				g := r.FindGroup(args[0])
				if g == nil {
					return fmt.Errorf("help: no such group %q", args[0])
				}
				cmd := g.FindCommand(args[1])
				if cmd == nil {
					return fmt.Errorf("help: %s has no sub-command %q", args[0], args[1])
				}
				c.Printer.Println(renderCommand(args[0]+" "+args[1], cmd))
				return nil
			}
		},
	}
}

func renderRoot(r *Registry) string {
	var b strings.Builder
	b.WriteString("VoiceTel CLI — interactive REPL.\n\n")
	b.WriteString("Top-level commands:\n")
	for _, c := range r.Builtins {
		fmt.Fprintf(&b, "  %-24s %s\n", strings.Join(c.Names, " | "), c.Synopsis)
	}
	b.WriteString("\nResource groups:\n")
	for _, g := range r.Groups {
		fmt.Fprintf(&b, "  %-24s %s\n", g.Name, g.Description)
	}
	b.WriteString("\nType `help <topic>` for details — for example `help numbers`.\n")
	return b.String()
}

func renderGroup(g *Group) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n\n", g.Name, g.Description)
	for _, c := range g.Commands {
		fmt.Fprintf(&b, "  %s %-24s %s\n", g.Name, strings.Join(c.Names, " | "), c.Synopsis)
	}
	fmt.Fprintf(&b, "\nType `help %s <sub-command>` for details.\n", g.Name)
	return b.String()
}

func renderCommand(label string, c *Command) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n\n", label, c.Synopsis)
	if c.Usage != "" {
		b.WriteString(c.Usage)
		b.WriteString("\n")
	}
	if len(c.Names) > 1 {
		fmt.Fprintf(&b, "\nAliases: %s\n", strings.Join(c.Names[1:], ", "))
	}
	return b.String()
}
