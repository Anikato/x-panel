package credential

import (
	"errors"
	"fmt"
	"path/filepath"

	"xpanel/global"
	initPermission "xpanel/init/permission"
	securityCredentials "xpanel/security/credentials"

	"gorm.io/gorm"
)

func Init() error {
	if global.DB == nil {
		return errors.New("database is not initialized")
	}
	if global.CONF.System.CredentialKeyPath == "" {
		return errors.New("credential keyring path is empty")
	}
	backupDir := filepath.Join(
		installRoot(global.CONF.System.DataDir),
		"backups",
		"credential-migration",
	)
	_, err := Initialize(
		global.DB,
		global.CONF.System.CredentialKeyPath,
		backupDir,
	)
	return err
}

func Initialize(db *gorm.DB, keyPath, backupDir string) (*securityCredentials.Manager, error) {
	if db == nil {
		return nil, errors.New("database is not initialized")
	}

	state, scanErr := securityCredentials.ScanDatabase(db)
	manager, _, err := securityCredentials.LoadOrCreate(keyPath, !state.HasEncrypted)
	if err != nil {
		return nil, err
	}
	if scanErr != nil {
		return nil, scanErr
	}
	if err := securityCredentials.MigrateDatabase(db, manager, backupDir); err != nil {
		return nil, fmt.Errorf("migrate stored credentials: %w", err)
	}
	if err := securityCredentials.ValidateDatabase(db, manager); err != nil {
		return nil, fmt.Errorf("validate stored credentials: %w", err)
	}

	global.CREDENTIALS = manager
	return manager, nil
}

func installRoot(dataDir string) string {
	return initPermission.ResolveInstallRoot(dataDir)
}
