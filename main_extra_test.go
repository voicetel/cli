package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdio points os.Stdout and os.Stderr at temp files while fn runs.
// main() writes to the process streams directly, so this keeps its output
// out of the test log and lets the caller assert on it.
func captureStdio(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	dir := t.TempDir()
	outF, err := os.CreateTemp(dir, "stdout-*")
	if err != nil {
		t.Fatalf("create stdout capture: %v", err)
	}
	defer func() { _ = outF.Close() }()
	errF, err := os.CreateTemp(dir, "stderr-*")
	if err != nil {
		t.Fatalf("create stderr capture: %v", err)
	}
	defer func() { _ = errF.Close() }()

	func() {
		origOut, origErr := os.Stdout, os.Stderr
		defer func() { os.Stdout, os.Stderr = origOut, origErr }()
		os.Stdout, os.Stderr = outF, errF
		fn()
	}()

	read := func(f *os.File) string {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			t.Fatalf("rewind %s: %v", f.Name(), err)
		}
		b, err := io.ReadAll(f)
		if err != nil {
			t.Fatalf("read %s: %v", f.Name(), err)
		}
		return string(b)
	}
	return read(outF), read(errF)
}

// main() with --version invokes exitOnErr(nil). Override exitOnErr so the
// test process doesn't actually exit, and assert it's called with nil.
func TestMainVersionExitsZero(t *testing.T) {
	isolatedHome(t)
	orig := exitOnErr
	origArgs := os.Args
	defer func() { exitOnErr = orig; os.Args = origArgs }()

	os.Args = []string{"voicetel-cli", "--version"}
	var got error
	called := false
	exitOnErr = func(err error) {
		called = true
		got = err
	}
	stdout, _ := captureStdio(t, main)
	if !called {
		t.Fatal("exitOnErr was not called")
	}
	if got != nil {
		t.Errorf("--version: want nil err, got %v", got)
	}
	if !strings.Contains(stdout, "voicetel-cli "+Version) {
		t.Errorf("--version stdout = %q, want the version banner", stdout)
	}
}

// main() with an unknown flag invokes exitOnErr(err). Confirm the wrapper
// surfaces a non-nil error.
func TestMainBadFlagExitsNonZero(t *testing.T) {
	isolatedHome(t)
	orig := exitOnErr
	origArgs := os.Args
	defer func() { exitOnErr = orig; os.Args = origArgs }()

	os.Args = []string{"voicetel-cli", "--nosuchflag"}
	var got error
	called := false
	exitOnErr = func(err error) {
		called = true
		got = err
	}
	_, stderr := captureStdio(t, main)
	if !called {
		t.Fatal("exitOnErr was not called")
	}
	if got == nil {
		t.Error("--nosuchflag: want non-nil err, got nil")
	}
	if !strings.Contains(stderr, "flag provided but not defined: -nosuchflag") {
		t.Errorf("--nosuchflag stderr = %q, want the flag parse error", stderr)
	}
}

// exitOnErr(nil) is a no-op (no os.Exit call). The default implementation
// only exits on a non-nil error; verifying nil works exercises that branch.
// The reassignability of exitOnErr itself is covered by
// TestMainVersionExitsZero / TestMainBadFlagExitsNonZero above, which
// override it and assert the override fires.
func TestExitOnErrNilIsNoOp(t *testing.T) {
	exitOnErr(nil) // would os.Exit(1) if it didn't gate on err != nil
}
