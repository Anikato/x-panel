package database

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type RestoreSQLFile struct {
	Path    string
	cleanup func()
}

func (f *RestoreSQLFile) Cleanup() {
	if f != nil && f.cleanup != nil {
		f.cleanup()
	}
}

func PrepareSQLRestoreFile(inFile string) (*RestoreSQLFile, error) {
	lower := strings.ToLower(inFile)
	switch {
	case strings.HasSuffix(lower, ".sql"):
		return &RestoreSQLFile{Path: inFile}, nil
	case strings.HasSuffix(lower, ".sql.gz"):
		return extractGzipSQL(inFile)
	case strings.HasSuffix(lower, ".zip"):
		return extractZipSQL(inFile)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarSQL(inFile, true)
	case strings.HasSuffix(lower, ".tar"):
		return extractTarSQL(inFile, false)
	default:
		return nil, fmt.Errorf("unsupported sql backup file: %s", inFile)
	}
}

func IsSQLRestoreFile(inFile string) bool {
	lower := strings.ToLower(inFile)
	return strings.HasSuffix(lower, ".sql") ||
		strings.HasSuffix(lower, ".sql.gz") ||
		strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".tar")
}

func extractGzipSQL(inFile string) (*RestoreSQLFile, error) {
	src, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("open gzip file: %v", err)
	}
	defer src.Close()

	gz, err := gzip.NewReader(src)
	if err != nil {
		return nil, fmt.Errorf("open gzip reader: %v", err)
	}
	defer gz.Close()

	return writeTempSQL(gz)
}

func extractZipSQL(inFile string) (*RestoreSQLFile, error) {
	reader, err := zip.OpenReader(inFile)
	if err != nil {
		return nil, fmt.Errorf("open zip file: %v", err)
	}
	defer reader.Close()

	var selected *zip.File
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || !isSQLName(file.Name) {
			continue
		}
		if selected == nil || isPreferredSQLName(file.Name) {
			selected = file
		}
		if isPreferredSQLName(file.Name) {
			break
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("no sql file found in zip")
	}

	src, err := selected.Open()
	if err != nil {
		return nil, fmt.Errorf("open sql in zip: %v", err)
	}
	defer src.Close()

	return writeTempSQL(src)
}

func extractTarSQL(inFile string, gzipped bool) (*RestoreSQLFile, error) {
	src, err := os.Open(inFile)
	if err != nil {
		return nil, fmt.Errorf("open tar file: %v", err)
	}
	defer src.Close()

	var reader io.Reader = src
	var gz *gzip.Reader
	if gzipped {
		gz, err = gzip.NewReader(src)
		if err != nil {
			return nil, fmt.Errorf("open gzip reader: %v", err)
		}
		defer gz.Close()
		reader = gz
	}

	tr := tar.NewReader(reader)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar file: %v", err)
		}
		if header.FileInfo().IsDir() || !isSQLName(header.Name) {
			continue
		}
		if isPreferredSQLName(header.Name) {
			return writeTempSQL(tr)
		}

		temp, err := writeTempSQL(tr)
		if err != nil {
			return nil, err
		}
		return temp, nil
	}

	return nil, fmt.Errorf("no sql file found in tar")
}

func writeTempSQL(reader io.Reader) (*RestoreSQLFile, error) {
	temp, err := os.CreateTemp("", "xpanel-db-restore-*.sql")
	if err != nil {
		return nil, fmt.Errorf("create temp sql file: %v", err)
	}
	tempPath := temp.Name()
	cleanup := func() { _ = os.Remove(tempPath) }

	if _, err := io.Copy(temp, reader); err != nil {
		_ = temp.Close()
		cleanup()
		return nil, fmt.Errorf("write temp sql file: %v", err)
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return nil, fmt.Errorf("close temp sql file: %v", err)
	}

	return &RestoreSQLFile{Path: tempPath, cleanup: cleanup}, nil
}

func isSQLName(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".sql")
}

func isPreferredSQLName(name string) bool {
	return strings.EqualFold(filepath.Base(name), "test.sql")
}
