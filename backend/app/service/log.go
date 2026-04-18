package service

import (
	"os/exec"
	"path/filepath"
	"strconv"

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
	GetSystemLog(lines int) (string, error)
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

func (l *LogService) GetSystemLog(lines int) (string, error) {
	logPath := global.CONF.Log.Path
	if logPath == "" {
		return "", nil
	}
	logFile := filepath.Join(logPath, "xpanel.log")

	cmd := exec.Command("tail", "-n", strconv.Itoa(lines), logFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return string(out), nil
}
