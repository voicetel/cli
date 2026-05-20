package main

import (
	"context"
	"errors"
	"fmt"
	"io"

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

// lineSource abstracts the bit of *readline.Instance that the REPL loop
// actually uses. *readline.Instance satisfies it natively; tests substitute a
// scripted source backed by a slice of lines.
type lineSource interface {
	Readline() (string, error)
	Close() error
}

// runLoop builds a readline-backed lineSource and hands off to runLoopWith.
// Split this way so unit tests can drive the inner loop without spawning a
// real PTY or interacting with the user's terminal config.
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
	return runLoopWith(ctx, opts, registry, rl, opts.Printer.Stdout())
}

// runLoopWith is the testable inner loop. It reads lines from src, dispatches
// each through the registry, and returns when src returns io.EOF, commands.
// ErrExit fires, or src.Readline errors with anything else. `eofOut` is where
// a trailing newline gets written on EOF (typically os.Stdout to keep the
// shell prompt cosmetic-correct).
func runLoopWith(ctx context.Context, opts loopOptions, registry *commands.Registry, src lineSource, eofOut io.Writer) error {
	defer func() { _ = src.Close() }()

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
		line, err := src.Readline()
		if errors.Is(err, readline.ErrInterrupt) {
			continue
		}
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(eofOut)
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

// dispatchLine parses a single line and dispatches it. Exposed at package
// level so tests can drive command dispatch without spinning up readline.
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
