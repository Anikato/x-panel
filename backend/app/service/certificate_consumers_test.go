package service

import (
	"errors"
	"reflect"
	"testing"
)

func TestRefreshCertificateConsumersRunsRemainingTargetsAfterFailure(t *testing.T) {
	var calls []string
	err := refreshCertificateConsumers(certificateConsumerTargets{
		Nginx:   true,
		HAProxy: true,
		GOST:    true,
	}, certificateConsumerRefreshActions{
		ReloadNginx: func() error {
			calls = append(calls, "nginx")
			return nil
		},
		ReloadHAProxy: func() error {
			calls = append(calls, "haproxy")
			return errors.New("reload failed")
		},
		ReloadGOST: func() error {
			calls = append(calls, "gost")
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected a consumer refresh error")
	}
	if want := []string{"nginx", "haproxy", "gost"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("expected all selected consumers to be attempted, got %v", calls)
	}
}

func TestCertificateSyncPostActionsRefreshBeforeCustomCommand(t *testing.T) {
	var calls []string
	refreshErr := errors.New("refresh failed")
	err := runCertificateSyncPostActions([]uint{7}, "custom command",
		func(ids []uint) error {
			if !reflect.DeepEqual(ids, []uint{7}) {
				t.Fatalf("unexpected certificate IDs: %v", ids)
			}
			calls = append(calls, "refresh")
			return refreshErr
		},
		func(command string) error {
			if command != "custom command" {
				t.Fatalf("unexpected command: %s", command)
			}
			calls = append(calls, "command")
			return nil
		},
	)
	if !errors.Is(err, refreshErr) {
		t.Fatalf("expected refresh error, got %v", err)
	}
	if want := []string{"refresh", "command"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("expected refresh before command, got %v", calls)
	}
}

func TestCertificateSyncStatusKeepsRefreshFailureRetryable(t *testing.T) {
	status := certificateSyncStatus(1, 0, 0, errors.New("consumer refresh failed"))
	if status != "warning" {
		t.Fatalf("expected refresh failure to remain retryable as warning, got %q", status)
	}
}
