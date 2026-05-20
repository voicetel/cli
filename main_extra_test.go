package main

import (
	"errors"
	"os"
	"testing"
)

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
	main()
	if !called {
		t.Fatal("exitOnErr was not called")
	}
	if got != nil {
		t.Errorf("--version: want nil err, got %v", got)
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
	main()
	if !called {
		t.Fatal("exitOnErr was not called")
	}
	if got == nil {
		t.Error("--nosuchflag: want non-nil err, got nil")
	}
}

// exitOnErr(nil) is a no-op (no os.Exit call). The default implementation
// only exits on a non-nil error; verifying nil works exercises that branch.
func TestExitOnErrNilIsNoOp(t *testing.T) {
	exitOnErr(nil) // would os.Exit(1) if it didn't gate on err != nil
}

// Sanity that the package-level variable is reassignable (important for
// the override pattern the other tests use).
func TestExitOnErrIsAVar(t *testing.T) {
	orig := exitOnErr
	defer func() { exitOnErr = orig }()
	called := false
	exitOnErr = func(err error) {
		called = true
		if !errors.Is(err, errSynthetic) {
			t.Errorf("err wasn't our sentinel: %v", err)
		}
	}
	exitOnErr(errSynthetic)
	if !called {
		t.Error("override didn't fire")
	}
}

var errSynthetic = errors.New("synthetic")
