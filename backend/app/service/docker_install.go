package service

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"xpanel/global"
)

var (
	dockerInstallMu      sync.Mutex
	dockerInstallRunning bool
	dockerInstallLog     []string
)

func RunDockerInstall() (string, error) {
	dockerInstallMu.Lock()
	if dockerInstallRunning {
		dockerInstallMu.Unlock()
		return "", fmt.Errorf("docker installation is already in progress")
	}
	dockerInstallRunning = true
	dockerInstallLog = []string{}
	dockerInstallMu.Unlock()

	defer func() {
		dockerInstallMu.Lock()
		dockerInstallRunning = false
		dockerInstallMu.Unlock()
	}()

	appendDockerLog("Starting Docker installation using official script...")
	global.LOG.Info("Starting Docker installation")

	cmd := exec.Command("bash", "-c", "curl -fsSL https://get.docker.com | bash")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		appendDockerLog(fmt.Sprintf("Failed to create stdout pipe: %v", err))
		return "", err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		appendDockerLog(fmt.Sprintf("Failed to start installation: %v", err))
		return "", err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		appendDockerLog(line)
	}

	if err := cmd.Wait(); err != nil {
		msg := fmt.Sprintf("Docker installation failed: %v", err)
		appendDockerLog(msg)
		global.LOG.Warn(msg)
		return strings.Join(dockerInstallLog, "\n"), err
	}

	appendDockerLog("Docker installation completed successfully")

	// Enable and start Docker service
	exec.Command("systemctl", "enable", "docker").Run()
	exec.Command("systemctl", "start", "docker").Run()
	appendDockerLog("Docker service enabled and started")

	global.LOG.Info("Docker installation completed")
	return strings.Join(dockerInstallLog, "\n"), nil
}

func GetDockerInstallLog() (string, bool) {
	dockerInstallMu.Lock()
	defer dockerInstallMu.Unlock()
	return strings.Join(dockerInstallLog, "\n"), dockerInstallRunning
}

func appendDockerLog(line string) {
	dockerInstallMu.Lock()
	defer dockerInstallMu.Unlock()
	dockerInstallLog = append(dockerInstallLog, line)
}
