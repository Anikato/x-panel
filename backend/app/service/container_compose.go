package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"

	"gorm.io/gorm"
)

type IComposeService interface {
	ListComposeProjects() ([]dto.ComposeInfo, error)
	CreateCompose(req dto.ComposeCreate) error
	OperateCompose(req dto.ComposeOperate) error
	GetComposeContent(id uint) (string, error)
	UpdateComposeContent(id uint, content string) error
	DeleteCompose(id uint) error
}

type ComposeService struct {
	repo repo.IComposeRepo
	run  func(name, path string, args ...string) (string, error)
	list func() ([]composeLSItem, error)
	root func() string
}

func NewIComposeService() IComposeService {
	return &ComposeService{
		repo: repo.NewIComposeRepo(),
		run:  defaultComposeRun,
		list: defaultComposeList,
		root: defaultComposeRoot,
	}
}

func defaultComposeRoot() string {
	return filepath.Join(global.CONF.System.DataDir, "compose")
}

func defaultComposeRun(name, path string, args ...string) (string, error) {
	cmdArgs := append([]string{"compose", "-p", name, "-f", path}, args...)
	cmd := exec.Command("docker", cmdArgs...)
	cmd.Dir = filepath.Dir(path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s", truncateComposeOutput(string(output)))
	}
	return string(output), nil
}

func defaultComposeList() ([]composeLSItem, error) {
	cmd := exec.Command("docker", "compose", "ls", "--format", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker compose ls failed: %s", truncateComposeOutput(string(output)))
	}
	return parseComposeLS(string(output))
}

func truncateComposeOutput(s string) string {
	const max = 4000
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n...[truncated]"
}

var composeLocks sync.Map

func lockCompose(name string) (func(), error) {
	v, _ := composeLocks.LoadOrStore(name, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	if !mu.TryLock() {
		return nil, fmt.Errorf("compose project %s is busy", name)
	}
	return mu.Unlock, nil
}

func isComposeNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

func (s *ComposeService) requireNameFree(name string) error {
	existing, err := s.repo.GetByName(name)
	if err != nil && !isComposeNotFound(err) {
		return err
	}
	if existing != nil {
		return fmt.Errorf("compose project already exists: %s", name)
	}
	return nil
}

func (s *ComposeService) ListComposeProjects() ([]dto.ComposeInfo, error) {
	registered, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	live, err := s.list()
	if err != nil {
		if len(registered) == 0 {
			return nil, err
		}
		if global.LOG != nil {
			global.LOG.Errorf("docker compose ls failed: %v", err)
		}
		live = nil
	}

	liveByName := make(map[string]composeLSItem, len(live))
	for _, item := range live {
		liveByName[item.Name] = item
	}

	out := make([]dto.ComposeInfo, 0, len(registered)+len(live))
	for _, row := range registered {
		info := dto.ComposeInfo{
			ID:     row.ID,
			Name:   row.Name,
			Path:   row.Path,
			Source: row.Source,
			Status: "stopped",
		}
		if !row.CreatedAt.IsZero() {
			info.Created = row.CreatedAt.Format("2006-01-02 15:04:05")
		}
		if liveItem, ok := liveByName[row.Name]; ok {
			info.Status = liveItem.Status
			delete(liveByName, row.Name)
		}
		out = append(out, info)
	}
	for _, liveItem := range liveByName {
		out = append(out, dto.ComposeInfo{
			Name:   liveItem.Name,
			Path:   firstConfigFile(liveItem.ConfigFiles),
			Source: "unmanaged",
			Status: liveItem.Status,
		})
	}
	return out, nil
}

func (s *ComposeService) CreateCompose(req dto.ComposeCreate) error {
	if err := validateComposeName(req.Name); err != nil {
		return err
	}
	hasContent := req.Content != ""
	hasPath := req.Path != ""
	if hasContent == hasPath {
		if hasContent {
			return fmt.Errorf("compose content and path cannot both be set")
		}
		return fmt.Errorf("compose content or path is required")
	}
	if err := s.requireNameFree(req.Name); err != nil {
		return err
	}

	if hasContent {
		live, err := s.list()
		if err != nil {
			return err
		}
		for _, item := range live {
			if item.Name == req.Name {
				return fmt.Errorf("compose project already exists: %s", req.Name)
			}
		}
		dir := filepath.Join(s.root(), req.Name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		path := filepath.Join(dir, "docker-compose.yml")
		if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
			return err
		}
		item := &model.ComposeProject{Name: req.Name, Path: path, Source: "created"}
		if err := s.repo.Create(item); err != nil {
			return err
		}
		// Lock around first up; if busy, keep the row so the user can retry.
		unlock, err := lockCompose(req.Name)
		if err != nil {
			return err
		}
		defer unlock()
		if _, err := s.run(req.Name, path, "up", "-d"); err != nil {
			return err
		}
		return nil
	}

	clean, err := validateComposePath(req.Path)
	if err != nil {
		return err
	}
	return s.repo.Create(&model.ComposeProject{Name: req.Name, Path: clean, Source: "attached"})
}

func (s *ComposeService) resolveProject(req dto.ComposeOperate) (*model.ComposeProject, error) {
	if req.ID != 0 {
		return s.repo.Get(req.ID)
	}
	if req.Name != "" {
		return s.repo.GetByName(req.Name)
	}
	return nil, fmt.Errorf("compose project id or name is required")
}

func (s *ComposeService) OperateCompose(req dto.ComposeOperate) error {
	if req.Operation == "attach" {
		if err := validateComposeName(req.Name); err != nil {
			return err
		}
		clean, err := validateComposePath(req.Path)
		if err != nil {
			return err
		}
		if err := s.requireNameFree(req.Name); err != nil {
			return err
		}
		return s.repo.Create(&model.ComposeProject{Name: req.Name, Path: clean, Source: "attached"})
	}

	project, err := s.resolveProject(req)
	if err != nil {
		return err
	}
	unlock, err := lockCompose(project.Name)
	if err != nil {
		return err
	}
	defer unlock()

	switch req.Operation {
	case "up":
		_, err = s.run(project.Name, project.Path, "up", "-d")
	case "stop", "restart", "down", "pull":
		_, err = s.run(project.Name, project.Path, req.Operation)
	case "update":
		if _, err = s.run(project.Name, project.Path, "pull"); err != nil {
			return err
		}
		_, err = s.run(project.Name, project.Path, "up", "-d")
	default:
		return fmt.Errorf("unknown operation: %s", req.Operation)
	}
	return err
}

func (s *ComposeService) GetComposeContent(id uint) (string, error) {
	project, err := s.repo.Get(id)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(project.Path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *ComposeService) UpdateComposeContent(id uint, content string) error {
	project, err := s.repo.Get(id)
	if err != nil {
		return err
	}
	unlock, err := lockCompose(project.Name)
	if err != nil {
		return err
	}
	defer unlock()

	project, err = s.repo.Get(id)
	if err != nil {
		return err
	}
	return os.WriteFile(project.Path, []byte(content), 0644)
}

func (s *ComposeService) DeleteCompose(id uint) error {
	project, err := s.repo.Get(id)
	if err != nil {
		return err
	}
	unlock, err := lockCompose(project.Name)
	if err != nil {
		return err
	}
	defer unlock()

	if _, statErr := os.Stat(project.Path); statErr == nil {
		if _, err := s.run(project.Name, project.Path, "down"); err != nil {
			return err
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}

	if shouldRemoveComposeDir(project.Source, project.Path, s.root()) {
		dir := filepath.Dir(project.Path)
		root := filepath.Clean(s.root())
		if dir != root && isUnderDir(dir, root) {
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
		}
	}
	return s.repo.Delete(id)
}

var composeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

func validateComposeName(name string) error {
	if !composeNamePattern.MatchString(name) {
		return fmt.Errorf("invalid compose project name: %s", name)
	}
	return nil
}

func validateComposePath(path string) (string, error) {
	if path == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("compose path must be absolute")
	}
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) || strings.Contains(clean, "..") {
		return "", fmt.Errorf("invalid compose path")
	}
	info, err := os.Stat(clean)
	if err != nil {
		return "", fmt.Errorf("compose file not found: %s", clean)
	}
	if info.IsDir() {
		return "", fmt.Errorf("compose path is a directory: %s", clean)
	}
	return clean, nil
}

func shouldRemoveComposeDir(source, path, root string) bool {
	if source != "created" {
		return false
	}
	return isUnderDir(path, root)
}

func isUnderDir(path, root string) bool {
	cleanPath := filepath.Clean(path)
	cleanRoot := filepath.Clean(root)
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

type composeLSItem struct {
	Name        string `json:"Name"`
	Status      string `json:"Status"`
	ConfigFiles string `json:"ConfigFiles"`
}

func parseComposeLS(raw string) ([]composeLSItem, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	if strings.HasPrefix(raw, "[") {
		var items []composeLSItem
		if err := json.Unmarshal([]byte(raw), &items); err != nil {
			return nil, err
		}
		return items, nil
	}
	var items []composeLSItem
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "[]" {
			continue
		}
		var item composeLSItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}
		if item.Name != "" {
			items = append(items, item)
		}
	}
	return items, nil
}

func firstConfigFile(configFiles string) string {
	parts := strings.Split(configFiles, ",")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}
