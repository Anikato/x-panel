package cloud_storage

import (
	"fmt"
	"regexp"
	"strings"
)

const verifiedUploadAttempts = 3

type remoteStore interface {
	put(src, dest string) error
	hash(path string) (string, error)
	rename(oldPath, newPath string) error
	remove(path string) error
}

func stagingObjectPath(target string) string {
	return strings.TrimRight(target, "/") + ".part"
}

func uploadWithIntegrity(src, target, localHash string, remote remoteStore, logf func(string, ...any)) error {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if localHash == "" {
		return fmt.Errorf("local sha256 is empty")
	}

	staging := stagingObjectPath(target)
	var lastErr error
	for attempt := 1; attempt <= verifiedUploadAttempts; attempt++ {
		logf("sftp upload attempt %d/%d to %s", attempt, verifiedUploadAttempts, staging)
		if err := remote.put(src, staging); err != nil {
			lastErr = fmt.Errorf("upload staging file failed: %w", err)
			logf("%v", lastErr)
			_ = remote.remove(staging)
			continue
		}

		remoteHash, err := remote.hash(staging)
		if err != nil {
			lastErr = fmt.Errorf("remote sha256 failed: %w", err)
			logf("%v", lastErr)
			_ = remote.remove(staging)
			continue
		}
		logf("local sha256=%s remote sha256=%s", localHash, remoteHash)
		if !strings.EqualFold(remoteHash, localHash) {
			lastErr = fmt.Errorf("sha256 mismatch: local %s remote %s", localHash, remoteHash)
			logf("%v", lastErr)
			_ = remote.remove(staging)
			continue
		}

		if err := remote.rename(staging, target); err != nil {
			lastErr = fmt.Errorf("rename verified file failed: %w", err)
			logf("%v", lastErr)
			_ = remote.remove(staging)
			continue
		}
		logf("sftp object verified and published: %s", target)
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("sftp upload failed after %d attempts", verifiedUploadAttempts)
	}
	return lastErr
}

var (
	sha256sumPattern = regexp.MustCompile(`(?i)\b([a-f0-9]{64})\b`)
	opensslPattern   = regexp.MustCompile(`(?i)SHA256\([^)]+\)=\s*([a-f0-9]{64})`)
)

func parseRemoteSHA256(output string) (string, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return "", fmt.Errorf("empty remote checksum output")
	}
	if match := opensslPattern.FindStringSubmatch(output); len(match) == 2 {
		return strings.ToLower(match[1]), nil
	}
	if match := sha256sumPattern.FindStringSubmatch(output); len(match) == 2 {
		return strings.ToLower(match[1]), nil
	}
	return "", fmt.Errorf("unrecognized checksum output: %s", output)
}

func remoteSHA256Command(path string) string {
	quoted := shellSingleQuote(path)
	return fmt.Sprintf("sha256sum -- %s 2>/dev/null || openssl dgst -sha256 %s", quoted, quoted)
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
