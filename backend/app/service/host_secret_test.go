package service

import (
	"testing"

	"xpanel/app/dto"
)

func TestHostUpdateOmitsEmptyCredentials(t *testing.T) {
	fields := buildHostUpdates(dto.HostUpdate{
		ID:         1,
		Name:       "server",
		Addr:       "127.0.0.1",
		Port:       22,
		User:       "root",
		AuthMode:   "password",
		Password:   "",
		PrivateKey: "",
		PassPhrase: "",
	})
	for _, key := range []string{"password", "private_key", "pass_phrase"} {
		if _, exists := fields[key]; exists {
			t.Fatalf("empty credential %s should not be included: %#v", key, fields)
		}
	}
}

func TestHostUpdateIncludesReplacementCredentials(t *testing.T) {
	fields := buildHostUpdates(dto.HostUpdate{
		Password:   "password",
		PrivateKey: "private-key",
		PassPhrase: "passphrase",
	})
	for key, want := range map[string]string{
		"password":    "password",
		"private_key": "private-key",
		"pass_phrase": "passphrase",
	} {
		if got := fields[key]; got != want {
			t.Fatalf("%s = %#v, want %q", key, got, want)
		}
	}
}
