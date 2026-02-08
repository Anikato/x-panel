package repo

import (
	"xpanel/app/model"
)

// ILogRepo 日志仓库接口
type ILogRepo interface {
	CreateLoginLog(log *model.LoginLog) error
	PageLoginLog(page, pageSize int, opts ...DBOption) (int64, []model.LoginLog, error)
	CreateOperationLog(log *model.OperationLog) error
	PageOperationLog(page, pageSize int, opts ...DBOption) (int64, []model.OperationLog, error)
	CleanLoginLog() error
	CleanOperationLog() error
}

// NewILogRepo 创建日志仓库实例
func NewILogRepo() ILogRepo {
	return &LogRepo{}
}

type LogRepo struct{}

func (l *LogRepo) CreateLoginLog(log *model.LoginLog) error {
	return getDB().Create(log).Error
}

func (l *LogRepo) PageLoginLog(page, pageSize int, opts ...DBOption) (int64, []model.LoginLog, error) {
	var (
		total int64
		logs  []model.LoginLog
	)
	db := getDB().Model(&model.LoginLog{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error
	return total, logs, err
}

func (l *LogRepo) CreateOperationLog(log *model.OperationLog) error {
	return getDB().Create(log).Error
}

func (l *LogRepo) PageOperationLog(page, pageSize int, opts ...DBOption) (int64, []model.OperationLog, error) {
	var (
		total int64
		logs  []model.OperationLog
	)
	db := getDB().Model(&model.OperationLog{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error
	return total, logs, err
}

func (l *LogRepo) CleanLoginLog() error {
	return getDB().Where("1 = 1").Delete(&model.LoginLog{}).Error
}

func (l *LogRepo) CleanOperationLog() error {
	return getDB().Where("1 = 1").Delete(&model.OperationLog{}).Error
}
