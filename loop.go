package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/chzyer/readline"

	"github.com/voicetel/cli/internal/commands"
	"github.com/voicetel/cli/internal/config"
	"github.com/voicetel/cli/internal/output"
	"github.com/voicetel/cli/internal/repl"
	"github.com/voicetel/cli/internal/sdkclient"
)

// loopOptions bundles the REPL's wiring.
type loopOptions struct {
	Client   sdkclient.Client
	Printer  *output.Printer
	HistFile string
	Prompt   string
	Cfg      *config.Config
}

// runLoop starts the interactive REPL. It blocks until the user exits
// or stdin closes.
func runLoop(ctx context.Context, opts loopOptions) error {
	registry := commands.BuildRegistry()
	builtins, helpTopics, groupSubs := registry.CompletionData()
	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 opts.Prompt,
		HistoryFile:            opts.HistFile,
		AutoComplete:           repl.BuildCompleter(builtins, helpTopics, groupSubs),
		InterruptPrompt:        "^C",
		EOFPrompt:              "exit",
		HistorySearchFold:      true,
		DisableAutoSaveHistory: false,
	})
	if err != nil {
		return fmt.Errorf("repl: init readline: %w", err)
	}
	defer func() { _ = rl.Close() }()

	cmdCtx := &commands.Context{
		Ctx:     ctx,
		Client:  opts.Client,
		Printer: opts.Printer,
		OnConfigChanged: func() {
			opts.Cfg.APIKey = opts.Client.APIKey()
			opts.Cfg.BaseURL = opts.Client.BaseURL()
			if err := config.Save(opts.Cfg); err != nil {
				opts.Printer.Errorf("config: save: %v", err)
			}
		},
	}

	for {
		line, err := rl.Readline()
		if errors.Is(err, readline.ErrInterrupt) {
			continue
		}
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(os.Stdout)
			return nil
		}
		if err != nil {
			return fmt.Errorf("repl: read line: %w", err)
		}
		if err := dispatchLine(cmdCtx, registry, line); err != nil {
			if errors.Is(err, commands.ErrExit) {
				return nil
			}
			opts.Printer.Error(err)
		}
	}
}

// dispatchLine parses a single line and dispatches it. Exposed at
// package level so tests can drive command dispatch without spinning up
// readline.
func dispatchLine(cctx *commands.Context, r *commands.Registry, line string) error {
	p, err := repl.Parse(line)
	if err != nil {
		if errors.Is(err, repl.ErrEmpty) {
			return nil
		}
		return err
	}
	return r.Dispatch(cctx, p.Tokens, p.Raw)
}
