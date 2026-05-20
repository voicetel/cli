package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
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
//nolint:gosec // env-var NAMES (read via os.Getenv), not credential values; G101 false-positives on API_KEY / PASSWORD substrings.
const (
	envAPIKey   = "VOICETEL_API_KEY"
	envUsername = "VOICETEL_USERNAME"
	envPassword = "VOICETEL_PASSWORD"
	envBaseURL  = "VOICETEL_BASE_URL"
)

// exitOnErr is the indirection layer between main() and os.Exit. Tests
// override it to assert on the exit code without actually terminating the
// test binary.
var exitOnErr = func(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "voicetel-cli:", err)
		os.Exit(1)
	}
}

// main is the binary entry point. Thin wrapper around run() + exitOnErr so
// every branch is testable.
func main() {
	exitOnErr(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entry point. It accepts the command-line args (without
// argv[0]), the writers to use for stdout/stderr, and parses flags via a
// scoped flag.FlagSet — never touching flag.CommandLine. Tests drive it
// with whatever args + writers they like.
func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("voicetel-cli", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		baseURL    = fs.String("base-url", "", "Override the API endpoint for this session.")
		apiKey     = fs.String("api-key", "", "Install an API key for this session (not persisted).")
		oneShot    = fs.String("x", "", "Run a single command non-interactively and exit. Example: -x 'account numbers'")
		cpuProfile = fs.String("cpu-profile", "", "Write a CPU profile to the given file (e.g. cpu.pprof). Hidden debug flag.")
		memProfile = fs.String("mem-profile", "", "Write a heap profile to the given file (e.g. mem.pprof). Hidden debug flag.")
		showVer    = fs.Bool("version", false, "Print version and exit.")
	)
	fs.Usage = func() { usage(stderr, fs) }
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *showVer {
		fmt.Fprintf(stdout, "voicetel-cli %s\n", Version)
		fmt.Fprintf(stdout, "  build time: %s\n", BuildTime)
		fmt.Fprintf(stdout, "  git commit: %s\n", GitCommit)
		return nil
	}

	// CPU profiling, if requested. Writes a profile to the named file for the
	// entire lifetime of the process. View via `go tool pprof <file>`.
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			return fmt.Errorf("cpu-profile: %w", err)
		}
		defer func() { _ = f.Close() }()
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("cpu-profile: %w", err)
		}
		defer pprof.StopCPUProfile()
	}

	printer := output.New(stdout, stderr)

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

	// Heap profiling, if requested. Captured at process exit so the snapshot
	// reflects steady-state memory after all command work is done.
	defer writeMemProfile(*memProfile, stderr)

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

// usage prints the --help text. Extracted from run() so tests can exercise
// it without driving the full flag-parse path.
func usage(w io.Writer, fs *flag.FlagSet) {
	fmt.Fprintf(w, "voicetel-cli %s — interactive REPL for the VoiceTel REST API.\n\n", Version)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  voicetel-cli [--base-url=URL] [--api-key=KEY]")
	fmt.Fprintln(w, "  voicetel-cli -x '<command>'      # one-shot, no REPL")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Environment variables (override config, are overridden by flags):")
	fmt.Fprintf(w, "  %s     32-hex API key — installed directly, no login round-trip\n", envAPIKey)
	fmt.Fprintf(w, "  %s    Numeric account id — combined with VOICETEL_PASSWORD, logs in on start\n", envUsername)
	fmt.Fprintf(w, "  %s    Password — paired with VOICETEL_USERNAME; never persisted\n", envPassword)
	fmt.Fprintf(w, "  %s    Override the API endpoint (rare; staging/sandbox)\n", envBaseURL)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Inside the REPL, type `help` for every command. Exit with `exit`, `quit`, or Ctrl-D.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fs.PrintDefaults()
}

// writeMemProfile captures a heap profile to the named file. No-op when path
// is empty. Errors are reported to stderr but never propagated — profiling
// failures should not break the main work the CLI was actually doing.
func writeMemProfile(path string, stderr io.Writer) {
	if path == "" {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(stderr, "mem-profile: %v\n", err)
		return
	}
	defer func() { _ = f.Close() }()
	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		fmt.Fprintf(stderr, "mem-profile: %v\n", err)
	}
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
