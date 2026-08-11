package main

import (
	"errors"
	"testing"
)

func TestRunUpdateAcceptsOnlyLatestAndRunsSynchronously(t *testing.T) {
	var migrated, updated bool
	err := executeUpdate([]string{"--latest"}, func() { migrated = true }, func() error {
		updated = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !migrated || !updated {
		t.Fatalf("migrated=%v updated=%v", migrated, updated)
	}
}

func TestRunUpdateRejectsUnsupportedArguments(t *testing.T) {
	for _, args := range [][]string{nil, {}, {"--other"}, {"--latest", "extra"}} {
		called := false
		if err := executeUpdate(args, func() { called = true }, func() error { called = true; return nil }); err == nil {
			t.Fatalf("executeUpdate(%q) error = nil", args)
		}
		if called {
			t.Fatalf("executeUpdate(%q) invoked dependencies", args)
		}
	}
}

func TestRunUpdateReturnsUpgradeFailure(t *testing.T) {
	want := errors.New("upgrade failed")
	if err := executeUpdate([]string{"--latest"}, func() {}, func() error { return want }); !errors.Is(err, want) {
		t.Fatalf("executeUpdate error = %v, want %v", err, want)
	}
}
