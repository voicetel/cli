package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// This file hardens ~/.voicetel/history:
//
//   - The durable history file is always rewritten atomically at mode 0600
//     (readline itself creates it 0666&umask).
//   - Credential-carrying lines — `login <user> <password>` and
//     `set api-key <key>` (plus the `set apikey` alias) — are never persisted.
//
// readline (v1.5.1) appends every committed line to its HistoryFile live, on
// Enter, so pointing it at the durable file directly would put credentials on
// disk for the duration of the session. Instead the REPL points readline at a
// per-session shim file (~/.voicetel/.history-session-*, mode 0600), seeded
// from the durable history at startup; on shutdown the shim is filtered and
// becomes the new durable history.

// sensitiveCommand reports whether a REPL line carries credentials and so
// must never be persisted to the history file.
func sensitiveCommand(line string) bool {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false
	}
	if fields[0] == "login" {
		return true
	}
	if fields[0] == "set" && len(fields) >= 2 && (fields[1] == "api-key" || fields[1] == "apikey") {
		return true
	}
	return false
}

// filterHistory drops blank lines and credential-carrying lines.
func filterHistory(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if strings.TrimSpace(l) == "" || sensitiveCommand(l) {
			continue
		}
		out = append(out, l)
	}
	return out
}

// writeHistory atomically rewrites path with lines (temp sibling + chmod 0600
// + rename), creating the parent directory if needed. The resulting file is
// always mode 0600 regardless of what was there before.
func writeHistory(path string, lines []string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("history: mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".history-*.tmp")
	if err != nil {
		return fmt.Errorf("history: temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }
	content := ""
	if len(lines) > 0 {
		content = strings.Join(lines, "\n") + "\n"
	}
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("history: write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("history: close: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		cleanup()
		return fmt.Errorf("history: chmod: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		cleanup()
		return fmt.Errorf("history: rename: %w", err)
	}
	return nil
}

// readHistoryLines splits a history file into lines; a missing file yields nil.
func readHistoryLines(path string) ([]string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is the CLI's own history file
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("history: read %s: %w", path, err)
	}
	return strings.Split(strings.TrimRight(string(data), "\n"), "\n"), nil
}

// prepareHistory readies the durable history file (rewrites it at mode 0600,
// scrubbing any credential-carrying lines left by older versions or a crash)
// and returns the per-session shim path readline should use, seeded with the
// scrubbed history. Stale shims from crashed sessions are removed.
func prepareHistory(histPath string) (sessionPath string, err error) {
	lines, err := readHistoryLines(histPath)
	if err != nil {
		return "", err
	}
	if err := writeHistory(histPath, filterHistory(lines)); err != nil {
		return "", err
	}
	dir := filepath.Dir(histPath)
	if stale, _ := filepath.Glob(filepath.Join(dir, ".history-session-*")); len(stale) > 0 {
		for _, s := range stale {
			_ = os.Remove(s)
		}
	}
	f, err := os.CreateTemp(dir, ".history-session-*") // mode 0600
	if err != nil {
		return "", fmt.Errorf("history: session file: %w", err)
	}
	if len(lines) > 0 {
		seed := filterHistory(lines)
		if _, err := f.WriteString(strings.Join(seed, "\n") + "\n"); err != nil {
			_ = f.Close()
			_ = os.Remove(f.Name())
			return "", fmt.Errorf("history: seed session: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("history: session file: %w", err)
	}
	return f.Name(), nil
}

// finalizeHistory filters the session shim and makes it the durable history,
// then removes the shim. Credential-carrying lines from the session never
// reach the durable file.
func finalizeHistory(sessionPath, histPath string) error {
	lines, err := readHistoryLines(sessionPath)
	if err != nil {
		return err
	}
	if err := writeHistory(histPath, filterHistory(lines)); err != nil {
		return err
	}
	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("history: remove session: %w", err)
	}
	return nil
}
