package service

import (
	"errors"
	"testing"
)

func TestPersistCertificateUpdateReturnsSensitiveRepositoryError(t *testing.T) {
	sentinel := errors.New("injected certificate encryption failure")
	err := persistCertificateUpdate(
		func(uint, map[string]interface{}) error { return sentinel },
		7,
		map[string]interface{}{"private_key": "private-key"},
	)
	if !errors.Is(err, sentinel) {
		t.Fatalf("persist certificate error = %v, want %v", err, sentinel)
	}
}

func TestPersistInstancePasswordReturnsSensitiveRepositoryError(t *testing.T) {
	sentinel := errors.New("injected database password encryption failure")
	err := persistInstancePassword(
		func(uint, map[string]interface{}) error { return sentinel },
		9,
		"database-password",
	)
	if !errors.Is(err, sentinel) {
		t.Fatalf("persist instance password error = %v, want %v", err, sentinel)
	}
}

func TestPersistSettingValuesStopsOnSensitiveRepositoryError(t *testing.T) {
	sentinel := errors.New("injected setting encryption failure")
	calls := 0
	err := persistSettingValues(
		func(key, value string) error {
			calls++
			if key == "Secret" {
				return sentinel
			}
			return nil
		},
		settingValue{Key: "Ordinary", Value: "value"},
		settingValue{Key: "Secret", Value: "secret"},
		settingValue{Key: "AfterFailure", Value: "must-not-run"},
	)
	if !errors.Is(err, sentinel) {
		t.Fatalf("persist settings error = %v, want %v", err, sentinel)
	}
	if calls != 2 {
		t.Fatalf("setting writes after failure = %d calls, want 2", calls)
	}
}
