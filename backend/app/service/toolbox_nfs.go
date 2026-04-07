package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"xpanel/app/dto"
	"xpanel/buserr"
	nfsutil "xpanel/utils/nfs"
)

const exportsPath = "/etc/exports"

type INfsService interface {
	GetStatus() (*dto.ServiceStatus, error)
	Install() error
	Uninstall() error
	Operate(req dto.ServiceOperate) error

	ListExports() ([]dto.NfsExport, error)
	CreateExport(req dto.NfsExportCreate) error
	UpdateExport(req dto.NfsExportUpdate) error
	DeleteExport(req dto.NfsExportDelete) error

	GetConnections() (*dto.NfsConnections, error)
}

type NfsService struct{}

func NewINfsService() INfsService { return &NfsService{} }

// ====== Service Management ======

func (s *NfsService) GetStatus() (*dto.ServiceStatus, error) {
	st := &dto.ServiceStatus{}

	out, err := exec.Command("dpkg", "-l", "nfs-kernel-server").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "ii  nfs-kernel-server") {
		return st, nil
	}
	st.IsInstalled = true

	if out, err := exec.Command("systemctl", "is-active", "nfs-kernel-server").Output(); err == nil {
		st.IsRunning = strings.TrimSpace(string(out)) == "active"
	}

	if out, err := exec.Command("nfsstat", "--version").CombinedOutput(); err == nil {
		st.Version = strings.TrimSpace(string(out))
	} else {
		if out2, err2 := exec.Command("rpc.nfsd", "--version").CombinedOutput(); err2 == nil {
			st.Version = strings.TrimSpace(string(out2))
		} else {
			st.Version = "nfs-kernel-server"
		}
	}

	if out, err := exec.Command("systemctl", "is-enabled", "nfs-kernel-server").Output(); err == nil {
		st.AutoStart = strings.TrimSpace(string(out)) == "enabled"
	}

	return st, nil
}

func (s *NfsService) Install() error {
	out, err := exec.Command("apt", "install", "-y", "nfs-kernel-server").CombinedOutput()
	if err != nil {
		return buserr.WithDetail("ErrNfsInstall", string(out), err)
	}
	return nil
}

func (s *NfsService) Uninstall() error {
	out, err := exec.Command("apt", "remove", "-y", "nfs-kernel-server").CombinedOutput()
	if err != nil {
		return buserr.WithDetail("ErrNfsUninstall", string(out), err)
	}
	return nil
}

func (s *NfsService) Operate(req dto.ServiceOperate) error {
	out, err := exec.Command("systemctl", req.Operation, "nfs-kernel-server").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s nfs-kernel-server failed: %s", req.Operation, strings.TrimSpace(string(out)))
	}
	return nil
}

// ====== Export Management ======

func (s *NfsService) ListExports() ([]dto.NfsExport, error) {
	exports, err := nfsutil.Parse(exportsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []dto.NfsExport{}, nil
		}
		return nil, buserr.WithErr("ErrNfsReadExports", err)
	}

	var result []dto.NfsExport
	for _, e := range exports {
		export := dto.NfsExport{
			Path:    e.Path,
			Comment: e.Comment,
		}
		for _, c := range e.Clients {
			export.Clients = append(export.Clients, dto.NfsClient{
				Host:    c.Host,
				Options: c.Options,
			})
		}
		result = append(result, export)
	}
	return result, nil
}

func (s *NfsService) CreateExport(req dto.NfsExportCreate) error {
	exports, err := nfsutil.Parse(exportsPath)
	if err != nil && !os.IsNotExist(err) {
		return buserr.WithErr("ErrNfsReadExports", err)
	}

	for _, e := range exports {
		if e.Path == req.Path {
			return buserr.WithName("ErrNfsExportExist", req.Path)
		}
	}

	if req.CreateDir {
		if err := os.MkdirAll(req.Path, 0755); err != nil {
			return fmt.Errorf("create directory failed: %v", err)
		}
	}

	export := nfsutil.Export{
		Path:    req.Path,
		Comment: req.Comment,
	}
	for _, c := range req.Clients {
		export.Clients = append(export.Clients, nfsutil.Client{
			Host:    c.Host,
			Options: c.Options,
		})
	}

	return s.safeWriteExport(func(list []nfsutil.Export) []nfsutil.Export {
		return append(list, export)
	})
}

func (s *NfsService) UpdateExport(req dto.NfsExportUpdate) error {
	return s.safeWriteExport(func(list []nfsutil.Export) []nfsutil.Export {
		var result []nfsutil.Export
		for _, e := range list {
			if e.Path == req.OrigPath {
				updated := nfsutil.Export{
					Path:    req.Path,
					Comment: req.Comment,
				}
				for _, c := range req.Clients {
					updated.Clients = append(updated.Clients, nfsutil.Client{
						Host:    c.Host,
						Options: c.Options,
					})
				}
				result = append(result, updated)
			} else {
				result = append(result, e)
			}
		}
		return result
	})
}

func (s *NfsService) DeleteExport(req dto.NfsExportDelete) error {
	return s.safeWriteExport(func(list []nfsutil.Export) []nfsutil.Export {
		var result []nfsutil.Export
		for _, e := range list {
			if e.Path != req.Path {
				result = append(result, e)
			}
		}
		return result
	})
}

// ====== Connections ======

func (s *NfsService) GetConnections() (*dto.NfsConnections, error) {
	result := &dto.NfsConnections{}

	if out, err := exec.Command("exportfs", "-v").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				result.ActiveExports = append(result.ActiveExports, line)
			}
		}
	}

	if out, err := exec.Command("showmount", "-a", "--no-headers").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			info := dto.NfsConnectionInfo{Hostname: parts[0]}
			if len(parts) > 1 {
				info.DirPath = parts[1]
			}
			result.Clients = append(result.Clients, info)
		}
	}

	return result, nil
}

// ====== Helpers ======

func (s *NfsService) safeWriteExport(mutate func([]nfsutil.Export) []nfsutil.Export) error {
	exports, err := nfsutil.Parse(exportsPath)
	if err != nil && !os.IsNotExist(err) {
		return buserr.WithErr("ErrNfsReadExports", err)
	}

	backup := exportsPath + ".bak"
	if data, readErr := os.ReadFile(exportsPath); readErr == nil {
		_ = os.WriteFile(backup, data, 0644)
	}

	exports = mutate(exports)
	if err := nfsutil.Write(exportsPath, exports); err != nil {
		return err
	}

	out, applyErr := exec.Command("exportfs", "-ra").CombinedOutput()
	if applyErr != nil {
		if bak, readErr := os.ReadFile(backup); readErr == nil {
			_ = os.WriteFile(exportsPath, bak, 0644)
		}
		_ = exec.Command("exportfs", "-ra").Run()
		return buserr.WithDetail("ErrNfsApplyExports", string(out), applyErr)
	}
	return nil
}
