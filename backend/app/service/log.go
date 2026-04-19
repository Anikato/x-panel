package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"fmt"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/global"
)

// ILogService 日志服务接口
type ILogService interface {
	PageLoginLog(req dto.SearchWithPage) (int64, []model.LoginLog, error)
	PageOperationLog(req dto.SearchWithPage) (int64, []model.OperationLog, error)
	CreateLoginLog(log model.LoginLog) error
	CleanLoginLog() error
	CleanOperationLog() error
	GetSystemLog(lines int, level string, keyword string) (string, error)
	CleanSystemLog() error
}

// NewILogService 创建日志服务实例
func NewILogService() ILogService {
	return &LogService{}
}

type LogService struct{}

func (l *LogService) PageLoginLog(req dto.SearchWithPage) (int64, []model.LoginLog, error) {
	return logRepo.PageLoginLog(req.Page, req.PageSize)
}

func (l *LogService) PageOperationLog(req dto.SearchWithPage) (int64, []model.OperationLog, error) {
	return logRepo.PageOperationLog(req.Page, req.PageSize)
}

func (l *LogService) CreateLoginLog(log model.LoginLog) error {
	return logRepo.CreateLoginLog(&log)
}

func (l *LogService) CleanLoginLog() error {
	return logRepo.CleanLoginLog()
}

func (l *LogService) CleanOperationLog() error {
	return logRepo.CleanOperationLog()
}

func (l *LogService) GetSystemLog(lines int, level string, keyword string) (string, error) {
	logPath := global.CONF.Log.Path
	if logPath == "" {
		return "", nil
	}
	logFile := filepath.Join(logPath, "xpanel.log")

	// 构造 bash 命令管道
	cmdStr := fmt.Sprintf("cat %s", logFile)
	if level != "" {
		var prefix string
		switch level {
		case "INFO":
			prefix = "^INFO"
		case "WARN":
			prefix = "^WARN"
		case "ERROR":
			prefix = "^(ERRO|FATA|PANI)"
		}
		if prefix != "" {
			cmdStr += fmt.Sprintf(" | grep -a -E '%s'", prefix)
		}
	}
	if keyword != "" {
		// 简单的防注入和单引号转义
		safeKeyword := strings.ReplaceAll(keyword, "'", "'\\''")
		cmdStr += fmt.Sprintf(" | grep -a -F '%s'", safeKeyword)
	}
	cmdStr += fmt.Sprintf(" | tail -n %d", lines)

	cmd := exec.Command("bash", "-c", cmdStr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// 如果 grep 没找到内容，返回退出码 1 是正常的
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return string(out), nil
		}
		return string(out), err
	}
	return string(out), nil
}

func (l *LogService) CleanSystemLog() error {
	logPath := global.CONF.Log.Path
	if logPath == "" {
		return nil
	}
	logFile := filepath.Join(logPath, "xpanel.log")
	return os.Truncate(logFile, 0)
}
