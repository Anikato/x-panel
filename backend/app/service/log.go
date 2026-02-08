package service

import (
	"xpanel/app/dto"
	"xpanel/app/model"
)

// ILogService 日志服务接口
type ILogService interface {
	PageLoginLog(req dto.SearchWithPage) (int64, []model.LoginLog, error)
	PageOperationLog(req dto.SearchWithPage) (int64, []model.OperationLog, error)
	CreateLoginLog(log model.LoginLog) error
	CleanLoginLog() error
	CleanOperationLog() error
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
