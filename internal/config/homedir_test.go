package config

import (
	"errors"
	"strings"
	"testing"
)

// Swap out the userHomeDir function to force the error path through Dir(),
// Path(), HistoryPath(), Save(), and Load() — the branches that depend on
// os.UserHomeDir failing.
func TestDirPropagatesHomeDirError(t *testing.T) {
	orig := userHomeDir
	userHomeDir = func() (string, error) { return "", errors.New("synthetic home failure") }
	defer func() { userHomeDir = orig }()

	if _, err := Dir(); err == nil || !strings.Contains(err.Error(), "synthetic home failure") {
		t.Errorf("Dir: expected wrapped error, got %v", err)
	}
	if _, err := Path(); err == nil {
		t.Error("Path: expected error when Dir fails")
	}
	if _, err := HistoryPath(); err == nil {
		t.Error("HistoryPath: expected error when Dir fails")
	}
	if err := Save(&Config{APIKey: "k"}); err == nil {
		t.Error("Save: expected error when Path fails")
	}
	if _, err := Load(); err == nil {
		t.Error("Load: expected error when Path fails")
	}
}
