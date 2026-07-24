package service

import (
	"encoding/json"
	"strings"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
)

func TestCronjobInfoDoesNotExposeEncryptionPassword(t *testing.T) {
	info := toCronjobInfo(&model.Cronjob{EncryptPassword: "archive-secret"})
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal cronjob info: %v", err)
	}
	if strings.Contains(string(data), "archive-secret") ||
		strings.Contains(string(data), `"encryptPassword"`) {
		t.Fatalf("cronjob response exposed encryption password: %s", data)
	}
	if !info.EncryptPasswordSet {
		t.Fatalf("cronjob response did not report configured encryption password")
	}
}

func TestCronjobEmptyPasswordUpdateRetainsExistingSecret(t *testing.T) {
	existing := &model.Cronjob{EncryptPassword: "archive-secret"}
	fields, updated := buildCronjobUpdate(existing, dto.CronjobUpdate{
		Name:            "backup",
		Type:            "directory",
		Spec:            "0 2 * * *",
		SourceDir:       "/srv/data",
		EncryptPassword: "",
	})

	if _, exists := fields["encrypt_password"]; exists {
		t.Fatalf("empty password update unexpectedly clears encryption password")
	}
	if updated.EncryptPassword != "archive-secret" {
		t.Fatalf("updated password = %q, want existing secret", updated.EncryptPassword)
	}
}
