package main

import "testing"

func TestRunDispatchesMigrateWithoutStartingServer(t *testing.T) {
	var started, migrated bool
	run(
		[]string{"migrate"},
		func() { started = true },
		func() { migrated = true },
		func([]string) {},
		func([]string) {},
		func([]string) {},
		func() {},
		func([]string) error { return nil },
		func([]string) {},
	)
	if started || !migrated {
		t.Fatalf("started=%v migrated=%v, want started=false migrated=true", started, migrated)
	}
}

func TestRunDispatchesSetupArguments(t *testing.T) {
	var got []string
	run(
		[]string{"setup", "--username", "admin"},
		func() {},
		func() {},
		func(args []string) { got = args },
		func([]string) {},
		func([]string) {},
		func() {},
		func([]string) error { return nil },
		func([]string) {},
	)
	if len(got) != 2 || got[0] != "--username" || got[1] != "admin" {
		t.Fatalf("setup args = %#v", got)
	}
}

func TestRunDispatchesVersionWithoutStartingServer(t *testing.T) {
	var started, printed bool
	run(
		[]string{"--version"},
		func() { started = true },
		func() {},
		func([]string) {},
		func([]string) {},
		func([]string) {},
		func() { printed = true },
		func([]string) error { return nil },
		func([]string) {},
	)
	if started || !printed {
		t.Fatalf("started=%v printed=%v, want started=false printed=true", started, printed)
	}
}

func TestRunDispatchesBootstrapConfigArguments(t *testing.T) {
	var got []string
	run(
		[]string{"bootstrap-config"},
		func() {},
		func() {},
		func([]string) {},
		func(args []string) { got = args },
		func([]string) {},
		func() {},
		func([]string) error { return nil },
		func([]string) {},
	)
	if got == nil || len(got) != 0 {
		t.Fatalf("bootstrap-config args = %#v", got)
	}
}

func TestRunDispatchesCredentialsArguments(t *testing.T) {
	var got []string
	run(
		[]string{"credentials", "verify", "--db", "/tmp/backup.db"},
		func() {},
		func() {},
		func([]string) {},
		func([]string) {},
		func(args []string) { got = args },
		func() {},
		func([]string) error { return nil },
		func([]string) {},
	)
	if len(got) != 3 || got[0] != "verify" || got[1] != "--db" || got[2] != "/tmp/backup.db" {
		t.Fatalf("credentials args = %#v", got)
	}
}

func TestRunDispatchesUpdateLatestWithoutStartingServer(t *testing.T) {
	var started bool
	var got []string
	err := run(
		[]string{"update", "--latest"},
		func() { started = true },
		func() {},
		func([]string) {},
		func([]string) {},
		func([]string) {},
		func() {},
		func(args []string) error {
			got = append([]string(nil), args...)
			return nil
		},
		func([]string) {},
	)
	if err != nil {
		t.Fatal(err)
	}
	if started || len(got) != 1 || got[0] != "--latest" {
		t.Fatalf("started=%v update args=%#v", started, got)
	}
}

func TestRunDispatchesInvokeWithoutStartingServer(t *testing.T) {
	var started bool
	var got []string
	err := run(
		[]string{"invoke"},
		func() { started = true },
		func() {},
		func([]string) {},
		func([]string) {},
		func([]string) {},
		func() {},
		func([]string) error { return nil },
		func(args []string) { got = append([]string(nil), args...) },
	)
	if err != nil {
		t.Fatal(err)
	}
	if started || len(got) != 0 {
		t.Fatalf("started=%v invoke args=%#v", started, got)
	}
}

func TestRunDispatchesInvokeArguments(t *testing.T) {
	var started bool
	var got []string
	err := run(
		[]string{"invoke", "--json"},
		func() { started = true },
		func() {},
		func([]string) {},
		func([]string) {},
		func([]string) {},
		func() {},
		func([]string) error { return nil },
		func(args []string) { got = append([]string(nil), args...) },
	)
	if err != nil {
		t.Fatal(err)
	}
	if started || len(got) != 1 || got[0] != "--json" {
		t.Fatalf("started=%v invoke args=%#v", started, got)
	}
}

func TestRunUnknownCommandDoesNotStartServer(t *testing.T) {
	var started bool
	err := run(
		[]string{"not-a-command"},
		func() { started = true },
		func() {},
		func([]string) {},
		func([]string) {},
		func([]string) {},
		func() {},
		func([]string) error { return nil },
		func([]string) {},
	)
	if err == nil || started {
		t.Fatalf("err=%v started=%v, want error and started=false", err, started)
	}
}
