package main

import "testing"

func TestRunDispatchesMigrateWithoutStartingServer(t *testing.T) {
	var started, migrated bool
	run([]string{"migrate"}, func() { started = true }, func() { migrated = true }, func([]string) {}, func() {})
	if started || !migrated {
		t.Fatalf("started=%v migrated=%v, want started=false migrated=true", started, migrated)
	}
}

func TestRunDispatchesSetupArguments(t *testing.T) {
	var got []string
	run([]string{"setup", "--username", "admin"}, func() {}, func() {}, func(args []string) { got = args }, func() {})
	if len(got) != 2 || got[0] != "--username" || got[1] != "admin" {
		t.Fatalf("setup args = %#v", got)
	}
}

func TestRunDispatchesVersionWithoutStartingServer(t *testing.T) {
	var started, printed bool
	run([]string{"--version"}, func() { started = true }, func() {}, func([]string) {}, func() { printed = true })
	if started || !printed {
		t.Fatalf("started=%v printed=%v, want started=false printed=true", started, printed)
	}
}
