package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/voicetel/cli/internal/commands"
	"github.com/voicetel/cli/internal/config"
	"github.com/voicetel/cli/internal/output"
	"github.com/voicetel/cli/internal/repl"
	"github.com/voicetel/cli/internal/sdkclient"
)

// Environment variable names recognized by the CLI. Flag values take
// precedence over env vars; env vars take precedence over ~/.voicetel/config.toml.
//
//nolint:gosec // These are env-var NAMES (constants the CLI reads from os.Getenv),
//             // not credential values. gosec G101 false-positives on the API_KEY substring.
const (
	envAPIKey   = "VOICETEL_API_KEY"
	envUsername = "VOICETEL_USERNAME"
	envPassword = "VOICETEL_PASSWORD"
	envBaseURL  = "VOICETEL_BASE_URL"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "voicetel-cli:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		baseURL = flag.String("base-url", "", "Override the API endpoint for this session.")
		apiKey  = flag.String("api-key", "", "Install an API key for this session (not persisted).")
		oneShot = flag.String("x", "", "Run a single command non-interactively and exit. Example: -x 'account numbers'")
		showVer = flag.Bool("version", false, "Print version and exit.")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "voicetel-cli %s — interactive REPL for the VoiceTel REST API.\n\n", Version)
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  voicetel-cli [--base-url=URL] [--api-key=KEY]")
		fmt.Fprintln(os.Stderr, "  voicetel-cli -x '<command>'      # one-shot, no REPL")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Environment variables (override config, are overridden by flags):")
		fmt.Fprintf(os.Stderr, "  %s     32-hex API key — installed directly, no login round-trip\n", envAPIKey)
		fmt.Fprintf(os.Stderr, "  %s    Numeric account id — combined with VOICETEL_PASSWORD, logs in on start\n", envUsername)
		fmt.Fprintf(os.Stderr, "  %s    Password — paired with VOICETEL_USERNAME; never persisted\n", envPassword)
		fmt.Fprintf(os.Stderr, "  %s    Override the API endpoint (rare; staging/sandbox)\n", envBaseURL)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Inside the REPL, type `help` for every command. Exit with `exit`, `quit`, or Ctrl-D.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVer {
		fmt.Printf("voicetel-cli %s\n", Version)
		fmt.Printf("  build time: %s\n", BuildTime)
		fmt.Printf("  git commit: %s\n", GitCommit)
		return nil
	}

	printer := output.New(os.Stdout, os.Stderr)

	cfg, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return fmt.Errorf("config: %w", err)
	}

	// Precedence: flag > env > config. Apply env first, then let flags override.
	if env := os.Getenv(envBaseURL); env != "" {
		cfg.BaseURL = env
	}
	if env := os.Getenv(envAPIKey); env != "" {
		cfg.APIKey = env
	}
	if *baseURL != "" {
		cfg.BaseURL = *baseURL
	}
	if *apiKey != "" {
		cfg.APIKey = *apiKey
	}

	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "voicetel-cli/"+Version)

	ctx, cancel := signalContext(*oneShot != "")
	defer cancel()

	// VOICETEL_USERNAME + VOICETEL_PASSWORD: log in on start if api-key wasn't
	// supplied via flag, env, or config. The exchanged key is installed on the
	// client but NOT written to disk (no OnConfigChanged here — env-driven
	// runs are intentionally ephemeral; -x mode should not mutate user state).
	if client.APIKey() == "" {
		uStr := os.Getenv(envUsername)
		pStr := os.Getenv(envPassword)
		if uStr != "" && pStr != "" {
			username, err := strconv.Atoi(uStr)
			if err != nil {
				return fmt.Errorf("%s must be an integer, got %q: %w", envUsername, uStr, err)
			}
			key, err := client.Login(ctx, username, pStr)
			if err != nil {
				return fmt.Errorf("login (from %s / %s): %w", envUsername, envPassword, err)
			}
			client.SetAPIKey(key)
		}
	}

	// One-shot mode: dispatch the single command and exit. Skip the banner,
	// skip readline, skip history.
	if *oneShot != "" {
		return runOneShot(ctx, client, printer, cfg, *oneShot)
	}

	histPath, err := config.HistoryPath()
	if err != nil {
		return err
	}

	printBanner(printer, client)

	return runLoop(ctx, loopOptions{
		Client:   client,
		Printer:  printer,
		HistFile: histPath,
		Prompt:   "voicetel> ",
		Cfg:      cfg,
	})
}

// runOneShot wires the command registry minus the readline loop, dispatches a
// single command line, and returns. It's the implementation behind the -x flag.
func runOneShot(ctx context.Context, client sdkclient.Client, printer *output.Printer, cfg *config.Config, line string) error {
	registry := commands.BuildRegistry()
	cctx := &commands.Context{
		Ctx:     ctx,
		Client:  client,
		Printer: printer,
		OnConfigChanged: func() {
			cfg.APIKey = client.APIKey()
			cfg.BaseURL = client.BaseURL()
			if err := config.Save(cfg); err != nil {
				printer.Errorf("config: save: %v", err)
			}
		},
	}
	p, err := repl.Parse(line)
	if err != nil {
		if errors.Is(err, repl.ErrEmpty) {
			return fmt.Errorf("-x: empty command")
		}
		return fmt.Errorf("-x: %w", err)
	}
	if err := registry.Dispatch(cctx, p.Tokens, p.Raw); err != nil {
		if errors.Is(err, commands.ErrExit) {
			return nil
		}
		return err
	}
	return nil
}

func printBanner(p *output.Printer, c sdkclient.Client) {
	p.Printf("VoiceTel CLI %s  —  type `help` for commands, `exit` to quit.\n", Version)
	p.Printf("Endpoint: %s\n", c.BaseURL())
	if c.APIKey() == "" {
		p.Printf("No API key configured. Run `login <username> <password>` or `set api-key <key>`.\n")
	}
}

// signalContext returns a context cancelled on SIGTERM (and SIGINT in one-shot
// mode). The REPL relies on readline for line editing — Ctrl-C there is
// intercepted by readline, so we only listen to SIGINT when there's no readline
// in the picture (one-shot mode).
func signalContext(oneShotMode bool) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := []os.Signal{syscall.SIGTERM}
	if oneShotMode {
		sigs = append(sigs, syscall.SIGINT)
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sigs...)
	go func() {
		<-ch
		cancel()
	}()
	return ctx, cancel
}
