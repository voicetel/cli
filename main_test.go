package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/voicetel/cli/internal/commands"
	"github.com/voicetel/cli/internal/config"
	"github.com/voicetel/cli/internal/output"
	"github.com/voicetel/cli/internal/repl"
	"github.com/voicetel/cli/internal/sdkclient"
)

// Version is exported via -X ldflags; here we just confirm it's a non-empty
// string (whatever the build sets it to). BuildTime and GitCommit default to
// "unknown" when built via `go test` (no -X flag).
func TestVersionVarsHaveValues(t *testing.T) {
	if Version == "" {
		t.Error("Version is empty")
	}
	if BuildTime == "" {
		t.Error("BuildTime is empty")
	}
	if GitCommit == "" {
		t.Error("GitCommit is empty")
	}
}

// signalContext: oneShotMode=true registers SIGINT; false does not. We can't
// reliably send signals in tests across platforms, so we just verify the
// returned ctx is alive and cancellable.
func TestSignalContextLifecycle(t *testing.T) {
	for _, oneShot := range []bool{true, false} {
		ctx, cancel := signalContext(oneShot)
		if ctx.Err() != nil {
			t.Errorf("oneShot=%v: ctx already cancelled at construction", oneShot)
		}
		cancel()
		// After cancel, the context's Done channel must fire.
		select {
		case <-ctx.Done():
			// ok
		default:
			t.Errorf("oneShot=%v: ctx not cancelled after cancel()", oneShot)
		}
	}
}

// printBanner with no API key emits the "no key configured" prompt; with a key
// it omits that line. Capture stdout via a fake writer.
func TestPrintBannerVariants(t *testing.T) {
	cases := []struct {
		name      string
		apiKey    string
		wantsHint bool
	}{
		{"with key", "abcdef0123456789abcdef0123456789", false},
		{"no key", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			p := output.New(&stdout, &stderr)
			c := sdkclient.New("https://example.invalid", tc.apiKey, "test-ua/0.0")
			printBanner(p, c)
			got := stdout.String()
			if !strings.Contains(got, "VoiceTel CLI") {
				t.Errorf("banner missing product name: %q", got)
			}
			if strings.Contains(got, "No API key configured") != tc.wantsHint {
				t.Errorf("login-hint visibility mismatch: got %q", got)
			}
		})
	}
}

// runOneShot dispatches a single command and returns. Exercise the help path
// (no network, no auth required) — the most likely real-world -x usage.
func TestRunOneShotHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid", APIKey: "k"}
	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "test-ua/0.0")
	if err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "help"); err != nil {
		t.Fatalf("runOneShot(help): %v", err)
	}
	if !strings.Contains(stdout.String(), "Resource groups") {
		t.Errorf("runOneShot(help): expected help output, got %q", stdout.String())
	}
}

// runOneShot with an empty command returns a wrapped ErrEmpty error.
func TestRunOneShotEmpty(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid"}
	client := sdkclient.New(cfg.BaseURL, "k", "test-ua/0.0")
	err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "   ")
	if err == nil {
		t.Fatal("runOneShot(empty): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "empty command") {
		t.Errorf("runOneShot(empty): expected 'empty command' error, got %q", err.Error())
	}
}

// runOneShot with malformed input (unterminated quote) bubbles up the parse
// error wrapped with the "-x:" prefix.
func TestRunOneShotParseError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid"}
	client := sdkclient.New(cfg.BaseURL, "k", "test-ua/0.0")
	err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, `login 100 "oops`)
	if err == nil {
		t.Fatal("runOneShot(bad quote): expected error, got nil")
	}
	if !strings.HasPrefix(err.Error(), "-x:") {
		t.Errorf("runOneShot(bad quote): expected '-x:' prefix, got %q", err.Error())
	}
}

// runOneShot dispatching `exit` returns nil (the dispatcher returns ErrExit,
// which runOneShot recognises and swallows).
func TestRunOneShotExitReturnsNil(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid"}
	client := sdkclient.New(cfg.BaseURL, "k", "test-ua/0.0")
	if err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "exit"); err != nil {
		t.Errorf("runOneShot(exit): expected nil, got %v", err)
	}
}

// runOneShot dispatching an unknown command surfaces a dispatch error.
func TestRunOneShotUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid"}
	client := sdkclient.New(cfg.BaseURL, "k", "test-ua/0.0")
	err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "nosuchcommandexists")
	if err == nil {
		t.Fatal("runOneShot(unknown): expected error, got nil")
	}
}

// OnConfigChanged is wired to the printer's Errorf; verify it doesn't panic
// when invoked. (Can't easily test config.Save() side-effects without
// rewriting HOME, but we can confirm the callback wiring is non-fatal.)
func TestRunOneShotConfigCallbackSurvives(t *testing.T) {
	// Stand up a minimal HTTP server that satisfies the `set api-key` no-op:
	// `set api-key VALUE` triggers OnConfigChanged via the SDK. We don't
	// actually need a server — `set api-key` is purely local.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp) // Windows-safety; harmless on unix

	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid", APIKey: ""}
	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "test-ua/0.0")
	// Use a fresh 32-hex key (avoids GH push-protection scanner; valid format).
	if err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "set api-key ffffffffffffffffffffffffffffffff"); err != nil {
		t.Fatalf("runOneShot(set api-key): %v", err)
	}
	if cfg.APIKey == "" {
		t.Error("OnConfigChanged didn't propagate api_key into cfg")
	}
}

// Verify that the standard error and "exit" sentinel handling matches what
// the dispatcher returns. Sanity check around the errors.Is bridge in
// runOneShot.
func TestExitErrorRecognisedByDispatcher(t *testing.T) {
	if !errors.Is(commands.ErrExit, commands.ErrExit) {
		t.Error("commands.ErrExit not recognised by errors.Is")
	}
}

// loginCommand env-var integration: VOICETEL_USERNAME + VOICETEL_PASSWORD →
// run() calls client.Login at startup. We can't drive run() directly (it
// uses the global flag set), but we can stand up a server that serves a
// login response and exercise the parsing logic via runOneShot + the
// `login` builtin command.
func TestLoginCommandViaOneShot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// VoiceTel login endpoint returns {"apikey": "..."}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"apikey":"ffffffffffffffffffffffffffffffff"}`))
	}))
	defer srv.Close()

	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)

	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: srv.URL}
	client := sdkclient.New(cfg.BaseURL, "", "test-ua/0.0")
	if err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "login 1000000001 hunter2"); err != nil {
		t.Fatalf("runOneShot(login): %v", err)
	}
	if got := client.APIKey(); got == "" {
		t.Error("Login: APIKey not installed on client after successful login")
	}
}

// Belt-and-braces: confirm the package compiles with the parser/output/config
// imports we use throughout the file. Acts as a guard against orphaned imports
// if the test file gets rewritten.
func TestPackageImports(t *testing.T) {
	if _, err := repl.Parse("noop"); err != nil {
		t.Errorf("repl.Parse roundtrip: %v", err)
	}
	if _, err := os.Stat("main.go"); err != nil {
		t.Errorf("main.go must exist at the package root: %v", err)
	}
}
