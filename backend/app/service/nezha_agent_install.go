package service

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"xpanel/app/dto"
	"xpanel/app/version"
)

const (
	agentBundleBinary = "nezha-agent/nezha-agent"
	agentBundleUnit   = "xpanel-nezha-agent.service"
	maxAgentBytes     = 256 << 20
	maxUnitBytes      = 1 << 20
)

type agentBundleInstallDeps struct {
	AgentPath string
	UnitPath  string
	Runner    nezhaCmdRunner
}

// Install restores only the bundled Agent assets. Configuration and service
// lifecycle remain the responsibility of Configure after this call succeeds.
func (s *NezhaAgentService) Install() error {
	// Share both the in-process guard and cross-process file lock with full upgrades.
	releaseUpgrade, err := (&UpgradeService{}).beginUpgrade()
	if err != nil {
		return err
	}
	defer releaseUpgrade()

	// Conflict and target checks must finish before any release request or write.
	if err := s.ensureNoConflicts(); err != nil {
		return err
	}
	if err := preflightLiveBinary(s.binaryPath); err != nil {
		return fmt.Errorf("agent install preflight: %w", err)
	}
	if err := preflightLiveBinary(s.unitPath); err != nil {
		return fmt.Errorf("unit install preflight: %w", err)
	}
	wasActive, err := s.isActive()
	if err != nil {
		return err
	}
	wasEnabled, err := s.isEnabled()
	if err != nil {
		return err
	}

	archivePath, cleanup, err := s.installBundle()
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return err
	}
	if wasActive {
		if err := s.systemctl("stop", s.unit); err != nil {
			return err
		}
		if err := s.waitUntilInactive(); err != nil {
			return s.restoreActive(true, err)
		}
	}
	if err := applyNezhaAgentBundle(agentBundleInstallDeps{
		AgentPath: s.binaryPath,
		UnitPath:  s.unitPath,
		Runner:    s.runner,
	}, archivePath); err != nil {
		return s.restoreActive(wasActive, err)
	}

	// Replacing a unit file normally preserves its enablement symlink. Repair it
	// explicitly only if systemd reports that the state changed unexpectedly.
	enabledNow, err := s.isEnabled()
	if err != nil {
		return s.restoreActive(wasActive, err)
	}
	if enabledNow != wasEnabled {
		verb := "disable"
		if wasEnabled {
			verb = "enable"
		}
		if err := s.systemctl(verb, s.unit); err != nil {
			return s.restoreActive(wasActive, err)
		}
	}
	if wasActive {
		return s.systemctl("start", s.unit)
	}
	return nil
}

func exactAgentPackageURLs(baseURL, releaseVersion, arch string) (string, string) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	name := fmt.Sprintf("xpanel-%s-linux-%s.tar.gz", releaseVersion, arch)
	downloadURL := fmt.Sprintf("%s/releases/%s/%s", baseURL, releaseVersion, name)
	return downloadURL, downloadURL + ".sha256"
}

func normalizedAgentReleaseVersion(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "dev" {
		return "", fmt.Errorf("cannot install Agent for X-Panel version %q", value)
	}
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	return value, nil
}

func downloadCurrentNezhaAgentBundle() (string, func(), error) {
	releaseVersion, err := normalizedAgentReleaseVersion(version.Version)
	if err != nil {
		return "", nil, err
	}
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return "", nil, fmt.Errorf("unsupported Agent architecture %q", runtime.GOARCH)
	}

	releaseURL, _ := settingRepo.GetValueByKey("UpgradeURL")
	releaseURL = strings.TrimSpace(releaseURL)
	if releaseURL == "" {
		releaseURL = DefaultUpdateBaseURL
	}
	token, _ := settingRepo.GetValueByKey("GitHubToken")

	var downloadURL, checksumURL string
	if isGitHubURL(releaseURL) {
		repoName := extractGitHubRepo(releaseURL)
		if repoName == "" {
			repoName = DefaultGitHubRepo
		}
		downloadURL, checksumURL, err = exactGitHubAgentAssetURLs(repoName, releaseVersion, runtime.GOARCH, token)
	} else {
		downloadURL, checksumURL = exactAgentPackageURLs(releaseURL, releaseVersion, runtime.GOARCH)
	}
	if err != nil {
		return "", nil, err
	}
	if downloadURL == "" || checksumURL == "" {
		return "", nil, fmt.Errorf("current release is missing Agent package or checksum")
	}

	tmpDir, err := os.MkdirTemp("", "xpanel-agent-install-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmpDir) }
	archivePath := filepath.Join(tmpDir, "xpanel.tar.gz")
	checksumPath := archivePath + ".sha256"
	if err := downloadFile(downloadURL, archivePath, token); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("download Agent package: %w", err)
	}
	if err := downloadFile(checksumURL, checksumPath, token); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("download Agent checksum: %w", err)
	}
	if err := verifySHA256(archivePath, checksumPath); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("verify Agent package: %w", err)
	}
	return archivePath, cleanup, nil
}

func exactGitHubAgentAssetURLs(repoName, releaseVersion, arch, token string) (string, string, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/releases/tags/%s", GitHubAPIBase, repoName, url.PathEscape(releaseVersion))
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "X-Panel/"+version.Version)
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("resolve current GitHub release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("resolve current GitHub release: HTTP %d", resp.StatusCode)
	}
	var release dto.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}
	wantArchive := fmt.Sprintf("xpanel-%s-linux-%s.tar.gz", releaseVersion, arch)
	wantChecksum := wantArchive + ".sha256"
	var downloadURL, checksumURL string
	for _, asset := range release.Assets {
		assetURL := asset.BrowserDownloadURL
		if token != "" && asset.URL != "" {
			assetURL = asset.URL
		}
		switch asset.Name {
		case wantArchive:
			downloadURL = assetURL
		case wantChecksum:
			checksumURL = assetURL
		}
	}
	return downloadURL, checksumURL, nil
}

func extractNezhaAgentBundle(archivePath, destDir string) (agentPath, unitPath string, err error) {
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return "", "", err
	}
	f, err := os.Open(archivePath)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, nextErr := tr.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return "", "", nextErr
		}
		rel, normalizeErr := normalizeComponentArchiveName(hdr.Name)
		if normalizeErr != nil {
			return "", "", normalizeErr
		}
		if hdr.Typeflag == tar.TypeDir {
			continue
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			return "", "", fmt.Errorf("archive entry %q is not a regular file", hdr.Name)
		}

		limit := int64(0)
		switch rel {
		case agentBundleBinary:
			if agentPath != "" {
				return "", "", fmt.Errorf("duplicate archive entry %q", rel)
			}
			limit = maxAgentBytes
		case agentBundleUnit:
			if unitPath != "" {
				return "", "", fmt.Errorf("duplicate archive entry %q", rel)
			}
			limit = maxUnitBytes
		default:
			continue
		}
		if hdr.Size < 0 || hdr.Size > limit {
			return "", "", fmt.Errorf("archive entry %q exceeds size limit", rel)
		}
		dst := filepath.Join(destDir, filepath.FromSlash(rel))
		if !isPathInside(destDir, dst) {
			return "", "", fmt.Errorf("archive entry %q escapes destination", rel)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return "", "", err
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return "", "", err
		}
		n, copyErr := io.Copy(out, io.LimitReader(tr, limit+1))
		closeErr := out.Close()
		if copyErr != nil {
			return "", "", copyErr
		}
		if closeErr != nil {
			return "", "", closeErr
		}
		if n > limit {
			return "", "", fmt.Errorf("archive entry %q exceeds size limit", rel)
		}
		if rel == agentBundleBinary {
			agentPath = dst
		} else {
			unitPath = dst
		}
	}
	if agentPath == "" || unitPath == "" {
		return "", "", fmt.Errorf("release package is missing bundled Agent assets")
	}
	return agentPath, unitPath, nil
}

func applyNezhaAgentBundle(deps agentBundleInstallDeps, archivePath string) (err error) {
	tmpDir, err := os.MkdirTemp("", "xpanel-agent-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	agentSrc, unitSrc, err := extractNezhaAgentBundle(archivePath, tmpDir)
	if err != nil {
		return err
	}
	if err := validateComponentELF(agentSrc); err != nil {
		return err
	}
	if err := preflightLiveBinary(deps.AgentPath); err != nil {
		return err
	}
	if err := preflightLiveBinary(deps.UnitPath); err != nil {
		return err
	}
	if err := ensureAgentParentDir(deps.AgentPath); err != nil {
		return err
	}
	if err := ensureInstallParent(filepath.Dir(deps.UnitPath), 0o755); err != nil {
		return err
	}

	agentBackup, err := backupAgentInstallFile(deps.AgentPath, 0o755)
	if err != nil {
		return err
	}
	defer agentBackup.cleanup()
	unitBackup, err := backupAgentInstallFile(deps.UnitPath, 0o644)
	if err != nil {
		return err
	}
	defer unitBackup.cleanup()
	mutated := false
	rollback := func(cause error) error {
		if !mutated {
			return cause
		}
		restoreErr := errors.Join(
			restoreAgentInstallFile(agentBackup),
			restoreAgentInstallFile(unitBackup),
		)
		if deps.Runner != nil {
			_, _ = deps.Runner.CombinedOutput("systemctl", "daemon-reload")
		}
		return errors.Join(cause, restoreErr)
	}
	if err := replaceAgentInstallFile(agentSrc, deps.AgentPath, 0o755); err != nil {
		return err
	}
	mutated = true
	if err := replaceAgentInstallFile(unitSrc, deps.UnitPath, 0o644); err != nil {
		return rollback(err)
	}
	if deps.Runner == nil {
		return rollback(fmt.Errorf("systemctl runner is unavailable"))
	}
	if out, err := deps.Runner.CombinedOutput("systemctl", "daemon-reload"); err != nil {
		return rollback(fmt.Errorf("systemctl daemon-reload: %w (%s)", err, strings.TrimSpace(string(out))))
	}
	return nil
}

func ensureInstallParent(dir string, mode os.FileMode) error {
	info, err := os.Lstat(dir)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("install parent %s is not a real directory", dir)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(dir, mode)
}

type agentInstallBackup struct {
	livePath string
	tmpPath  string
	mode     os.FileMode
	existed  bool
}

func (b agentInstallBackup) cleanup() { _ = os.Remove(b.tmpPath) }

func backupAgentInstallFile(livePath string, mode os.FileMode) (agentInstallBackup, error) {
	b := agentInstallBackup{livePath: livePath, mode: mode}
	info, err := os.Lstat(livePath)
	if os.IsNotExist(err) {
		return b, nil
	}
	if err != nil {
		return b, err
	}
	if !info.Mode().IsRegular() {
		return b, fmt.Errorf("install target %s is not a regular file", livePath)
	}
	b.existed = true
	f, err := os.CreateTemp(filepath.Dir(livePath), filepath.Base(livePath)+".backup-*")
	if err != nil {
		return b, err
	}
	b.tmpPath = f.Name()
	if err := f.Close(); err != nil {
		b.cleanup()
		return b, err
	}
	if err := copyFile(livePath, b.tmpPath); err != nil {
		b.cleanup()
		return b, err
	}
	if err := os.Chmod(b.tmpPath, mode); err != nil {
		b.cleanup()
		return b, err
	}
	return b, nil
}

func restoreAgentInstallFile(b agentInstallBackup) error {
	if !b.existed {
		if err := os.Remove(b.livePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return replaceAgentInstallFile(b.tmpPath, b.livePath, b.mode)
}

func replaceAgentInstallFile(src, dst string, mode os.FileMode) error {
	if err := preflightLiveBinary(dst); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(dst), filepath.Base(dst)+".install-*")
	if err != nil {
		return err
	}
	staging := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(staging)
		return err
	}
	defer os.Remove(staging)
	if err := copyFile(src, staging); err != nil {
		return err
	}
	if err := os.Chmod(staging, mode); err != nil {
		return err
	}
	return os.Rename(staging, dst)
}
