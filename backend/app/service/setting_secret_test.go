package service

import (
	"path/filepath"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
	"xpanel/security/credentials"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSettingInfoDoesNotReturnTokens(t *testing.T) {
	openSettingServiceDatabase(t)
	settings := repo.NewISettingRepo()
	if err := settings.CreateOrUpdate("GitHubToken", "github-secret"); err != nil {
		t.Fatalf("store GitHub token: %v", err)
	}
	if err := settings.CreateOrUpdate("AgentToken", "agent-secret"); err != nil {
		t.Fatalf("store Agent token: %v", err)
	}
	if err := settings.CreateOrUpdate("SecurityEntrance", "private-entry"); err != nil {
		t.Fatalf("store security entrance: %v", err)
	}
	if err := settings.CreateOrUpdate("ProxyAddress", "socks5://user:proxy-secret@127.0.0.1:1080"); err != nil {
		t.Fatalf("store proxy address: %v", err)
	}

	info, err := NewISettingService().GetSettingInfo()
	if err != nil {
		t.Fatalf("get setting info: %v", err)
	}
	if info.GitHubToken != "" || info.AgentToken != "" ||
		info.SecurityEntrance != "" || info.ProxyAddress != "" {
		t.Fatalf("setting response exposed secrets: %#v", info)
	}
	if !info.GitHubTokenSet || !info.AgentTokenSet ||
		!info.SecurityEntranceSet || !info.ProxyAddressSet {
		t.Fatalf("setting response did not report configured secrets: %#v", info)
	}
}

func TestExplicitSecretClearRemovesStoredValue(t *testing.T) {
	openSettingServiceDatabase(t)
	settings := repo.NewISettingRepo()
	if err := settings.CreateOrUpdate("AgentToken", "agent-secret"); err != nil {
		t.Fatalf("store Agent token: %v", err)
	}
	if err := NewISettingService().Update(dto.SettingUpdate{
		Key:   "AgentToken",
		Clear: true,
	}); err != nil {
		t.Fatalf("clear Agent token: %v", err)
	}
	value, err := settings.GetValueByKey("AgentToken")
	if err != nil {
		t.Fatalf("read cleared Agent token: %v", err)
	}
	if value != "" {
		t.Fatalf("cleared Agent token = %q", value)
	}
}

func TestCertServerSettingDoesNotReturnToken(t *testing.T) {
	openSettingServiceDatabase(t)
	settings := repo.NewISettingRepo()
	if err := settings.CreateOrUpdate("CertServerToken", "certificate-server-secret"); err != nil {
		t.Fatalf("store certificate server token: %v", err)
	}
	info, err := NewICertServerService().GetSetting()
	if err != nil {
		t.Fatalf("get certificate server setting: %v", err)
	}
	if info.Token != "" || !info.TokenSet {
		t.Fatalf("certificate server setting exposed token or lost state: %#v", info)
	}
}

func TestEmptyTokenUpdateLeavesExistingSecretUnchanged(t *testing.T) {
	openSettingServiceDatabase(t)
	settings := repo.NewISettingRepo()
	if err := settings.CreateOrUpdate("GitHubToken", "github-secret"); err != nil {
		t.Fatalf("store GitHub token: %v", err)
	}

	if err := NewISettingService().Update(dto.SettingUpdate{
		Key:   "GitHubToken",
		Value: "",
	}); err != nil {
		t.Fatalf("empty token update: %v", err)
	}

	value, err := settings.GetValueByKey("GitHubToken")
	if err != nil {
		t.Fatalf("read GitHub token: %v", err)
	}
	if value != "github-secret" {
		t.Fatalf("GitHub token = %q, want unchanged", value)
	}
}

func openSettingServiceDatabase(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "settings.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open settings database: %v", err)
	}
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatalf("migrate settings: %v", err)
	}
	manager, _, err := credentials.LoadOrCreate(
		filepath.Join(t.TempDir(), "secrets", "credential-keyring.json"),
		true,
	)
	if err != nil {
		t.Fatalf("create credential manager: %v", err)
	}
	previousDB := global.DB
	previousCredentials := global.CREDENTIALS
	global.DB = db
	global.CREDENTIALS = manager
	t.Cleanup(func() {
		global.DB = previousDB
		global.CREDENTIALS = previousCredentials
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
}
