package service

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"xpanel/app/model"
)

func TestCertificateRefreshRetryDue(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	if certificateRefreshRetryDue(model.CertSource{}, now) {
		t.Fatal("source without pending refresh is due")
	}
	pending := now.Add(-certificateRefreshRetryInterval + time.Second)
	if certificateRefreshRetryDue(model.CertSource{RefreshPendingAt: &pending}, now) {
		t.Fatal("refresh became due before retry interval")
	}
	pending = now.Add(-certificateRefreshRetryInterval)
	if !certificateRefreshRetryDue(model.CertSource{RefreshPendingAt: &pending}, now) {
		t.Fatal("refresh is not due at retry interval")
	}
}

func TestRetryPendingCertificateRefreshSuccess(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	lastSync := now.Add(-time.Hour)
	pending := now.Add(-certificateRefreshRetryInterval)
	source := model.CertSource{
		BaseModel:        model.BaseModel{ID: 4},
		Name:             "primary",
		PostSyncCommand:  "post command",
		LastSyncAt:       &lastSync,
		RefreshPendingAt: &pending,
	}
	var updated map[string]interface{}
	var refreshedIDs []uint
	commandCalls := 0
	err := retryPendingCertificateRefresh(source, now, certificateRefreshRetryOps{
		listCertificates: func(sourceID uint) ([]model.Certificate, error) {
			if sourceID != source.ID {
				t.Fatalf("source ID = %d", sourceID)
			}
			return []model.Certificate{{BaseModel: model.BaseModel{ID: 9}}, {BaseModel: model.BaseModel{ID: 10}}}, nil
		},
		updateSource: func(id uint, updates map[string]interface{}) error {
			if id != source.ID {
				t.Fatalf("updated source ID = %d", id)
			}
			updated = updates
			return nil
		},
		createLog: func(*model.CertSyncLog) error { return nil },
		refresh: func(ids []uint) error {
			refreshedIDs = append([]uint(nil), ids...)
			return nil
		},
		runCommand: func(command string) error {
			commandCalls++
			if command != source.PostSyncCommand {
				t.Fatalf("command = %q", command)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("retry pending refresh: %v", err)
	}
	if !reflect.DeepEqual(refreshedIDs, []uint{9, 10}) || commandCalls != 1 {
		t.Fatalf("refreshed IDs = %#v, command calls = %d", refreshedIDs, commandCalls)
	}
	if value, ok := updated["refresh_pending_at"]; !ok || value != nil {
		t.Fatalf("refresh_pending_at update = %#v", value)
	}
	if updated["last_sync_status"] != "success" || updated["last_sync_message"] != "服务刷新重试成功" {
		t.Fatalf("updates = %#v", updated)
	}
	if _, ok := updated["last_sync_at"]; ok {
		t.Fatalf("retry changed last_sync_at: %#v", updated)
	}
}

func TestRetryPendingCertificateRefreshFailure(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	pending := now.Add(-certificateRefreshRetryInterval)
	source := model.CertSource{BaseModel: model.BaseModel{ID: 4}, Name: "primary", RefreshPendingAt: &pending}
	refreshErr := errors.New("reload failed")
	var updated map[string]interface{}
	var logStatus string
	err := retryPendingCertificateRefresh(source, now, certificateRefreshRetryOps{
		listCertificates: func(uint) ([]model.Certificate, error) {
			return []model.Certificate{{BaseModel: model.BaseModel{ID: 9}}}, nil
		},
		updateSource: func(_ uint, updates map[string]interface{}) error {
			updated = updates
			return nil
		},
		createLog: func(entry *model.CertSyncLog) error {
			logStatus = entry.Status
			return nil
		},
		refresh:    func([]uint) error { return refreshErr },
		runCommand: func(string) error { return nil },
	})
	if !errors.Is(err, refreshErr) {
		t.Fatalf("retry error = %v", err)
	}
	gotPending, ok := updated["refresh_pending_at"].(*time.Time)
	if !ok || !gotPending.Equal(now) {
		t.Fatalf("refresh_pending_at = %#v", updated["refresh_pending_at"])
	}
	if updated["last_sync_status"] != "warning" || logStatus != "warning" {
		t.Fatalf("updates = %#v, log status = %q", updated, logStatus)
	}
	if _, ok := updated["last_sync_at"]; ok {
		t.Fatalf("retry changed last_sync_at: %#v", updated)
	}
}

func TestRetryPendingCertificateRefreshListFailureBacksOff(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	pending := now.Add(-certificateRefreshRetryInterval)
	source := model.CertSource{BaseModel: model.BaseModel{ID: 4}, Name: "primary", RefreshPendingAt: &pending}
	listErr := errors.New("list certificates failed")
	var updated map[string]interface{}
	var logStatus string
	err := retryPendingCertificateRefresh(source, now, certificateRefreshRetryOps{
		listCertificates: func(uint) ([]model.Certificate, error) { return nil, listErr },
		updateSource: func(_ uint, updates map[string]interface{}) error {
			updated = updates
			return nil
		},
		createLog: func(entry *model.CertSyncLog) error {
			logStatus = entry.Status
			return nil
		},
		refresh:    func([]uint) error { t.Fatal("refresh called after list failure"); return nil },
		runCommand: func(string) error { t.Fatal("command called after list failure"); return nil },
	})
	if !errors.Is(err, listErr) {
		t.Fatalf("retry error = %v", err)
	}
	gotPending, ok := updated["refresh_pending_at"].(*time.Time)
	if !ok || !gotPending.Equal(now) {
		t.Fatalf("refresh_pending_at = %#v", updated["refresh_pending_at"])
	}
	if updated["last_sync_status"] != "warning" || logStatus != "warning" {
		t.Fatalf("updates = %#v, log status = %q", updated, logStatus)
	}
}

func TestRetryPendingCertificateRefreshDoesNotLogSuccessWhenStateUpdateFails(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	pending := now.Add(-certificateRefreshRetryInterval)
	source := model.CertSource{BaseModel: model.BaseModel{ID: 4}, Name: "primary", RefreshPendingAt: &pending}
	updateErr := errors.New("update source failed")
	logCalls := 0
	err := retryPendingCertificateRefresh(source, now, certificateRefreshRetryOps{
		listCertificates: func(uint) ([]model.Certificate, error) {
			return []model.Certificate{{BaseModel: model.BaseModel{ID: 9}}}, nil
		},
		updateSource: func(uint, map[string]interface{}) error { return updateErr },
		createLog: func(*model.CertSyncLog) error {
			logCalls++
			return nil
		},
		refresh:    func([]uint) error { return nil },
		runCommand: func(string) error { return nil },
	})
	if !errors.Is(err, updateErr) {
		t.Fatalf("retry error = %v", err)
	}
	if logCalls != 0 {
		t.Fatalf("success log calls = %d, want 0 when state was not persisted", logCalls)
	}
}

func TestApplyCertificateRefreshPendingUpdate(t *testing.T) {
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)

	updates := map[string]interface{}{}
	applyCertificateRefreshPendingUpdate(updates, 0, nil, now)
	if _, ok := updates["refresh_pending_at"]; ok {
		t.Fatalf("empty sync changed pending state: %#v", updates)
	}

	updates = map[string]interface{}{}
	applyCertificateRefreshPendingUpdate(updates, 1, nil, now)
	if value, ok := updates["refresh_pending_at"]; !ok || value != nil {
		t.Fatalf("successful refresh pending update = %#v", updates)
	}

	refreshErr := errors.New("reload failed")
	updates = map[string]interface{}{}
	applyCertificateRefreshPendingUpdate(updates, 1, refreshErr, now)
	gotPending, ok := updates["refresh_pending_at"].(*time.Time)
	if !ok || !gotPending.Equal(now) {
		t.Fatalf("failed refresh pending update = %#v", updates)
	}
}
