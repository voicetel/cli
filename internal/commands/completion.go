package commands

// CompletionData returns the inputs the repl package needs to build
// a tab-completer. We expose this rather than the readline-typed
// completer to keep commands free of readline coupling and avoid an
// import cycle with the repl package.
func (r *Registry) CompletionData() (builtins []string, helpTopics []string, groupSubs map[string][]string) {
	groupSubs = map[string][]string{}
	seenTopic := map[string]bool{}
	for _, b := range r.Builtins {
		for i, n := range b.Names {
			if i == 0 {
				builtins = append(builtins, n)
			}
			if !seenTopic[n] {
				helpTopics = append(helpTopics, n)
				seenTopic[n] = true
			}
		}
	}
	for _, g := range r.Groups {
		subs := make([]string, 0, len(g.Commands))
		for _, c := range g.Commands {
			subs = append(subs, c.Names[0])
		}
		groupSubs[g.Name] = subs
		if !seenTopic[g.Name] {
			helpTopics = append(helpTopics, g.Name)
			seenTopic[g.Name] = true
		}
	}
	return builtins, helpTopics, groupSubs
}
