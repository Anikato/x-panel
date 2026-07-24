package credentials

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"xpanel/global"

	"gorm.io/gorm"
)

const migrationMarker = "_mig_credential_encryption_v1"

type DatabaseState struct {
	HasPlaintext bool
	HasEncrypted bool
	KeyIDs       map[string]struct{}
}

type credentialValueRow struct {
	ID    uint
	Value string
}

func ScanDatabase(db *gorm.DB) (DatabaseState, error) {
	return scanDatabase(db)
}

func MigrateDatabase(db *gorm.DB, protector global.CredentialProtector, backupDir string) error {
	return migrateDatabase(db, protector, backupDir, scrubSQLite)
}

func migrateDatabase(
	db *gorm.DB,
	protector global.CredentialProtector,
	backupDir string,
	scrub func(*gorm.DB) error,
) error {
	if db == nil {
		return errors.New("credential migration database is nil")
	}
	if protector == nil {
		return errors.New("credential protector is not initialized")
	}
	if scrub == nil {
		return errors.New("credential migration scrub function is nil")
	}

	state, scanErr := scanDatabase(db)
	if state.HasPlaintext {
		if _, err := createMigrationSnapshot(db, backupDir); err != nil {
			return err
		}
	}
	if scanErr != nil {
		return scanErr
	}

	marked, err := migrationMarked(db)
	if err != nil {
		return err
	}
	needsReencryption := stateUsesInactiveKey(state, protector.ActiveKeyID())
	if state.HasPlaintext {
		if err := db.Exec("PRAGMA secure_delete = ON").Error; err != nil {
			return fmt.Errorf("enable SQLite secure_delete: %w", err)
		}
	}
	if state.HasPlaintext || !marked || needsReencryption {
		if marked {
			if err := clearMigrationMarker(db); err != nil {
				return err
			}
		}
		if err := db.Transaction(func(tx *gorm.DB) error {
			if state.HasPlaintext || needsReencryption {
				if err := transformDatabase(tx, protector, needsReencryption); err != nil {
					return err
				}
			}
			if err := ValidateDatabase(tx, protector); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
		if err := scrub(db); err != nil {
			return err
		}
		if err := setMigrationMarker(db); err != nil {
			return err
		}
	} else if err := ValidateDatabase(db, protector); err != nil {
		return err
	}
	if err := removeMigrationSnapshots(backupDir); err != nil {
		return err
	}
	return nil
}

func stateUsesInactiveKey(state DatabaseState, activeKeyID string) bool {
	for keyID := range state.KeyIDs {
		if keyID != activeKeyID {
			return true
		}
	}
	return false
}

func ValidateDatabase(db *gorm.DB, protector global.CredentialProtector) error {
	if db == nil {
		return errors.New("credential validation database is nil")
	}
	if protector == nil {
		return errors.New("credential protector is not initialized")
	}
	return visitCredentialValues(db, func(scope string, row credentialValueRow) error {
		if !protector.IsEncrypted(row.Value) {
			return fmt.Errorf("%s record %d contains plaintext", scope, row.ID)
		}
		if err := protector.Validate(scope, row.Value); err != nil {
			return fmt.Errorf("validate %s record %d: %w", scope, row.ID, err)
		}
		return nil
	})
}

func ReencryptDatabase(db *gorm.DB, protector global.CredentialProtector) error {
	if db == nil {
		return errors.New("credential re-encryption database is nil")
	}
	if protector == nil {
		return errors.New("credential protector is not initialized")
	}
	if err := db.Exec("PRAGMA secure_delete = ON").Error; err != nil {
		return fmt.Errorf("enable SQLite secure_delete: %w", err)
	}
	if err := clearMigrationMarker(db); err != nil {
		return err
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := transformDatabase(tx, protector, true); err != nil {
			return err
		}
		if err := ValidateDatabase(tx, protector); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	if err := scrubSQLite(db); err != nil {
		return err
	}
	return setMigrationMarker(db)
}

func scanDatabase(db *gorm.DB) (DatabaseState, error) {
	state := DatabaseState{KeyIDs: make(map[string]struct{})}
	var firstErr error
	err := visitCredentialValues(db, func(scope string, row credentialValueRow) error {
		if IsEncrypted(row.Value) {
			state.HasEncrypted = true
			keyID, err := EnvelopeKeyID(row.Value)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("inspect %s record %d: %w", scope, row.ID, err)
				}
				return nil
			}
			state.KeyIDs[keyID] = struct{}{}
			return nil
		}
		state.HasPlaintext = true
		return nil
	})
	if err != nil {
		return state, err
	}
	return state, firstErr
}

func transformDatabase(db *gorm.DB, protector global.CredentialProtector, reencrypt bool) error {
	for _, spec := range FieldSpecs {
		rows, err := readFieldRows(db, spec)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if protector.IsEncrypted(row.Value) && !reencrypt {
				if err := protector.Validate(spec.Scope, row.Value); err != nil {
					return fmt.Errorf("validate %s record %d: %w", spec.Scope, row.ID, err)
				}
				continue
			}
			protected, err := protector.Protect(spec.Scope, row.Value)
			if err != nil {
				return fmt.Errorf("protect %s record %d: %w", spec.Scope, row.ID, err)
			}
			if protected == row.Value {
				continue
			}
			if err := db.Table(spec.Table).Where("id = ?", row.ID).
				Update(spec.Column, protected).Error; err != nil {
				return fmt.Errorf("update %s record %d: %w", spec.Scope, row.ID, err)
			}
		}
	}

	rows, err := readSecretSettingRows(db)
	if err != nil {
		return err
	}
	for _, row := range rows {
		var key string
		if err := db.Table("settings").Select("key").Where("id = ?", row.ID).Scan(&key).Error; err != nil {
			return fmt.Errorf("read secret setting key for record %d: %w", row.ID, err)
		}
		scope := SettingScope(key)
		if protector.IsEncrypted(row.Value) && !reencrypt {
			if err := protector.Validate(scope, row.Value); err != nil {
				return fmt.Errorf("validate %s record %d: %w", scope, row.ID, err)
			}
			continue
		}
		protected, err := protector.Protect(scope, row.Value)
		if err != nil {
			return fmt.Errorf("protect %s record %d: %w", scope, row.ID, err)
		}
		if protected == row.Value {
			continue
		}
		if err := db.Table("settings").Where("id = ?", row.ID).
			Update("value", protected).Error; err != nil {
			return fmt.Errorf("update %s record %d: %w", scope, row.ID, err)
		}
	}
	return nil
}

func visitCredentialValues(db *gorm.DB, visit func(scope string, row credentialValueRow) error) error {
	for _, spec := range FieldSpecs {
		rows, err := readFieldRows(db, spec)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if err := visit(spec.Scope, row); err != nil {
				return err
			}
		}
	}

	var settings []struct {
		ID    uint
		Key   string
		Value string
	}
	if err := db.Table("settings").
		Select("id, key, value").
		Where("key IN ? AND value IS NOT NULL AND value <> ''", SecretSettingKeyList()).
		Find(&settings).Error; err != nil {
		return fmt.Errorf("read registered secret settings: %w", err)
	}
	for _, setting := range settings {
		if err := visit(SettingScope(setting.Key), credentialValueRow{
			ID:    setting.ID,
			Value: setting.Value,
		}); err != nil {
			return err
		}
	}
	return nil
}

func readFieldRows(db *gorm.DB, spec FieldSpec) ([]credentialValueRow, error) {
	if err := validateIdentifier(spec.Table); err != nil {
		return nil, err
	}
	if err := validateIdentifier(spec.Column); err != nil {
		return nil, err
	}
	var rows []credentialValueRow
	selectSQL := fmt.Sprintf("id, %s AS value", quoteIdentifier(spec.Column))
	whereSQL := fmt.Sprintf("%s IS NOT NULL AND %s <> ''", quoteIdentifier(spec.Column), quoteIdentifier(spec.Column))
	if err := db.Table(spec.Table).Select(selectSQL).Where(whereSQL).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("read %s: %w", spec.Scope, err)
	}
	return rows, nil
}

func readSecretSettingRows(db *gorm.DB) ([]credentialValueRow, error) {
	var rows []credentialValueRow
	if err := db.Table("settings").
		Select("id, value").
		Where("key IN ? AND value IS NOT NULL AND value <> ''", SecretSettingKeyList()).
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("read registered secret settings: %w", err)
	}
	return rows, nil
}

func createMigrationSnapshot(db *gorm.DB, backupDir string) (string, error) {
	if backupDir == "" {
		return "", errors.New("credential migration backup directory is empty")
	}
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return "", fmt.Errorf("create credential migration backup directory %s: %w", backupDir, err)
	}
	if err := os.Chmod(backupDir, 0700); err != nil {
		return "", fmt.Errorf("harden credential migration backup directory %s: %w", backupDir, err)
	}

	randomID := make([]byte, 8)
	if _, err := rand.Read(randomID); err != nil {
		return "", fmt.Errorf("generate credential migration snapshot name: %w", err)
	}
	path := filepath.Join(backupDir, "pre-credential-migration-"+hex.EncodeToString(randomID)+".db")
	escapedPath := strings.ReplaceAll(path, "'", "''")
	if err := db.Exec("VACUUM INTO '" + escapedPath + "'").Error; err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("create credential migration snapshot %s: %w", path, err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		return "", fmt.Errorf("harden credential migration snapshot %s: %w", path, err)
	}
	return path, nil
}

func removeMigrationSnapshots(backupDir string) error {
	if backupDir == "" {
		return nil
	}
	entries, err := os.ReadDir(backupDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read credential migration backup directory %s: %w", backupDir, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "pre-credential-migration-") || !strings.HasSuffix(name, ".db") {
			continue
		}
		path := filepath.Join(backupDir, name)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove successful credential migration snapshot %s: %w", path, err)
		}
	}
	return nil
}

func scrubSQLite(db *gorm.DB) error {
	if err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error; err != nil {
		return fmt.Errorf("truncate SQLite WAL after credential migration: %w", err)
	}
	if err := db.Exec("VACUUM").Error; err != nil {
		return fmt.Errorf("vacuum SQLite after credential migration: %w", err)
	}
	return nil
}

func migrationMarked(db *gorm.DB) (bool, error) {
	var count int64
	if err := db.Table("settings").Where("key = ?", migrationMarker).Count(&count).Error; err != nil {
		return false, fmt.Errorf("read credential migration marker: %w", err)
	}
	return count > 0, nil
}

func setMigrationMarker(db *gorm.DB) error {
	var count int64
	if err := db.Table("settings").Where("key = ?", migrationMarker).Count(&count).Error; err != nil {
		return fmt.Errorf("read credential migration marker: %w", err)
	}
	if count == 0 {
		if err := db.Table("settings").Create(map[string]any{
			"key":   migrationMarker,
			"value": "done",
		}).Error; err != nil {
			return fmt.Errorf("create credential migration marker: %w", err)
		}
		return nil
	}
	if err := db.Table("settings").Where("key = ?", migrationMarker).Update("value", "done").Error; err != nil {
		return fmt.Errorf("update credential migration marker: %w", err)
	}
	return nil
}

func clearMigrationMarker(db *gorm.DB) error {
	if err := db.Exec("DELETE FROM settings WHERE key = ?", migrationMarker).Error; err != nil {
		return fmt.Errorf("clear credential migration marker: %w", err)
	}
	return nil
}

func validateIdentifier(value string) error {
	if value == "" {
		return errors.New("empty SQL identifier")
	}
	for _, char := range value {
		if (char < 'a' || char > 'z') && char != '_' {
			return fmt.Errorf("unsafe SQL identifier %q", value)
		}
	}
	return nil
}

func quoteIdentifier(value string) string {
	return `"` + value + `"`
}
