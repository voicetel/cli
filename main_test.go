package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/chzyer/readline"

	"github.com/voicetel/cli/internal/commands"
	"github.com/voicetel/cli/internal/config"
	"github.com/voicetel/cli/internal/output"
	"github.com/voicetel/cli/internal/repl"
	"github.com/voicetel/cli/internal/sdkclient"
)

// --- helpers ------------------------------------------------------------

// scriptedSource feeds a fixed sequence of lines to runLoopWith and returns
// io.EOF when exhausted. Tests use it to drive the REPL deterministically.
type scriptedSource struct {
	lines []string
	i     int
	// err lets a test inject a non-EOF error after some lines (e.g.
	// readline.ErrInterrupt or a synthetic "transport closed").
	err error
}

func (s *scriptedSource) Readline() (string, error) {
	if s.i >= len(s.lines) {
		if s.err != nil {
			return "", s.err
		}
		return "", io.EOF
	}
	line := s.lines[s.i]
	s.i++
	return line, nil
}

func (s *scriptedSource) Close() error { return nil }

// isolatedHome points HOME at a t.TempDir() so config.Save() doesn't touch
// the user's real ~/.voicetel.
func isolatedHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp) // Windows
	return tmp
}

// loginServer stands up an httptest server that serves the VoiceTel login
// endpoint with a canned 32-hex key. The returned key matches what the
// server emits, so tests can assert on it.
func loginServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	const key = "ffffffffffffffffffffffffffffffff"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"apikey":"` + key + `"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, key
}

// --- version vars -------------------------------------------------------

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

// --- signalContext ------------------------------------------------------

func TestSignalContextLifecycle(t *testing.T) {
	for _, oneShot := range []bool{true, false} {
		ctx, cancel := signalContext(oneShot)
		if ctx.Err() != nil {
			t.Errorf("oneShot=%v: ctx already cancelled at construction", oneShot)
		}
		cancel()
		select {
		case <-ctx.Done():
			// expected
		case <-time.After(time.Second):
			t.Errorf("oneShot=%v: ctx not cancelled after cancel()", oneShot)
		}
	}
}

// Sending SIGTERM to ourselves while signalContext is active should cancel
// the context. This actually exercises the goroutine inside signalContext.
func TestSignalContextSIGTERM(t *testing.T) {
	ctx, cancel := signalContext(false)
	defer cancel()

	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("kill: %v", err)
	}
	select {
	case <-ctx.Done():
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("ctx not cancelled within 2s of SIGTERM")
	}
}

// --- printBanner --------------------------------------------------------

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

// --- runOneShot --------------------------------------------------------

func TestRunOneShotHelp(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid", APIKey: "k"}
	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "test-ua/0.0")
	if err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "help"); err != nil {
		t.Fatalf("runOneShot(help): %v", err)
	}
	if !strings.Contains(stdout.String(), "Resource groups") {
		t.Errorf("expected help output, got %q", stdout.String())
	}
}

func TestRunOneShotEmpty(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid"}
	client := sdkclient.New(cfg.BaseURL, "k", "test-ua/0.0")
	err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "   ")
	if err == nil || !strings.Contains(err.Error(), "empty command") {
		t.Fatalf("expected 'empty command' error, got %v", err)
	}
}

func TestRunOneShotParseError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid"}
	client := sdkclient.New(cfg.BaseURL, "k", "test-ua/0.0")
	err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, `login 100 "oops`)
	if err == nil || !strings.HasPrefix(err.Error(), "-x:") {
		t.Fatalf("expected '-x:' prefix, got %v", err)
	}
}

func TestRunOneShotExitReturnsNil(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid"}
	client := sdkclient.New(cfg.BaseURL, "k", "test-ua/0.0")
	if err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "exit"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestRunOneShotUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid"}
	client := sdkclient.New(cfg.BaseURL, "k", "test-ua/0.0")
	if err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "nosuchcommandexists"); err == nil {
		t.Fatal("expected error from unknown command, got nil")
	}
}

func TestRunOneShotConfigCallbackSaves(t *testing.T) {
	tmp := isolatedHome(t)
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid"}
	client := sdkclient.New(cfg.BaseURL, "", "test-ua/0.0")
	if err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "set api-key ffffffffffffffffffffffffffffffff"); err != nil {
		t.Fatalf("set api-key: %v", err)
	}
	if cfg.APIKey == "" {
		t.Error("OnConfigChanged didn't propagate api_key into cfg")
	}
	// Verify the persisted file exists (proves OnConfigChanged.Save() was called).
	if _, err := os.Stat(filepath.Join(tmp, ".voicetel", "config.toml")); err != nil {
		t.Errorf("config.toml not written: %v", err)
	}
}

func TestRunOneShotLoginViaSDK(t *testing.T) {
	isolatedHome(t)
	srv, _ := loginServer(t)
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: srv.URL}
	client := sdkclient.New(cfg.BaseURL, "", "test-ua/0.0")
	if err := runOneShot(context.Background(), client, output.New(&stdout, &stderr), cfg, "login 1000000001 hunter2"); err != nil {
		t.Fatalf("login: %v", err)
	}
	if client.APIKey() == "" {
		t.Error("login: APIKey not installed")
	}
}

// --- run() entry point --------------------------------------------------

func TestRunVersionFlag(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--version"}, &stdout, &stderr); err != nil {
		t.Fatalf("run --version: %v", err)
	}
	if !strings.Contains(stdout.String(), "voicetel-cli") {
		t.Errorf("--version output missing product name: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "build time:") {
		t.Errorf("--version output missing build time: %q", stdout.String())
	}
}

func TestRunBadFlagReturnsError(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{"--nosuchflag"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run with bad flag: expected error, got nil")
	}
}

func TestRunOneShotViaFlagX(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--api-key=ffffffffffffffffffffffffffffffff", "-x", "help"}, &stdout, &stderr); err != nil {
		t.Fatalf("run -x help: %v", err)
	}
	if !strings.Contains(stdout.String(), "Resource groups") {
		t.Errorf("expected help output, got %q", stdout.String())
	}
}

func TestRunOneShotViaFlagXParseError(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{"--api-key=ffffffffffffffffffffffffffffffff", "-x", `login 100 "oops`}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run -x with bad quote: expected error, got nil")
	}
}

func TestRunEnvVarBaseURLAndAPIKey(t *testing.T) {
	isolatedHome(t)
	t.Setenv(envBaseURL, "https://staging.example.invalid")
	t.Setenv(envAPIKey, "ffffffffffffffffffffffffffffffff")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-x", "help"}, &stdout, &stderr); err != nil {
		t.Fatalf("run with env: %v", err)
	}
}

func TestRunEnvUsernamePasswordTriggersLogin(t *testing.T) {
	isolatedHome(t)
	srv, _ := loginServer(t)
	t.Setenv(envBaseURL, srv.URL)
	t.Setenv(envUsername, "1000000001")
	t.Setenv(envPassword, "hunter2")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"-x", "help"}, &stdout, &stderr); err != nil {
		t.Fatalf("run with env login: %v", err)
	}
}

func TestRunEnvUsernameNonInteger(t *testing.T) {
	isolatedHome(t)
	t.Setenv(envUsername, "not-a-number")
	t.Setenv(envPassword, "hunter2")
	var stdout, stderr bytes.Buffer
	err := run([]string{"-x", "help"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "must be an integer") {
		t.Fatalf("expected integer-parse error, got %v", err)
	}
}

func TestRunEnvLoginFailurePropagates(t *testing.T) {
	isolatedHome(t)
	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer failSrv.Close()
	t.Setenv(envBaseURL, failSrv.URL)
	t.Setenv(envUsername, "1000000001")
	t.Setenv(envPassword, "wrong")
	var stdout, stderr bytes.Buffer
	err := run([]string{"-x", "help"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "login") {
		t.Fatalf("expected login error, got %v", err)
	}
}

func TestRunFlagOverridesEnv(t *testing.T) {
	isolatedHome(t)
	t.Setenv(envAPIKey, "envkey0123456789envkey0123456789")
	t.Setenv(envBaseURL, "https://from-env.invalid")
	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"--api-key=ffffffffffffffffffffffffffffffff",
		"--base-url=https://from-flag.invalid",
		"-x", "help",
	}, &stdout, &stderr); err != nil {
		t.Fatalf("run flag-over-env: %v", err)
	}
}

func TestRunCPUProfile(t *testing.T) {
	isolatedHome(t)
	dir := t.TempDir()
	cpuFile := filepath.Join(dir, "cpu.pprof")
	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"--api-key=ffffffffffffffffffffffffffffffff",
		"--cpu-profile", cpuFile,
		"-x", "help",
	}, &stdout, &stderr); err != nil {
		t.Fatalf("run --cpu-profile: %v", err)
	}
	info, err := os.Stat(cpuFile)
	if err != nil {
		t.Fatalf("cpu profile not written: %v", err)
	}
	if info.Size() == 0 {
		t.Error("cpu profile is empty")
	}
}

func TestRunMemProfile(t *testing.T) {
	isolatedHome(t)
	dir := t.TempDir()
	memFile := filepath.Join(dir, "mem.pprof")
	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"--api-key=ffffffffffffffffffffffffffffffff",
		"--mem-profile", memFile,
		"-x", "help",
	}, &stdout, &stderr); err != nil {
		t.Fatalf("run --mem-profile: %v", err)
	}
	info, err := os.Stat(memFile)
	if err != nil {
		t.Fatalf("mem profile not written: %v", err)
	}
	if info.Size() == 0 {
		t.Error("mem profile is empty")
	}
}

func TestRunCPUProfileBadPath(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--cpu-profile", "/nonexistent-dir/cpu.pprof",
		"-x", "help",
	}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "cpu-profile") {
		t.Fatalf("expected cpu-profile error, got %v", err)
	}
}

// --- writeMemProfile (the error-reporting path) ------------------------

func TestWriteMemProfileEmptyPathIsNoOp(t *testing.T) {
	var stderr bytes.Buffer
	writeMemProfile("", &stderr)
	if stderr.Len() != 0 {
		t.Errorf("expected no output, got %q", stderr.String())
	}
}

func TestWriteMemProfileBadPathReportsError(t *testing.T) {
	var stderr bytes.Buffer
	writeMemProfile("/nonexistent-dir/mem.pprof", &stderr)
	if !strings.Contains(stderr.String(), "mem-profile:") {
		t.Errorf("expected error message in stderr, got %q", stderr.String())
	}
}

// --- usage --------------------------------------------------------------

func TestUsagePrints(t *testing.T) {
	var buf bytes.Buffer
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("x", "", "")
	usage(&buf, fs)
	got := buf.String()
	for _, want := range []string{"voicetel-cli", envAPIKey, envUsername, envPassword, envBaseURL} {
		if !strings.Contains(got, want) {
			t.Errorf("usage output missing %q", want)
		}
	}
}

// --- runLoopWith --------------------------------------------------------

func TestRunLoopWithEOFExitsCleanly(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr, eofBuf bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid", APIKey: "k"}
	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "test-ua/0.0")
	opts := loopOptions{
		Client:  client,
		Printer: output.New(&stdout, &stderr),
		Cfg:     cfg,
		Prompt:  "test> ",
	}
	registry := commands.BuildRegistry()
	src := &scriptedSource{lines: []string{"help"}}
	if err := runLoopWith(context.Background(), opts, registry, src, &eofBuf); err != nil {
		t.Fatalf("runLoopWith: %v", err)
	}
	if !strings.Contains(stdout.String(), "Resource groups") {
		t.Errorf("help didn't dispatch in loop: %q", stdout.String())
	}
	if !strings.Contains(eofBuf.String(), "\n") {
		t.Error("expected trailing newline on EOF")
	}
}

func TestRunLoopWithExitCommand(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr, eofBuf bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid", APIKey: "k"}
	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "test-ua/0.0")
	opts := loopOptions{
		Client:  client,
		Printer: output.New(&stdout, &stderr),
		Cfg:     cfg,
	}
	registry := commands.BuildRegistry()
	// "exit" returns commands.ErrExit → runLoopWith returns nil before
	// hitting the EOF newline.
	src := &scriptedSource{lines: []string{"exit", "should-never-run"}}
	if err := runLoopWith(context.Background(), opts, registry, src, &eofBuf); err != nil {
		t.Fatalf("runLoopWith(exit): %v", err)
	}
	if eofBuf.Len() != 0 {
		t.Error("EOF newline written despite explicit exit")
	}
	if src.i != 1 {
		t.Errorf("expected to consume 1 line before exit, consumed %d", src.i)
	}
}

func TestRunLoopWithSkipsEmptyAndComment(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr, eofBuf bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid", APIKey: "k"}
	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "test-ua/0.0")
	opts := loopOptions{
		Client:  client,
		Printer: output.New(&stdout, &stderr),
		Cfg:     cfg,
	}
	registry := commands.BuildRegistry()
	src := &scriptedSource{lines: []string{"", "# a comment", "  ", "help"}}
	if err := runLoopWith(context.Background(), opts, registry, src, &eofBuf); err != nil {
		t.Fatalf("runLoopWith: %v", err)
	}
	if !strings.Contains(stdout.String(), "Resource groups") {
		t.Error("help didn't dispatch after blank/comment lines")
	}
}

func TestRunLoopWithInterruptContinues(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr, eofBuf bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid", APIKey: "k"}
	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "test-ua/0.0")
	opts := loopOptions{
		Client:  client,
		Printer: output.New(&stdout, &stderr),
		Cfg:     cfg,
	}
	registry := commands.BuildRegistry()
	// Interrupting source: returns ErrInterrupt mid-script, then keeps going.
	src := &interruptOnceSource{lines: []string{"help"}}
	if err := runLoopWith(context.Background(), opts, registry, src, &eofBuf); err != nil {
		t.Fatalf("runLoopWith(interrupt): %v", err)
	}
	if !strings.Contains(stdout.String(), "Resource groups") {
		t.Errorf("expected help to run after interrupt, got %q", stdout.String())
	}
}

// interruptOnceSource returns ErrInterrupt on the first Readline call, then
// behaves like scriptedSource.
type interruptOnceSource struct {
	lines   []string
	i       int
	tripped bool
}

func (s *interruptOnceSource) Readline() (string, error) {
	if !s.tripped {
		s.tripped = true
		return "", readline.ErrInterrupt
	}
	if s.i >= len(s.lines) {
		return "", io.EOF
	}
	line := s.lines[s.i]
	s.i++
	return line, nil
}

func (s *interruptOnceSource) Close() error { return nil }

func TestRunLoopWithReadlineError(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr, eofBuf bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid", APIKey: "k"}
	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "test-ua/0.0")
	opts := loopOptions{
		Client:  client,
		Printer: output.New(&stdout, &stderr),
		Cfg:     cfg,
	}
	registry := commands.BuildRegistry()
	src := &scriptedSource{lines: nil, err: errors.New("synthetic transport closed")}
	err := runLoopWith(context.Background(), opts, registry, src, &eofBuf)
	if err == nil || !strings.Contains(err.Error(), "read line") {
		t.Fatalf("expected 'read line' wrap, got %v", err)
	}
}

func TestRunLoopWithCommandError(t *testing.T) {
	isolatedHome(t)
	var stdout, stderr, eofBuf bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid", APIKey: "k"}
	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "test-ua/0.0")
	opts := loopOptions{
		Client:  client,
		Printer: output.New(&stdout, &stderr),
		Cfg:     cfg,
	}
	registry := commands.BuildRegistry()
	// Unknown command should be printed to stderr but NOT abort the loop.
	src := &scriptedSource{lines: []string{"nosuchcommand", "help"}}
	if err := runLoopWith(context.Background(), opts, registry, src, &eofBuf); err != nil {
		t.Fatalf("runLoopWith: %v", err)
	}
	if !strings.Contains(stderr.String(), "Error") {
		t.Errorf("expected error message on stderr, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Resource groups") {
		t.Errorf("expected help to still run after unknown command, got %q", stdout.String())
	}
}

func TestRunLoopWithConfigSavePropagates(t *testing.T) {
	tmp := isolatedHome(t)
	var stdout, stderr, eofBuf bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid", APIKey: ""}
	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "test-ua/0.0")
	opts := loopOptions{
		Client:  client,
		Printer: output.New(&stdout, &stderr),
		Cfg:     cfg,
	}
	registry := commands.BuildRegistry()
	src := &scriptedSource{lines: []string{"set api-key ffffffffffffffffffffffffffffffff"}}
	if err := runLoopWith(context.Background(), opts, registry, src, &eofBuf); err != nil {
		t.Fatalf("runLoopWith: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".voicetel", "config.toml")); err != nil {
		t.Errorf("OnConfigChanged didn't persist config: %v", err)
	}
}

// --- dispatchLine -------------------------------------------------------

func TestDispatchLineEmptyReturnsNil(t *testing.T) {
	registry := commands.BuildRegistry()
	cctx := &commands.Context{
		Ctx:     context.Background(),
		Printer: output.New(io.Discard, io.Discard),
	}
	if err := dispatchLine(cctx, registry, ""); err != nil {
		t.Errorf("expected nil for empty line, got %v", err)
	}
	if err := dispatchLine(cctx, registry, "# comment"); err != nil {
		t.Errorf("expected nil for comment line, got %v", err)
	}
}

func TestDispatchLineParseError(t *testing.T) {
	registry := commands.BuildRegistry()
	cctx := &commands.Context{
		Ctx:     context.Background(),
		Printer: output.New(io.Discard, io.Discard),
	}
	if err := dispatchLine(cctx, registry, `login 100 "oops`); err == nil {
		t.Error("expected parse error, got nil")
	}
}

// --- runLoop (the outer readline-backed version) -----------------------

// runLoop opens stdin via readline, which requires a TTY-or-pipe. Driving
// it via /dev/null lets readline finish without input; it should exit
// cleanly via EOF.
func TestRunLoopWithDevNullStdin(t *testing.T) {
	devnull, err := os.Open(os.DevNull)
	if err != nil {
		t.Skipf("can't open /dev/null: %v", err)
	}
	defer devnull.Close()
	// Save + restore stdin around the call — readline reads from os.Stdin.
	oldStdin := os.Stdin
	os.Stdin = devnull
	defer func() { os.Stdin = oldStdin }()

	tmp := isolatedHome(t)
	var stdout, stderr bytes.Buffer
	cfg := &config.Config{BaseURL: "https://example.invalid", APIKey: "k"}
	client := sdkclient.New(cfg.BaseURL, cfg.APIKey, "test-ua/0.0")
	if err := runLoop(context.Background(), loopOptions{
		Client:   client,
		Printer:  output.New(&stdout, &stderr),
		Cfg:      cfg,
		Prompt:   "test> ",
		HistFile: filepath.Join(tmp, "hist"),
	}); err != nil {
		t.Fatalf("runLoop: %v", err)
	}
}

// --- main() subprocess test --------------------------------------------

// TestMainSubprocess re-execs the test binary with VOICETEL_CLI_TEST_MAIN
// set, which causes TestMain (below) to call main() directly with the
// scripted args. Covers the os.Exit path the in-process tests can't reach.
//
// Build the binary fresh — re-execing the test binary works but we'd lose
// the version-flag short-circuit since flag.CommandLine isn't isolated.
func TestMainBinaryHandlesBadFlag(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "voicetel-cli-test")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	// Run with an unknown flag — main() should exit non-zero.
	cmd = exec.Command(bin, "--nosuchflag")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Errorf("expected non-zero exit, got success. output: %s", out)
	}
}

func TestMainBinaryPrintsVersion(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "voicetel-cli-test")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	cmd = exec.Command(bin, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "voicetel-cli") {
		t.Errorf("expected version output, got: %s", out)
	}
}

// --- sanity: package can import its deps ------------------------------

func TestPackageImports(t *testing.T) {
	if _, err := repl.Parse("noop"); err != nil {
		t.Errorf("repl.Parse roundtrip: %v", err)
	}
}
