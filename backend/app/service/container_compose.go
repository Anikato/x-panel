package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"xpanel/app/dto"
	"xpanel/global"
)

type IComposeService interface {
	ListComposeProjects() ([]dto.ComposeInfo, error)
	CreateCompose(req dto.ComposeCreate) error
	OperateCompose(req dto.ComposeOperate) error
	GetComposeContent(path string) (string, error)
}

func NewIComposeService() IComposeService {
	return &ComposeService{}
}

type ComposeService struct{}

func (s *ComposeService) ListComposeProjects() ([]dto.ComposeInfo, error) {
	cmd := exec.Command("docker", "compose", "ls", "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker compose ls failed: %s", string(output))
	}

	var items []dto.ComposeInfo
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" || line == "[]" {
			continue
		}
		items = append(items, dto.ComposeInfo{Name: line})
	}
	return items, nil
}

func (s *ComposeService) CreateCompose(req dto.ComposeCreate) error {
	dir := filepath.Dir(req.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if req.Content != "" {
		if err := os.WriteFile(req.Path, []byte(req.Content), 0644); err != nil {
			return err
		}
	}
	cmd := exec.Command("docker", "compose", "-f", req.Path, "up", "-d")
	output, err := cmd.CombinedOutput()
	if err != nil {
		global.LOG.Errorf("compose up failed: %s", string(output))
		return fmt.Errorf("%s", string(output))
	}
	return nil
}

func (s *ComposeService) OperateCompose(req dto.ComposeOperate) error {
	var args []string
	switch req.Operation {
	case "up":
		args = []string{"compose", "-f", req.Path, "up", "-d"}
	case "down":
		args = []string{"compose", "-f", req.Path, "down"}
	case "restart":
		args = []string{"compose", "-f", req.Path, "restart"}
	case "stop":
		args = []string{"compose", "-f", req.Path, "stop"}
	default:
		return fmt.Errorf("unknown operation: %s", req.Operation)
	}
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}
	return nil
}

func (s *ComposeService) GetComposeContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
