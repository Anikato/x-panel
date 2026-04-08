package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ArchiveOptions struct {
	SourceDir      string // directory to archive
	OutFile        string // output file path (extension auto-adjusted)
	CompressFormat string // gzip (default), zstd, xz
	EncryptPassword string // if set, encrypt with openssl AES-256-CBC
	ExclusionRules string // newline-separated patterns for tar --exclude
}

// CreateArchive creates a compressed (and optionally encrypted) tar archive.
// Returns the actual output file path (may differ from OutFile due to extension).
func CreateArchive(opts ArchiveOptions) (string, error) {
	if opts.SourceDir == "" {
		return "", fmt.Errorf("source directory is empty")
	}
	if _, err := os.Stat(opts.SourceDir); err != nil {
		return "", fmt.Errorf("source directory not found: %s", opts.SourceDir)
	}

	format := opts.CompressFormat
	if format == "" {
		format = "gzip"
	}

	outFile := adjustExtension(opts.OutFile, format, opts.EncryptPassword != "")

	if err := os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
		return "", fmt.Errorf("create output dir: %v", err)
	}

	tarFile := outFile
	if opts.EncryptPassword != "" {
		tarFile = outFile + ".tmp"
		defer os.Remove(tarFile)
	}

	args := buildTarArgs(format, tarFile, opts.SourceDir, opts.ExclusionRules)
	cmd := exec.Command("tar", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("tar failed: %s", strings.TrimSpace(string(output)))
	}

	if opts.EncryptPassword != "" {
		if err := encryptFile(tarFile, outFile, opts.EncryptPassword); err != nil {
			return "", err
		}
	}

	return outFile, nil
}

func buildTarArgs(format, outFile, sourceDir, exclusionRules string) []string {
	var compressFlag string
	switch format {
	case "zstd":
		compressFlag = "--zstd"
	case "xz":
		compressFlag = "-J"
	default:
		compressFlag = "-z"
	}

	args := []string{"-cf", outFile, compressFlag}

	if exclusionRules != "" {
		for _, rule := range strings.Split(exclusionRules, "\n") {
			rule = strings.TrimSpace(rule)
			if rule != "" {
				args = append(args, "--exclude="+rule)
			}
		}
	}

	args = append(args, "-C", filepath.Dir(sourceDir), filepath.Base(sourceDir))
	return args
}

func adjustExtension(outFile, format string, encrypted bool) string {
	base := strings.TrimSuffix(outFile, filepath.Ext(outFile))
	ext2 := filepath.Ext(base)
	if ext2 == ".tar" {
		base = strings.TrimSuffix(base, ext2)
	}

	var ext string
	switch format {
	case "zstd":
		ext = ".tar.zst"
	case "xz":
		ext = ".tar.xz"
	default:
		ext = ".tar.gz"
	}

	result := base + ext
	if encrypted {
		result += ".enc"
	}
	return result
}

func encryptFile(src, dst, password string) error {
	cmd := exec.Command("openssl", "enc", "-aes-256-cbc", "-salt", "-pbkdf2",
		"-in", src, "-out", dst, "-pass", "pass:"+password)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("encryption failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// DecryptFile decrypts an AES-256-CBC encrypted file.
func DecryptFile(src, dst, password string) error {
	cmd := exec.Command("openssl", "enc", "-aes-256-cbc", "-d", "-salt", "-pbkdf2",
		"-in", src, "-out", dst, "-pass", "pass:"+password)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("decryption failed: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

// SupportedFormats returns the list of supported compression formats.
func SupportedFormats() []string {
	return []string{"gzip", "zstd", "xz"}
}
