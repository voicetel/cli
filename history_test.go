package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSensitiveCommand(t *testing.T) {
	sensitive := []string{
		"login 1000000001 hunter2",
		"login",
		"  login   1000000001   hunter2  ",
		"\tlogin 1000000001 hunter2",
		"set api-key abcdef0123456789abcdef0123456789",
		"set apikey abcdef0123456789abcdef0123456789",
		"  set   api-key   abcdef",
	}
	for _, line := range sensitive {
		if !sensitiveCommand(line) {
			t.Errorf("%q should be flagged sensitive", line)
		}
	}
	harmless := []string{
		"",
		"help",
		"numbers list",
		"loginhelp",              // prefix collision must not match
		"set base-url https://x", // not credential-bearing
		"set",                    // incomplete
		"account get",
		"messaging send {\"text\":\"login 1 2\"}", // credentials inside JSON bodies are out of scope
	}
	for _, line := range harmless {
		if sensitiveCommand(line) {
			t.Errorf("%q should NOT be flagged sensitive", line)
		}
	}
}

func TestFilterHistory(t *testing.T) {
	got := filterHistory([]string{
		"numbers list",
		"",
		"   ",
		"login 1000000001 hunter2",
		"account get",
		"set api-key abcdef0123456789abcdef0123456789",
	})
	want := []string{"numbers list", "account get"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("filterHistory = %v, want %v", got, want)
	}
}

// prepareHistory on a first run: durable file is created mode 0600 and the
// session shim exists (also 0600) ready for readline.
func TestPrepareHistoryFirstRun(t *testing.T) {
	dir := t.TempDir()
	hist := filepath.Join(dir, ".voicetel", "history") // parent does not exist yet
	session, err := prepareHistory(hist)
	if err != nil {
		t.Fatalf("prepareHistory: %v", err)
	}
	for _, p := range []string{hist, session} {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Errorf("%s mode = %o, want 600", p, perm)
		}
	}
	// finalize removes the shim.
	if err := finalizeHistory(session, hist); err != nil {
		t.Fatalf("finalizeHistory: %v", err)
	}
	if _, err := os.Stat(session); !os.IsNotExist(err) {
		t.Errorf("session shim should be removed after finalize: %v", err)
	}
}

// A durable file left world-readable and credential-bearing (older versions,
// or a crash) is scrubbed and locked down on startup.
func TestPrepareHistoryScrubsLegacyFile(t *testing.T) {
	dir := t.TempDir()
	hist := filepath.Join(dir, "history")
	legacy := "numbers list\nlogin 1000000001 hunter2\naccount get\nset api-key abcdef0123456789abcdef0123456789\n"
	if err := os.WriteFile(hist, []byte(legacy), 0o644); err != nil { //nolint:gosec // 0644 is the point of the test
		t.Fatal(err)
	}
	session, err := prepareHistory(hist)
	if err != nil {
		t.Fatalf("prepareHistory: %v", err)
	}
	defer func() { _ = os.Remove(session) }()

	data, err := os.ReadFile(hist)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "hunter2") || strings.Contains(string(data), "abcdef0123456789") {
		t.Errorf("durable history still carries credentials: %q", data)
	}
	if !strings.Contains(string(data), "numbers list") || !strings.Contains(string(data), "account get") {
		t.Errorf("harmless lines lost: %q", data)
	}
	fi, err := os.Stat(hist)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("durable mode = %o, want 600", perm)
	}
	// The shim is seeded with the scrubbed history so ↑ reaches yesterday.
	seed, err := os.ReadFile(session)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(seed), "hunter2") || !strings.Contains(string(seed), "numbers list") {
		t.Errorf("session shim seeded wrong: %q", seed)
	}
}

// finalizeHistory: lines readline captured during the session are filtered
// before they become the durable history.
func TestFinalizeHistoryFiltersSession(t *testing.T) {
	dir := t.TempDir()
	hist := filepath.Join(dir, "history")
	session, err := prepareHistory(hist)
	if err != nil {
		t.Fatalf("prepareHistory: %v", err)
	}
	// Simulate readline appending committed lines during the session.
	if err := os.WriteFile(session, []byte("numbers list\nlogin 1000000001 hunter2\naccount credits\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := finalizeHistory(session, hist); err != nil {
		t.Fatalf("finalizeHistory: %v", err)
	}
	data, err := os.ReadFile(hist)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "hunter2") {
		t.Errorf("durable history carries credentials after finalize: %q", data)
	}
	if !strings.Contains(string(data), "numbers list") || !strings.Contains(string(data), "account credits") {
		t.Errorf("session lines lost: %q", data)
	}
	if _, err := os.Stat(session); !os.IsNotExist(err) {
		t.Errorf("session shim should be gone: %v", err)
	}
}

// prepareHistory against an impossible parent path (a regular file where the
// directory should be) must fail — runLoop refuses to start rather than run
// without history hardening.
func TestPrepareHistoryFailsOnBlockedParent(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareHistory(filepath.Join(blocker, "history")); err == nil {
		t.Fatal("expected error when the history dir is a regular file")
	}
}

// Stale shims from crashed sessions are cleaned up on the next start.
func TestPrepareHistoryRemovesStaleShims(t *testing.T) {
	dir := t.TempDir()
	hist := filepath.Join(dir, "history")
	stale := filepath.Join(dir, ".history-session-999")
	if err := os.WriteFile(stale, []byte("login 1 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	session, err := prepareHistory(hist)
	if err != nil {
		t.Fatalf("prepareHistory: %v", err)
	}
	defer func() { _ = os.Remove(session) }()
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale shim should be removed: %v", err)
	}
}
