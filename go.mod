module github.com/voicetel/cli

go 1.26.0

require (
	github.com/chzyer/readline v1.5.1
	github.com/pelletier/go-toml/v2 v2.4.3
	github.com/voicetel/go-sdk/v2 v2.2.17
	golang.org/x/term v0.47.0
)

require golang.org/x/sys v0.49.0 // indirect

// Build against the local SDK source tree, not the registry copy.
