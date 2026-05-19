package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/voicetel/cli/internal/config"
	"github.com/voicetel/cli/internal/output"
	"github.com/voicetel/cli/internal/sdkclient"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "voicetel:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		baseURL = flag.String("base-url", "", "Override the API endpoint for this session.")
		apiKey  = flag.String("api-key", "", "Install an API key for this session (not persisted).")
		showVer = flag.Bool("version", false, "Print version and exit.")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "voicetel %s — interactive REPL for the VoiceTel REST API.\n\n", Version)
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  voicetel [--base-url=URL] [--api-key=KEY]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Inside the REPL, type `help` to see every command. Exit with `exit`,")
		fmt.Fprintln(os.Stderr, "`quit`, or Ctrl-D.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVer {
		fmt.Printf("voicetel %s\n", Version)
		return nil
	}

	printer := output.New(os.Stdout, os.Stderr)

	cfg, err := config.Load()
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return fmt.Errorf("config: %w", err)
	}

	if *baseURL != "" {
		cfg.BaseURL = *baseURL
	}
	if *apiKey != "" {
		cfg.APIKey = *apiKey
	}

	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "voicetel-cli/"+Version)

	histPath, err := config.HistoryPath()
	if err != nil {
		return err
	}

	ctx, cancel := signalContext()
	defer cancel()

	printBanner(printer, client)

	return runLoop(ctx, loopOptions{
		Client:   client,
		Printer:  printer,
		HistFile: histPath,
		Prompt:   "voicetel> ",
		Cfg:      cfg,
	})
}

func printBanner(p *output.Printer, c sdkclient.Client) {
	p.Printf("VoiceTel CLI %s  —  type `help` for commands, `exit` to quit.\n", Version)
	p.Printf("Endpoint: %s\n", c.BaseURL())
	if c.APIKey() == "" {
		p.Printf("No API key configured. Run `login <username> <password>` or `set api-key <key>`.\n")
	}
}

// signalContext returns a context cancelled on SIGINT or SIGTERM. Inside the
// REPL we rely on readline for line editing — the signal context here is only
// used by SDK calls in flight.
func signalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	go func() {
		<-ch
		cancel()
	}()
	return ctx, cancel
}
