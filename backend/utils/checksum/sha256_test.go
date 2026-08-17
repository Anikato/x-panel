package checksum

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSHA256MatchesKnownEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := FileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	const want = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Fatalf("FileSHA256() = %q, want %q", got, want)
	}
}

func TestFileSHA256ReadsFileFromDiskNotMemorySnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	first, err := FileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}
	second, err := FileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("hash did not change after file content changed: %s", first)
	}
}
