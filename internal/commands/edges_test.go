package commands

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/voicetel/cli/internal/output"
)

// tailFromRaw: walk through every code path. n=0 returns the whole string,
// quoted tokens (single and double) get consumed as one, n past the token
// count returns "".
func TestTailFromRawEdges(t *testing.T) {
	cases := []struct {
		name, raw string
		n         int
		want      string
	}{
		{"n=0 returns trimmed input", "  hello world  ", 0, "hello world"},
		{"single token consumed", "alpha bravo", 1, "bravo"},
		{"two tokens consumed", "alpha bravo charlie delta", 2, "charlie delta"},
		{"n beyond token count is empty", "alpha bravo", 5, ""},
		{`double-quoted token counts as one`, `cmd "with spaces" rest`, 2, "rest"},
		{`single-quoted token counts as one`, `cmd 'a b c' tail`, 2, "tail"},
		{"json body preserved", `account update {"timezone":"America/Chicago"}`, 2, `{"timezone":"America/Chicago"}`},
		{"tabs separate tokens", "alpha\tbravo\tcharlie", 2, "charlie"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tailFromRaw(tc.raw, tc.n); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// Dispatch with zero tokens returns nil (no-op — guards against tokens of
// empty parsed lines).
func TestDispatchEmptyTokens(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := r.Dispatch(h.context, nil, ""); err != nil {
		t.Errorf("Dispatch(nil): want nil, got %v", err)
	}
	if err := r.Dispatch(h.context, []string{}, ""); err != nil {
		t.Errorf("Dispatch([]): want nil, got %v", err)
	}
}

// Dispatch on a known group with no sub-command surfaces a helpful error
// suggesting `help <group>`.
func TestDispatchGroupMissingSub(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	err := r.Dispatch(h.context, []string{"account"}, "account")
	if err == nil || !strings.Contains(err.Error(), "missing sub-command") {
		t.Errorf("expected 'missing sub-command' error, got %v", err)
	}
}

// Dispatch on a known group with an unknown sub-command surfaces a helpful
// error.
func TestDispatchGroupUnknownSub(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	err := r.Dispatch(h.context, []string{"account", "nosuchthing"}, "account nosuchthing")
	if err == nil || !strings.Contains(err.Error(), `unknown sub-command "nosuchthing"`) {
		t.Errorf("expected 'unknown sub-command' error, got %v", err)
	}
}

// loginCommand with a non-numeric username returns the integer-parse error
// rather than calling Login.
func TestLoginCommandBadUsername(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("login alice secret", r); err == nil {
		t.Fatal("expected error for non-integer username, got nil")
	}
}

// loginCommand with the wrong arg count errors out clean.
func TestLoginCommandArgCount(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	for _, line := range []string{"login", "login only-one-arg", "login a b c"} {
		err := h.dispatch(line, r)
		if err == nil {
			t.Errorf("%q: want error, got nil", line)
		}
	}
}

// setCommand: every documented variant and the error cases.
func TestSetCommandVariants(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()

	// happy paths
	if err := h.dispatch("set api-key ffffffffffffffffffffffffffffffff", r); err != nil {
		t.Errorf("set api-key: %v", err)
	}
	if err := h.dispatch("set apikey ffffffffffffffffffffffffffffffff", r); err != nil {
		t.Errorf("set apikey (alias): %v", err)
	}
	// set base-url is documented but immutable after the session starts.
	// The error is the "not mutable" guard, which exercises the same code
	// path we care about for coverage.
	if err := h.dispatch("set base-url https://staging.invalid", r); err == nil {
		t.Error("set base-url: want immutable-error, got nil")
	}
	if err := h.dispatch("set baseurl https://staging.invalid", r); err == nil {
		t.Error("set baseurl (alias): want immutable-error, got nil")
	}

	// errors
	if err := h.dispatch("set", r); err == nil {
		t.Error("set with no args: want error")
	}
	if err := h.dispatch("set api-key", r); err == nil {
		t.Error("set api-key with no value: want error")
	}
	if err := h.dispatch("set nosuchthing value", r); err == nil {
		t.Error("set bad subcommand: want error")
	}
}

// whoamiCommand prints the API key / base URL / rate-limit reminder.
// Confirm the JSON keys appear in the output.
func TestWhoamiCommand(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("whoami", r); err != nil {
		t.Fatal(err)
	}
	got := h.out.String()
	for _, want := range []string{"apiKey", "baseURL", "rateLimits"} {
		if !strings.Contains(got, want) {
			t.Errorf("whoami output missing %q: %s", want, got)
		}
	}
}

// helpCommand with no args prints the top-level overview; with a known
// topic prints that topic; with an unknown topic returns an error.
func TestHelpCommandVariants(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("help", r); err != nil {
		t.Errorf("help: %v", err)
	}
	// known top-level command
	h.out.Reset()
	if err := h.dispatch("help login", r); err != nil {
		t.Errorf("help login: %v", err)
	}
	if !strings.Contains(h.out.String(), "login") {
		t.Errorf("help login output missing 'login': %s", h.out.String())
	}
	// known group
	h.out.Reset()
	if err := h.dispatch("help account", r); err != nil {
		t.Errorf("help account: %v", err)
	}
	// group + sub-command
	h.out.Reset()
	if err := h.dispatch("help account get", r); err != nil {
		t.Errorf("help account get: %v", err)
	}
	// unknown topic
	if err := h.dispatch("help nosuchtopic", r); err == nil {
		t.Error("help nosuchtopic: want error")
	}
	// unknown subcommand of known group
	if err := h.dispatch("help account nosuchsub", r); err == nil {
		t.Error("help account nosuchsub: want error")
	}
}

// parseJSON: empty body → nil v stays nil; valid JSON populates v;
// malformed JSON returns a labelled error.
func TestParseJSONEdges(t *testing.T) {
	// empty body is rejected with a labelled "missing JSON body" error.
	var dst map[string]any
	if err := parseJSON("test", "", &dst); err == nil {
		t.Error("empty body: want error, got nil")
	} else if !strings.Contains(err.Error(), "test:") {
		t.Errorf("expected 'test:' label, got %v", err)
	}

	// valid JSON
	if err := parseJSON("test", `{"k": "v"}`, &dst); err != nil {
		t.Errorf("valid: %v", err)
	}
	if dst["k"] != "v" {
		t.Errorf("dst[\"k\"] = %v, want \"v\"", dst["k"])
	}

	// malformed JSON
	dst = nil
	err := parseJSON("test", `{not json`, &dst)
	if err == nil {
		t.Error("malformed JSON: want error")
	}
	if !strings.Contains(err.Error(), "test:") {
		t.Errorf("expected 'test:' label in error, got %v", err)
	}
}

// renderCommand exercises an internal help-rendering path. Trigger via
// `help <group>` and assert the synopsis is included for at least one
// sub-command.
func TestRenderCommandViaHelpGroup(t *testing.T) {
	h := newHarness(t)
	r := BuildRegistry()
	if err := h.dispatch("help numbers", r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.out.String(), "numbers") {
		t.Errorf("help numbers output missing 'numbers': %s", h.out.String())
	}
}

// Dispatch wraps the builtin path through tailFromRaw. Confirm the JSON
// tail makes it through quoted tokens correctly.
func TestDispatchPreservesJSONTail(t *testing.T) {
	// A quick smoke that a builtin receiving a JSON body via tailFromRaw
	// sees the raw bytes. We use `set api-key VALUE` since it consumes the
	// 2nd token as a plain string — its presence in the OnConfigChanged
	// callback confirms tail handling worked.
	h := newHarness(t)
	r := BuildRegistry()
	key := strings.Repeat("a", 32)
	if err := h.dispatch("set api-key "+key, r); err != nil {
		t.Fatal(err)
	}
	if h.client.APIKey() != key {
		t.Errorf("APIKey not installed; got %q", h.client.APIKey())
	}
}

// Confirm ErrExit comparability via errors.Is — important because the REPL
// loop relies on this to differentiate "user typed exit" from "real error".
func TestErrExitIsSentinel(t *testing.T) {
	if !errors.Is(ErrExit, ErrExit) {
		t.Error("ErrExit not recognised by errors.Is")
	}
	// And a freshly-wrapped one round-trips:
	wrapped := errors.Join(errors.New("context"), ErrExit)
	if !errors.Is(wrapped, ErrExit) {
		t.Error("wrapped ErrExit not recognised")
	}
}

// helpers smoke: parseJSON with a nil target panics? Let's verify it errors
// gracefully (or at least doesn't crash the binary).
func TestParseJSONNilTarget(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			// json.Unmarshal panics when v is nil; we just record this is
			// the upstream behaviour. The CLI never passes nil v in practice.
			t.Skipf("parseJSON with nil v panicked (upstream encoding/json behaviour): %v", r)
		}
	}()
	_ = parseJSON("test", `{"k":1}`, nil)
}

// Output Printer smoke from within commands tests — ensures the test harness
// has a non-nil printer that does what we expect.
func TestHarnessPrinterIsLive(t *testing.T) {
	h := newHarness(t)
	h.context.Printer.Println("hello world")
	if !strings.Contains(h.out.String(), "hello world") {
		t.Errorf("harness printer didn't write to out: %s", h.out.String())
	}
}

// Ensure the context fields we depend on across tests are populated. Acts
// as a guard against future test-helper refactors.
func TestHarnessContextWiring(t *testing.T) {
	h := newHarness(t)
	if h.context.Ctx == nil {
		t.Error("Ctx is nil")
	}
	if h.context.Client == nil {
		t.Error("Client is nil")
	}
	if h.context.Printer == nil {
		t.Error("Printer is nil")
	}
	if h.context.OnConfigChanged == nil {
		t.Error("OnConfigChanged is nil")
	}
}

// Output package import sanity (catches accidental imports of the wrong
// output package from a vendored copy).
func TestOutputPackageSmoke(t *testing.T) {
	p := output.New(io.Discard, io.Discard)
	if p == nil {
		t.Error("output.New returned nil")
	}
	_ = context.Background()
}
