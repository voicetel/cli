package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromMissing(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "nope.toml")
	_, err := LoadFrom(p)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	in := &Config{APIKey: "deadbeef" + "deadbeef" + "deadbeef" + "deadbeef", BaseURL: "https://sandbox.voicetel.com"}
	if err := SaveTo(p, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := LoadFrom(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if out.APIKey != in.APIKey {
		t.Errorf("api key: got %q want %q", out.APIKey, in.APIKey)
	}
	if out.BaseURL != in.BaseURL {
		t.Errorf("base url: got %q want %q", out.BaseURL, in.BaseURL)
	}

	// Permissions: 0600.
	st, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if mode := st.Mode().Perm(); mode != 0o600 {
		t.Errorf("mode = %o, want 0600", mode)
	}
}

func TestLoadFromDefaultsBaseURL(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(p, []byte("api_key = \"abc\"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := LoadFrom(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if out.BaseURL != DefaultBaseURL {
		t.Errorf("base url default: got %q want %q", out.BaseURL, DefaultBaseURL)
	}
}

func TestLoadFromMalformed(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(p, []byte("not = valid = toml ===\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := LoadFrom(p)
	if err == nil {
		t.Fatal("expected error from malformed TOML")
	}
}

func TestSaveCreatesMissingParent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "nested", "deeper", "config.toml")
	if err := SaveTo(p, &Config{APIKey: "x"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("stat: %v", err)
	}
}

func TestLoadFromIsAtomic(t *testing.T) {
	// Writing twice should not leave a temp file behind.
	dir := t.TempDir()
	p := filepath.Join(dir, "c.toml")
	for i := 0; i < 3; i++ {
		if err := SaveTo(p, &Config{APIKey: "k"}); err != nil {
			t.Fatal(err)
		}
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if e.Name() != "c.toml" {
			t.Errorf("unexpected leftover file: %s", e.Name())
		}
	}
}

func TestDirAndPathRespectsHome(t *testing.T) {
	t.Setenv("HOME", "/tmp/voicetel-test-home")
	d, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if d != "/tmp/voicetel-test-home/.voicetel" {
		t.Errorf("Dir = %q", d)
	}
	p, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if p != "/tmp/voicetel-test-home/.voicetel/config.toml" {
		t.Errorf("Path = %q", p)
	}
	h, err := HistoryPath()
	if err != nil {
		t.Fatal(err)
	}
	if h != "/tmp/voicetel-test-home/.voicetel/history" {
		t.Errorf("HistoryPath = %q", h)
	}
}
