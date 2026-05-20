package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// SaveTo with a writable parent succeeds and produces a 0600 file.
func TestSaveToWritesFileMode0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.toml")
	if err := SaveTo(path, &Config{APIKey: "k", BaseURL: "https://x.invalid"}); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if runtime.GOOS != "windows" {
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Errorf("mode = %o, want 0600", mode)
		}
	}
}

// SaveTo with an unwritable parent should bubble the mkdir error up.
// On Windows the test runner often has admin → skip.
func TestSaveToMkdirError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based unwritable-dir tests don't translate to Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses chmod restrictions")
	}
	parent := t.TempDir()
	readonly := filepath.Join(parent, "readonly")
	if err := os.Mkdir(readonly, 0o500); err != nil { // r-x — can't create children
		t.Fatalf("mkdir readonly: %v", err)
	}
	defer func() { _ = os.Chmod(readonly, 0o700) }() // let TempDir clean up

	target := filepath.Join(readonly, "child", "config.toml")
	err := SaveTo(target, &Config{APIKey: "k"})
	if err == nil {
		t.Fatal("SaveTo on readonly: want error, got nil")
	}
	if !strings.Contains(err.Error(), "mkdir") {
		t.Errorf("expected mkdir-wrapped error, got %v", err)
	}
}

// SaveTo: when the path's parent dir IS writable but the target name conflicts
// with an existing directory, os.Rename returns an error → SaveTo wraps it
// as "config: rename".
func TestSaveToRenameError(t *testing.T) {
	dir := t.TempDir()
	// Make "config.toml" exist as a non-empty directory → rename(tmp, dir)
	// fails on most platforms because the destination directory is non-empty.
	target := filepath.Join(dir, "config.toml")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "occupant"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write occupant: %v", err)
	}
	err := SaveTo(target, &Config{APIKey: "k"})
	if err == nil {
		t.Fatal("SaveTo with non-empty-dir target: want error, got nil")
	}
}

// LoadFrom with a malformed TOML file returns a parse error.
func TestLoadFromInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte("this is = not = valid = toml"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadFrom(path)
	if err == nil || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

// LoadFrom with a permission-denied file (not a "not-exist" error) bubbles
// the read error up, wrapped with "config: read".
func TestLoadFromReadError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based unreadable-file tests don't translate to Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses chmod restrictions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("api_key='x'\n"), 0o000); err != nil {
		t.Fatalf("write: %v", err)
	}
	defer func() { _ = os.Chmod(path, 0o600) }() // for cleanup

	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected read error, got nil")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("expected 'read' in error, got %v", err)
	}
}

// LoadFrom of a missing file returns ErrNotFound (the sentinel).
func TestLoadFromMissingReturnsErrNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadFrom(filepath.Join(dir, "absent.toml"))
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

// LoadFrom with empty BaseURL fills in DefaultBaseURL (additional coverage —
// the sibling test in config_test.go covers the no-base_url path; this one
// just sanity-checks that the helper still works when an explicit non-default
// api_key is also present).
func TestLoadFromDefaultsBaseURLWithKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.toml")
	if err := os.WriteFile(path, []byte(`api_key="k"`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	c, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL: want %q, got %q", DefaultBaseURL, c.BaseURL)
	}
}

// Load (no path arg) on a fresh HOME returns ErrNotFound + a defaults-only
// Config.
func TestLoadDefaultsOnFreshHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	c, err := Load()
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
	if c == nil || c.BaseURL != DefaultBaseURL {
		t.Errorf("expected default BaseURL, got %+v", c)
	}
}

// Load on a TOML file that parses but unrelated read error bubbles up
// non-ErrNotFound. Hit via Load() → LoadFrom() with a malformed file.
func TestLoadPropagatesNonNotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	// Pre-populate ~/.voicetel/config.toml with malformed content.
	dir := filepath.Join(tmp, ".voicetel")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("bad = = ="), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load()
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("expected non-ErrNotFound parse error, got %v", err)
	}
}

// Save (no path arg) writes to ~/.voicetel/config.toml when HOME is set.
func TestSaveDefaultPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	if err := Save(&Config{APIKey: "k", BaseURL: "https://x.invalid"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".voicetel", "config.toml")); err != nil {
		t.Errorf("config.toml not written: %v", err)
	}
}

// HistoryPath returns ~/.voicetel/history under a real HOME.
func TestHistoryPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	p, err := HistoryPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, ".voicetel", "history")
	if p != want {
		t.Errorf("HistoryPath: want %q, got %q", want, p)
	}
}
