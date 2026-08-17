package cloud_storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xpanel/utils/checksum"
)

func TestStagingObjectPathAppendsPartSuffix(t *testing.T) {
	got := stagingObjectPath("database/app/db.sql")
	if got != "database/app/db.sql.part" {
		t.Fatalf("stagingObjectPath() = %q", got)
	}
}

func TestParseRemoteSHA256FromSha256sum(t *testing.T) {
	out := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  /data/db.sql.part\n"
	got, err := parseRemoteSHA256(out)
	if err != nil {
		t.Fatal(err)
	}
	if got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("parseRemoteSHA256() = %q", got)
	}
}

func TestParseRemoteSHA256FromOpenSSL(t *testing.T) {
	out := "SHA256(/data/db.sql.part)= E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855\n"
	got, err := parseRemoteSHA256(out)
	if err != nil {
		t.Fatal(err)
	}
	if got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Fatalf("parseRemoteSHA256() = %q", got)
	}
}

func TestUploadWithIntegrityRenamesAfterHashMatch(t *testing.T) {
	src := writeTempBackup(t, "payload")
	localHash := mustHash(t, src)
	remote := newMemoryRemote()
	remote.hashResults = []string{localHash}

	if err := uploadWithIntegrity(src, "db.sql", localHash, remote, nil); err != nil {
		t.Fatal(err)
	}
	if remote.files["db.sql"] != "payload" {
		t.Fatalf("final object = %q, want payload", remote.files["db.sql"])
	}
	if _, ok := remote.files["db.sql.part"]; ok {
		t.Fatal("staging object should be renamed away")
	}
}

func TestUploadWithIntegrityRetriesWhenRemoteHashMismatches(t *testing.T) {
	src := writeTempBackup(t, "payload")
	localHash := mustHash(t, src)
	remote := newMemoryRemote()
	remote.hashResults = []string{"deadbeef", localHash}

	if err := uploadWithIntegrity(src, "db.sql", localHash, remote, nil); err != nil {
		t.Fatal(err)
	}
	if remote.puts != 2 {
		t.Fatalf("puts = %d, want 2", remote.puts)
	}
	if remote.files["db.sql"] != "payload" {
		t.Fatal("verified object missing after retry")
	}
}

func TestUploadWithIntegrityFailsAfterThreeMismatches(t *testing.T) {
	src := writeTempBackup(t, "payload")
	localHash := mustHash(t, src)
	remote := newMemoryRemote()
	remote.hashResults = []string{"aaa", "bbb", "ccc"}

	err := uploadWithIntegrity(src, "db.sql", localHash, remote, nil)
	if err == nil {
		t.Fatal("expected checksum failure")
	}
	if remote.puts != 3 {
		t.Fatalf("puts = %d, want 3", remote.puts)
	}
	if _, ok := remote.files["db.sql"]; ok {
		t.Fatal("final object must not exist after failed verify")
	}
}

func TestRemoteSHA256CommandPrefersSha256sum(t *testing.T) {
	cmd := remoteSHA256Command("/data/x panel/db.sql.part")
	if !strings.Contains(cmd, "sha256sum") || !strings.Contains(cmd, "openssl dgst -sha256") {
		t.Fatalf("command missing checksum tools: %s", cmd)
	}
	if !strings.Contains(cmd, `'/data/x panel/db.sql.part'`) {
		t.Fatalf("path not shell-quoted: %s", cmd)
	}
}

type memoryRemote struct {
	files       map[string]string
	hashResults []string
	puts        int
}

func newMemoryRemote() *memoryRemote {
	return &memoryRemote{files: map[string]string{}}
}

func (m *memoryRemote) put(src, dest string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	m.puts++
	m.files[dest] = string(data)
	return nil
}

func (m *memoryRemote) hash(path string) (string, error) {
	if len(m.hashResults) == 0 {
		return "", fmt.Errorf("no hash scripted for %s", path)
	}
	value := m.hashResults[0]
	m.hashResults = m.hashResults[1:]
	return value, nil
}

func (m *memoryRemote) rename(oldPath, newPath string) error {
	data, ok := m.files[oldPath]
	if !ok {
		return fmt.Errorf("missing %s", oldPath)
	}
	m.files[newPath] = data
	delete(m.files, oldPath)
	return nil
}

func (m *memoryRemote) remove(path string) error {
	delete(m.files, path)
	return nil
}

func writeTempBackup(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "backup.dat")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func mustHash(t *testing.T, path string) string {
	t.Helper()
	hash, err := checksum.FileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	return hash
}
