package main

import (
	"path/filepath"
	"testing"

	"xpanel/app/model"
	"xpanel/global"
	"xpanel/security/credentials"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestParseFleetEnrollmentToken(t *testing.T) {
	const validFlagToken = "fenr_flag-token-with-at-least-32-characters"
	const validEnvToken = "fenr_environment.token.with.sufficient.length"

	tests := []struct {
		name      string
		args      []string
		envToken  string
		wantToken string
		wantErr   bool
	}{
		{
			name:      "explicit flag",
			args:      []string{"--token", validFlagToken},
			envToken:  validEnvToken,
			wantToken: validFlagToken,
		},
		{
			name:      "environment fallback",
			envToken:  validEnvToken,
			wantToken: validEnvToken,
		},
		{
			name:    "missing token",
			wantErr: true,
		},
		{
			name:    "short token",
			args:    []string{"--token", "too-short"},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := parseFleetEnrollmentToken(test.args, func(key string) string {
				if key == fleetEnrollmentTokenEnv {
					return test.envToken
				}
				return ""
			})
			if test.wantErr {
				if err == nil {
					t.Fatalf("parseFleetEnrollmentToken() token = %q, want error", token)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFleetEnrollmentToken() error = %v", err)
			}
			if token != test.wantToken {
				t.Fatalf("token = %q, want %q", token, test.wantToken)
			}
		})
	}
}

func TestSaveFleetEnrollmentTokenCreatesOrUpdatesSetting(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
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
	global.CREDENTIALS = manager
	t.Cleanup(func() {
		global.CREDENTIALS = nil
	})

	for _, token := range []string{
		"fenr_first-token-with-at-least-32-characters",
		"fenr_replacement.token.with.sufficient.length",
	} {
		if err := saveFleetEnrollmentToken(db, token); err != nil {
			t.Fatalf("save token: %v", err)
		}
	}

	var setting model.Setting
	if err := db.Where("`key` = ?", "FleetEnrollmentToken").First(&setting).Error; err != nil {
		t.Fatalf("load setting: %v", err)
	}
	if !manager.IsEncrypted(setting.Value) {
		t.Fatalf("stored token is plaintext: %q", setting.Value)
	}
	revealed, err := manager.Reveal("settings.FleetEnrollmentToken", setting.Value)
	if err != nil {
		t.Fatalf("reveal stored token: %v", err)
	}
	if want := "fenr_replacement.token.with.sufficient.length"; revealed != want {
		t.Fatalf("stored token = %q, want %q", revealed, want)
	}

	var count int64
	if err := db.Model(&model.Setting{}).
		Where("`key` = ?", "FleetEnrollmentToken").
		Count(&count).Error; err != nil {
		t.Fatalf("count settings: %v", err)
	}
	if count != 1 {
		t.Fatalf("setting count = %d, want 1", count)
	}
}
