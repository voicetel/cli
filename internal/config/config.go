// Package config loads and persists the CLI's per-user configuration.
//
// The on-disk format is TOML at ~/.voicetel/config.toml. The file is
// written atomically (write to a temp sibling, fsync, rename) so we
// never leave a half-written config behind, and is locked down to 0600.
//
// The password is never persisted. Only the API key (a bearer token) is.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// DefaultBaseURL is the production VoiceTel API endpoint. Stored as a
// constant rather than imported from the SDK to keep this package
// dependency-free.
const DefaultBaseURL = "https://api.voicetel.com"

// Config is the persisted shape.
type Config struct {
	APIKey  string `toml:"api_key,omitempty"`
	BaseURL string `toml:"base_url,omitempty"`
}

// ErrNotFound is returned by Load when the file does not exist.
var ErrNotFound = errors.New("config: file does not exist")

// userHomeDir is a function variable so tests can simulate the
// `os.UserHomeDir` error path (which is otherwise hard to trigger on a
// normal dev/CI machine where HOME is always set).
var userHomeDir = os.UserHomeDir

// Dir returns ~/.voicetel for the current user. Exported for tests.
func Dir() (string, error) {
	home, err := userHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: %w", err)
	}
	return filepath.Join(home, ".voicetel"), nil
}

// Path returns the absolute path to config.toml.
func Path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.toml"), nil
}

// HistoryPath returns the absolute path to the readline history file.
func HistoryPath() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "history"), nil
}

// LoadFrom reads and parses a config file from an explicit path.
// Returns ErrNotFound if the file is missing.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // user-owned path
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	c := &Config{}
	if err := toml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	return c, nil
}

// Load reads the default ~/.voicetel/config.toml. If the file is
// missing, a Config with defaults is returned alongside ErrNotFound so
// callers can distinguish "first run" from "real error".
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	c, err := LoadFrom(p)
	if errors.Is(err, ErrNotFound) {
		return &Config{BaseURL: DefaultBaseURL}, ErrNotFound
	}
	return c, err
}

// SaveTo writes the config to an explicit path atomically. Creates the
// parent directory with 0700 if missing.
func SaveTo(path string, c *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", dir, err)
	}
	data, err := toml.Marshal(c) //nolint:gosec // API key is intentionally persisted; this is its file.
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	// Atomic write: temp sibling -> fsync -> rename.
	tmp, err := os.CreateTemp(dir, ".config-*.toml.tmp")
	if err != nil {
		return fmt.Errorf("config: temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("config: write: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("config: sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("config: close: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		cleanup()
		return fmt.Errorf("config: chmod: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("config: rename: %w", err)
	}
	return nil
}

// Save writes the config to ~/.voicetel/config.toml atomically.
func Save(c *Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	return SaveTo(p, c)
}
