package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type LinuxUser struct {
	Username string `json:"username"`
	UID      int    `json:"uid"`
	GID      int    `json:"gid"`
	Comment  string `json:"comment"`
	Home     string `json:"home"`
	Shell    string `json:"shell"`
	Groups   string `json:"groups"`
	IsSystem bool   `json:"isSystem"`
	IsSudo   bool   `json:"isSudo"`
}

type LinuxUserCreate struct {
	Username   string `json:"username" validate:"required"`
	Password   string `json:"password"`
	Home       string `json:"home"`
	Shell      string `json:"shell"`
	Comment    string `json:"comment"`
	CreateHome bool   `json:"createHome"`
	Sudo       bool   `json:"sudo"`
}

type LinuxUserUpdate struct {
	Username string `json:"username" validate:"required"`
	Shell    string `json:"shell"`
	Comment  string `json:"comment"`
	Home     string `json:"home"`
	Password string `json:"password"`
	Sudo     *bool  `json:"sudo"`
}

type LinuxUserDelete struct {
	Username   string `json:"username" validate:"required"`
	RemoveHome bool   `json:"removeHome"`
}

type IHostUserService interface {
	List(showSystem bool) ([]LinuxUser, error)
	Create(req LinuxUserCreate) error
	Update(req LinuxUserUpdate) error
	Delete(req LinuxUserDelete) error
	ListShells() ([]string, error)
	ListGroups() ([]string, error)
}

type HostUserService struct{}

func NewIHostUserService() IHostUserService { return &HostUserService{} }

func (s *HostUserService) List(showSystem bool) ([]LinuxUser, error) {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var users []LinuxUser
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue
		}
		uid, _ := strconv.Atoi(parts[2])
		gid, _ := strconv.Atoi(parts[3])
		isSystem := uid < 1000 && uid != 0

		if !showSystem && isSystem {
			continue
		}

		groups := ""
		isSudo := false
		if out, err := exec.Command("id", "-Gn", parts[0]).Output(); err == nil {
			groups = strings.TrimSpace(string(out))
			for _, g := range strings.Fields(groups) {
				if g == "sudo" || g == "wheel" {
					isSudo = true
					break
				}
			}
		}

		users = append(users, LinuxUser{
			Username: parts[0],
			UID:      uid,
			GID:      gid,
			Comment:  parts[4],
			Home:     parts[5],
			Shell:    parts[6],
			Groups:   groups,
			IsSystem: isSystem || uid == 0,
			IsSudo:   isSudo,
		})
	}
	return users, nil
}

func (s *HostUserService) Create(req LinuxUserCreate) error {
	args := []string{}
	if req.Comment != "" {
		args = append(args, "-c", req.Comment)
	}
	if req.Home != "" {
		args = append(args, "-d", req.Home)
	}
	if req.Shell != "" {
		args = append(args, "-s", req.Shell)
	}
	if req.CreateHome {
		args = append(args, "-m")
	}
	args = append(args, req.Username)

	out, err := exec.Command("useradd", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("useradd failed: %s", strings.TrimSpace(string(out)))
	}

	if req.Password != "" {
		cmd := exec.Command("chpasswd")
		cmd.Stdin = strings.NewReader(fmt.Sprintf("%s:%s", req.Username, req.Password))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("set password failed: %s", strings.TrimSpace(string(out)))
		}
	}

	if req.Sudo {
		s.setSudo(req.Username, true)
	}

	return nil
}

func (s *HostUserService) Update(req LinuxUserUpdate) error {
	args := []string{}
	if req.Shell != "" {
		args = append(args, "-s", req.Shell)
	}
	if req.Comment != "" {
		args = append(args, "-c", req.Comment)
	}
	if req.Home != "" {
		args = append(args, "-d", req.Home)
	}

	if len(args) > 0 {
		args = append(args, req.Username)
		out, err := exec.Command("usermod", args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("usermod failed: %s", strings.TrimSpace(string(out)))
		}
	}

	if req.Password != "" {
		cmd := exec.Command("chpasswd")
		cmd.Stdin = strings.NewReader(fmt.Sprintf("%s:%s", req.Username, req.Password))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("set password failed: %s", strings.TrimSpace(string(out)))
		}
	}

	if req.Sudo != nil {
		s.setSudo(req.Username, *req.Sudo)
	}

	return nil
}

func (s *HostUserService) setSudo(username string, enable bool) {
	sudoGroup := "sudo"
	if out, _ := exec.Command("getent", "group", "wheel").Output(); len(out) > 0 {
		sudoGroup = "wheel"
	}
	if enable {
		exec.Command("usermod", "-aG", sudoGroup, username).Run()
	} else {
		exec.Command("gpasswd", "-d", username, sudoGroup).Run()
	}
}

func (s *HostUserService) Delete(req LinuxUserDelete) error {
	args := []string{}
	if req.RemoveHome {
		args = append(args, "-r")
	}
	args = append(args, req.Username)

	out, err := exec.Command("userdel", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("userdel failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *HostUserService) ListShells() ([]string, error) {
	f, err := os.Open("/etc/shells")
	if err != nil {
		return []string{"/bin/bash", "/bin/sh", "/usr/sbin/nologin"}, nil
	}
	defer f.Close()

	var shells []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			shells = append(shells, line)
		}
	}
	return shells, nil
}

func (s *HostUserService) ListGroups() ([]string, error) {
	f, err := os.Open("/etc/group")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var groups []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) >= 1 && parts[0] != "" {
			groups = append(groups, parts[0])
		}
	}
	return groups, nil
}
