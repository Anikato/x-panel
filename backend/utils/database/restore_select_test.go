package database

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeZipSQL(t *testing.T, path string, files map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	zw := zip.NewWriter(file)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeTarSQL(t *testing.T, path string, files map[string]string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	tw := tar.NewWriter(file)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tw, content); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareSQLRestoreFileSelectsOnlySQL(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "one.zip")
	writeZipSQL(t, zipPath, map[string]string{"dump.sql": "SELECT 1;\n"})
	got, err := PrepareSQLRestoreFile(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer got.Cleanup()
	data, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("SELECT 1;")) {
		t.Fatalf("selected content = %q", data)
	}
}

func TestPrepareSQLRestoreFileRejectsMultipleSQLInZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "many.zip")
	writeZipSQL(t, zipPath, map[string]string{
		"dump.sql": "SELECT 1;\n",
		"test.sql": "SELECT 2;\n",
	})
	_, err := PrepareSQLRestoreFile(zipPath)
	if err == nil {
		t.Fatal("expected multiple sql error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "multiple") {
		t.Fatalf("error = %v", err)
	}
}

func TestPrepareSQLRestoreFileRejectsMultipleSQLInTar(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "many.tar")
	writeTarSQL(t, tarPath, map[string]string{
		"a.sql": "SELECT 1;\n",
		"b.sql": "SELECT 2;\n",
	})
	_, err := PrepareSQLRestoreFile(tarPath)
	if err == nil {
		t.Fatal("expected multiple sql error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "multiple") {
		t.Fatalf("error = %v", err)
	}
}
