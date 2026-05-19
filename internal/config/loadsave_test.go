package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadDefaults uses HOME to redirect Load to a tempdir, then verifies
// the file-missing branch returns ErrNotFound + defaulted base url.
func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	c, err := Load()
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if c.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
}

func TestSaveRoundTripUnderHOME(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	in := &Config{APIKey: "abc", BaseURL: "https://example.com"}
	if err := Save(in); err != nil {
		t.Fatal(err)
	}
	out, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if out.APIKey != in.APIKey || out.BaseURL != in.BaseURL {
		t.Errorf("round-trip differs: %+v vs %+v", in, out)
	}
}

func TestSaveToOverwrite(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.toml")
	if err := SaveTo(p, &Config{APIKey: "a"}); err != nil {
		t.Fatal(err)
	}
	if err := SaveTo(p, &Config{APIKey: "b"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p) //nolint:gosec // test reads its own file
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(data), "api_key", "b") {
		t.Errorf("expected file to reflect second save, got %q", string(data))
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
