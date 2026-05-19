package repl

import (
	"github.com/chzyer/readline"
)

// BuildCompleter constructs a readline auto-completer from a flat
// description of the verb tree.
//
// builtins      — top-level verbs (login, exit, set, whoami, clear, help, ...).
// helpTopics    — what `help <TAB>` should suggest (groups + builtins).
// groupSubs     — for each resource group, the list of sub-command names.
//
// We deliberately avoid coupling this package to the commands package
// (which would create an import cycle, since commands references repl
// for flag parsing).
func BuildCompleter(builtins []string, helpTopics []string, groupSubs map[string][]string) readline.AutoCompleter {
	root := readline.NewPrefixCompleter()
	for _, b := range builtins {
		if b == "help" {
			children := make([]readline.PrefixCompleterInterface, 0, len(helpTopics))
			for _, h := range helpTopics {
				children = append(children, readline.PcItem(h))
			}
			root.Children = append(root.Children, readline.PcItem(b, children...))
			continue
		}
		root.Children = append(root.Children, readline.PcItem(b))
	}
	for name, subs := range groupSubs {
		children := make([]readline.PrefixCompleterInterface, 0, len(subs))
		for _, s := range subs {
			children = append(children, readline.PcItem(s))
		}
		root.Children = append(root.Children, readline.PcItem(name, children...))
	}
	return root
}
