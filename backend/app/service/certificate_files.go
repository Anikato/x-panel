package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"xpanel/app/model"
)

type syncedCertFileTransaction struct {
	certPath     string
	keyPath      string
	certTmp      string
	keyTmp       string
	certBak      string
	keyBak       string
	certHad      bool
	keyHad       bool
	certReplaced bool
	keyReplaced  bool
	committed    bool
}

func prepareSyncedCertFileTransaction(sslDir string, cert model.Certificate, preserveExisting bool) (*syncedCertFileTransaction, error) {
	certPath, keyPath := certFilePaths(sslDir, cert)
	if preserveExisting {
		legacyCert, legacyKey := existingCertFilePaths(sslDir, cert)
		if _, err := os.Stat(legacyCert); err == nil {
			certPath, keyPath = legacyCert, legacyKey
		}
	}
	dir := filepath.Dir(certPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create cert dir: %w", err)
	}

	certTmp, err := writeCertTempFile(dir, ".fullchain-*.tmp", cert.Pem, 0644)
	if err != nil {
		return nil, err
	}
	keyTmp, err := writeCertTempFile(dir, ".privkey-*.tmp", cert.PrivateKey, 0600)
	if err != nil {
		_ = os.Remove(certTmp)
		return nil, err
	}
	if _, err := tls.LoadX509KeyPair(certTmp, keyTmp); err != nil {
		_ = os.Remove(certTmp)
		_ = os.Remove(keyTmp)
		return nil, fmt.Errorf("validate certificate key pair: %w", err)
	}
	return &syncedCertFileTransaction{
		certPath: certPath,
		keyPath:  keyPath,
		certTmp:  certTmp,
		keyTmp:   keyTmp,
	}, nil
}

func (tx *syncedCertFileTransaction) Commit() error {
	if tx.committed {
		return nil
	}
	var err error
	if tx.certBak, tx.certHad, err = backupCertFile(tx.certPath, ".fullchain-backup-*"); err != nil {
		return fmt.Errorf("backup certificate: %w", err)
	}
	if tx.keyBak, tx.keyHad, err = backupCertFile(tx.keyPath, ".privkey-backup-*"); err != nil {
		_ = tx.cleanup()
		return fmt.Errorf("backup private key: %w", err)
	}
	if err := os.Rename(tx.certTmp, tx.certPath); err != nil {
		_ = tx.cleanup()
		return fmt.Errorf("replace certificate: %w", err)
	}
	tx.certTmp = ""
	tx.certReplaced = true
	if err := os.Rename(tx.keyTmp, tx.keyPath); err != nil {
		restoreErr := tx.Rollback()
		if restoreErr != nil {
			return errors.Join(fmt.Errorf("replace private key: %w", err), restoreErr)
		}
		return fmt.Errorf("replace private key: %w", err)
	}
	tx.keyTmp = ""
	tx.keyReplaced = true
	ensureCertPermissions(filepath.Dir(tx.certPath), tx.certPath, tx.keyPath)
	tx.committed = true
	return nil
}

func (tx *syncedCertFileTransaction) Rollback() error {
	err := tx.restoreOriginalFiles()
	cleanupErr := tx.cleanup()
	tx.committed = false
	return errors.Join(err, cleanupErr)
}

func (tx *syncedCertFileTransaction) Finalize() error {
	tx.committed = false
	return tx.cleanup()
}

func (tx *syncedCertFileTransaction) restoreOriginalFiles() error {
	var certErr, keyErr error
	if tx.certReplaced {
		certErr = restoreCertFile(tx.certPath, tx.certBak, tx.certHad)
		if certErr == nil {
			tx.certReplaced = false
		}
	}
	if tx.keyReplaced {
		keyErr = restoreCertFile(tx.keyPath, tx.keyBak, tx.keyHad)
		if keyErr == nil {
			tx.keyReplaced = false
		}
	}
	return errors.Join(certErr, keyErr)
}

func (tx *syncedCertFileTransaction) cleanup() error {
	var errs []error
	for _, path := range []string{tx.certTmp, tx.keyTmp, tx.certBak, tx.keyBak} {
		if path != "" {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				errs = append(errs, err)
			}
		}
	}
	tx.certTmp, tx.keyTmp, tx.certBak, tx.keyBak = "", "", "", ""
	return errors.Join(errs...)
}

func writeCertTempFile(dir, pattern, content string, perm os.FileMode) (string, error) {
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	path := file.Name()
	if _, err = io.WriteString(file, content); err == nil {
		err = file.Chmod(perm)
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func backupCertFile(path, pattern string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	backup, err := writeCertTempFile(filepath.Dir(path), pattern, string(data), 0600)
	if err != nil {
		return "", false, err
	}
	return backup, true, nil
}

func restoreCertFile(path, backup string, existed bool) error {
	if !existed {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if backup == "" {
		return fmt.Errorf("missing backup for %s", path)
	}
	return os.Rename(backup, path)
}

func saveSyncedCertFilesAtomic(sslDir string, cert model.Certificate, preserveExisting bool) error {
	tx, err := prepareSyncedCertFileTransaction(sslDir, cert, preserveExisting)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Finalize()
}
