package service

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
	"xpanel/security/credentials"
)

type failingManualCertificateRepo struct {
	repo.ICertificateRepo
	err error
}

func (r failingManualCertificateRepo) Update(id uint, updates map[string]interface{}) error {
	if _, committing := updates["pem"]; committing {
		return r.err
	}
	return r.ICertificateRepo.Update(id, updates)
}

func TestUpdateManualCertificatePreservesBindingsAndRollsBackFiles(t *testing.T) {
	installCertificateConsumerDatabase(t)
	installFakeNginx(t, false)
	manager, _, err := credentials.LoadOrCreate(filepath.Join(t.TempDir(), "keyring.json"), true)
	if err != nil {
		t.Fatal(err)
	}
	previousCredentials := global.CREDENTIALS
	global.CREDENTIALS = manager
	t.Cleanup(func() { global.CREDENTIALS = previousCredentials })

	service := NewICertificateService().(*CertificateService)
	sslDir := t.TempDir()
	if err := service.settingRepo.CreateOrUpdate("SSLDir", sslDir); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	oldPEM, oldKey := generateUploadPEMsForPeriod(t, now.Add(-48*time.Hour), now.Add(-time.Hour))
	if err := service.Upload(dto.CertificateUpload{Certificate: oldPEM, PrivateKey: oldKey, Description: "keep"}); err != nil {
		t.Fatal(err)
	}
	certs, err := service.certRepo.GetList()
	if err != nil || len(certs) != 1 {
		t.Fatalf("uploaded certificates: count=%d error=%v", len(certs), err)
	}
	cert := certs[0]
	// Older installations bind services to the domain-based certificate directory.
	certPath, _ := certFilePaths(sslDir, cert)
	legacyDir := filepath.Join(sslDir, "certs", cert.PrimaryDomain)
	if err := os.Rename(filepath.Dir(certPath), legacyDir); err != nil {
		t.Fatal(err)
	}
	panelID := strconv.FormatUint(uint64(cert.ID), 10)
	if err := service.settingRepo.CreateOrUpdate("PanelSSLCertificateID", panelID); err != nil {
		t.Fatal(err)
	}
	site := &model.Website{PrimaryDomain: cert.PrimaryDomain, Alias: "manual", Status: "stopped", SSLEnable: true, CertificateID: cert.ID}
	if err := global.DB.Create(site).Error; err != nil {
		t.Fatal(err)
	}
	assertFiles := func(wantPEM, wantKey string) {
		t.Helper()
		for name, want := range map[string]string{"fullchain.pem": wantPEM, "privkey.pem": wantKey} {
			got, err := os.ReadFile(filepath.Join(legacyDir, name))
			if err != nil || string(got) != want {
				t.Fatalf("unexpected contents of %s: %v", name, err)
			}
		}
	}

	newPEM, newKey := generateUploadPEMs(t)
	req := dto.ManualCertificateUpdate{ID: cert.ID, Certificate: newPEM, PrivateKey: newKey}
	persistErr := errors.New("database update failed")
	originalRepo := service.certRepo
	service.certRepo = failingManualCertificateRepo{ICertificateRepo: originalRepo, err: persistErr}
	if _, err := service.UpdateManual(req); !errors.Is(err, persistErr) {
		t.Fatalf("expected database error, got %v", err)
	}
	assertFiles(oldPEM, oldKey)
	service.certRepo = originalRepo

	result, err := service.UpdateManual(req)
	if err != nil || result.ActivationError != "" {
		t.Fatalf("update failed: %#v %v", result, err)
	}
	assertFiles(newPEM, newKey)
	updated, err := service.certRepo.Get(repo.WithByID(cert.ID))
	if err != nil {
		t.Fatal(err)
	}
	if updated.Fingerprint == cert.Fingerprint || updated.Pem != newPEM || updated.PrivateKey != newKey ||
		updated.LineageUID != cert.LineageUID || updated.Description != cert.Description {
		t.Fatal("expected updated certificate material and metadata with the same identity")
	}
	if updated.BackupSavedAt == nil || updated.BackupPEM != oldPEM || updated.BackupPrivateKey != oldKey {
		t.Fatal("missing previous certificate and key")
	}
	savedAt := *updated.BackupSavedAt
	var raw model.Certificate
	if err := global.DB.Select("backup_private_key").First(&raw, cert.ID).Error; err != nil || !credentials.IsEncrypted(raw.BackupPrivateKey) {
		t.Fatal("stored backup key was not encrypted")
	}
	if _, err := service.UpdateManual(req); err != nil {
		t.Fatal(err)
	}
	detail, err := service.GetDetail(cert.ID)
	if err != nil || detail.Backup == nil || !detail.Backup.SavedAt.Equal(savedAt) {
		t.Fatalf("retry changed the original backup: %v", err)
	}

	// A process may stop after replacing one file and before committing the row.
	if err := service.certRepo.Update(cert.ID, map[string]interface{}{"pending_pem": newPEM, "pending_private_key": newKey}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "fullchain.pem"), []byte(oldPEM), 0644); err != nil {
		t.Fatal(err)
	}
	RecoverInterruptedManualCertificateUpdates()
	assertFiles(newPEM, newKey)
	recovered, err := service.certRepo.Get(repo.WithByID(cert.ID))
	if err != nil || recovered.PendingPrivateKey != "" || recovered.BackupPEM != oldPEM {
		t.Fatalf("interrupted update recovery did not preserve the last backup: %v", err)
	}

	restoreReq := dto.ManualCertificateRestore{ID: cert.ID, SavedAt: savedAt}
	service.certRepo = failingManualCertificateRepo{ICertificateRepo: originalRepo, err: persistErr}
	if _, err := service.RestoreManual(restoreReq); !errors.Is(err, persistErr) {
		t.Fatalf("expected database error on restore, got %v", err)
	}
	assertFiles(newPEM, newKey)
	service.certRepo = originalRepo

	// An explicit restore can bring back an expired original during incident recovery.
	result, err = service.RestoreManual(restoreReq)
	if err != nil || result.ActivationError != "" {
		t.Fatalf("restore failed: %#v %v", result, err)
	}
	assertFiles(oldPEM, oldKey)
	restored, err := service.certRepo.Get(repo.WithByID(cert.ID))
	if err != nil || restored.Pem != oldPEM || restored.PrivateKey != oldKey ||
		restored.Fingerprint != cert.Fingerprint || restored.BackupPEM != newPEM || restored.BackupPrivateKey != newKey {
		t.Fatalf("expected previous certificate to become active and the replaced pair to become the backup: %v", err)
	}
	if _, err := service.RestoreManual(restoreReq); err == nil {
		t.Fatal("stale restore request unexpectedly replaced the current certificate")
	}
	if linkedPanelID, err := service.settingRepo.GetValueByKey("PanelSSLCertificateID"); err != nil || linkedPanelID != panelID {
		t.Fatalf("panel lost its binding: %v", err)
	}
	var linked model.Website
	if err := global.DB.First(&linked, site.ID).Error; err != nil || linked.CertificateID != cert.ID {
		t.Fatalf("website lost its binding: %v", err)
	}
}

func TestManualCertificateWritesRejectSyncedCertificates(t *testing.T) {
	svc := setupCertificateServiceDB(t)
	for _, remote := range []model.Certificate{
		{PrimaryDomain: "synced.example.com", Provider: "manual", Type: "synced", SourceType: "synced"},
		{PrimaryDomain: "legacy.example.com", Provider: "manual", Type: "upload", SourceType: "synced"},
	} {
		if err := svc.certRepo.Create(&remote); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.UpdateManual(dto.ManualCertificateUpdate{ID: remote.ID}); err == nil {
			t.Fatalf("updated upstream-provided certificate #%d", remote.ID)
		}
		if _, err := svc.RestoreManual(dto.ManualCertificateRestore{ID: remote.ID, SavedAt: time.Now()}); err == nil {
			t.Fatalf("restored upstream-provided certificate #%d", remote.ID)
		}
		current, err := svc.certRepo.Get(repo.WithByID(remote.ID))
		if err != nil || current.SourceType != "synced" || current.Type != remote.Type {
			t.Fatalf("source identity changed, error=%v", err)
		}
	}
}
