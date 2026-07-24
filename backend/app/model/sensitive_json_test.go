package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSensitiveModelFieldsAreNotSerialized(t *testing.T) {
	tests := []struct {
		name      string
		value     any
		forbidden []string
	}{
		{
			name:      "node credentials",
			value:     Node{Token: "node-token", SSHPassword: "ssh-password"},
			forbidden: []string{"node-token", "ssh-password", `"token"`, `"sshPassword"`},
		},
		{
			name:      "backup credentials",
			value:     BackupAccount{AccessKey: "access-key", Credential: "credential"},
			forbidden: []string{"access-key", "credential", `"accessKey"`, `"credential"`},
		},
		{
			name:      "database server password",
			value:     DatabaseServer{Password: "database-password"},
			forbidden: []string{"database-password", `"password"`},
		},
		{
			name:      "ACME EAB secret",
			value:     AcmeAccount{EabHmacKey: "eab-secret"},
			forbidden: []string{"eab-secret", `"eabHmacKey"`},
		},
		{
			name:      "cron archive password",
			value:     Cronjob{EncryptPassword: "archive-password"},
			forbidden: []string{"archive-password", `"encryptPassword"`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.value)
			if err != nil {
				t.Fatalf("marshal model: %v", err)
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(string(data), forbidden) {
					t.Fatalf("serialized sensitive value %q in %s", forbidden, data)
				}
			}
		})
	}
}
