package credentials

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadOrCreateKeyringUsesRestrictiveModes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets", "credential-keyring.json")

	manager, created, err := LoadOrCreate(path, true)
	if err != nil {
		t.Fatalf("LoadOrCreate: %v", err)
	}
	if !created {
		t.Fatal("first LoadOrCreate must report creation")
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat keyring directory: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("keyring directory mode = %04o, want 0700", got)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat keyring: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0600 {
		t.Fatalf("keyring mode = %04o, want 0600", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read keyring: %v", err)
	}
	var doc keyringDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode keyring: %v", err)
	}
	if doc.Version != 1 {
		t.Fatalf("keyring version = %d, want 1", doc.Version)
	}
	if doc.ActiveKeyID == "" || doc.ActiveKeyID != manager.ActiveKeyID() {
		t.Fatalf("active key id = %q, manager = %q", doc.ActiveKeyID, manager.ActiveKeyID())
	}
	encodedKey, ok := doc.Keys[doc.ActiveKeyID]
	if !ok {
		t.Fatalf("active key %q missing from keyring", doc.ActiveKeyID)
	}
	key, err := base64.RawStdEncoding.DecodeString(encodedKey)
	if err != nil {
		t.Fatalf("decode active key: %v", err)
	}
	if len(key) != keySize {
		t.Fatalf("decoded key length = %d, want %d", len(key), keySize)
	}
}

func TestLoadOrCreateKeyringDoesNotReplaceExistingKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets", "credential-keyring.json")
	first, created, err := LoadOrCreate(path, true)
	if err != nil {
		t.Fatalf("create keyring: %v", err)
	}
	if !created {
		t.Fatal("first load must create keyring")
	}
	value, err := first.Protect("nodes.token", "finst-secret")
	if err != nil {
		t.Fatalf("protect value: %v", err)
	}
	firstKeyID := first.ActiveKeyID()

	second, created, err := LoadOrCreate(path, false)
	if err != nil {
		t.Fatalf("reload keyring: %v", err)
	}
	if created {
		t.Fatal("reload must not report keyring creation")
	}
	if second.ActiveKeyID() != firstKeyID {
		t.Fatalf("active key changed from %q to %q", firstKeyID, second.ActiveKeyID())
	}
	plaintext, err := second.Reveal("nodes.token", value)
	if err != nil {
		t.Fatalf("reveal with reloaded keyring: %v", err)
	}
	if plaintext != "finst-secret" {
		t.Fatalf("plaintext = %q, want finst-secret", plaintext)
	}
}

func TestLoadKeyringRejectsUnsafeOrMalformedFiles(t *testing.T) {
	t.Run("missing without create permission", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "secrets", "credential-keyring.json")
		if _, _, err := LoadOrCreate(path, false); err == nil {
			t.Fatal("missing keyring must fail when creation is disabled")
		}
	})

	t.Run("symlink", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "real.json")
		if err := os.WriteFile(target, []byte(`{"version":1}`), 0600); err != nil {
			t.Fatalf("write symlink target: %v", err)
		}
		path := filepath.Join(dir, "credential-keyring.json")
		if err := os.Symlink(target, path); err != nil {
			t.Fatalf("create symlink: %v", err)
		}
		if _, _, err := LoadOrCreate(path, false); err == nil {
			t.Fatal("symlink keyring must be rejected")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "credential-keyring.json")
		if err := os.WriteFile(path, []byte(`{"version":1,"activeKeyId":"missing","keys":{}}`), 0600); err != nil {
			t.Fatalf("write malformed keyring: %v", err)
		}
		if _, _, err := LoadOrCreate(path, false); err == nil {
			t.Fatal("malformed keyring must be rejected")
		}
	})
}

func TestManagerProtectRevealAndRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets", "credential-keyring.json")
	manager, _, err := LoadOrCreate(path, true)
	if err != nil {
		t.Fatalf("create keyring: %v", err)
	}

	oldKeyID := manager.ActiveKeyID()
	oldEnvelope, err := manager.Protect("hosts.password", "secret")
	if err != nil {
		t.Fatalf("protect with old key: %v", err)
	}
	newKeyID, err := manager.AddActiveKey()
	if err != nil {
		t.Fatalf("rotate keyring: %v", err)
	}
	if newKeyID == oldKeyID {
		t.Fatal("rotation must create a different active key")
	}
	if !slices.Contains(manager.KeyIDs(), oldKeyID) {
		t.Fatalf("old key %q was removed during rotation", oldKeyID)
	}

	plaintext, err := manager.Reveal("hosts.password", oldEnvelope)
	if err != nil {
		t.Fatalf("reveal old envelope after rotation: %v", err)
	}
	if plaintext != "secret" {
		t.Fatalf("old envelope plaintext = %q, want secret", plaintext)
	}

	rewrapped, err := manager.Protect("hosts.password", oldEnvelope)
	if err != nil {
		t.Fatalf("protect old envelope with active key: %v", err)
	}
	rewrappedKeyID, err := EnvelopeKeyID(rewrapped)
	if err != nil {
		t.Fatalf("read rewrapped key id: %v", err)
	}
	if rewrappedKeyID != newKeyID {
		t.Fatalf("rewrapped key id = %q, want %q", rewrappedKeyID, newKeyID)
	}
	if err := manager.Validate("hosts.password", rewrapped); err != nil {
		t.Fatalf("validate rewrapped envelope: %v", err)
	}
}
