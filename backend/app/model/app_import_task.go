package model

import "time"

// AppImportTask 应用导入任务
type AppImportTask struct {
	BaseModel
	Name        string    `gorm:"not null;uniqueIndex" json:"name"`        // 任务名称（与安装名称相同）
	BackupPath  string    `gorm:"not null" json:"backupPath"`              // 备份文件路径
	AppKey      string    `json:"appKey"`                                   // 应用标识
	Version     string    `json:"version"`                                  // 版本
	Status      string    `gorm:"not null;default:'pending'" json:"status"` // 状态：pending/running/success/failed
	Progress    int       `gorm:"default:0" json:"progress"`                // 进度百分比 0-100
	CurrentStep string    `json:"currentStep"`                              // 当前步骤描述
	Message     string    `json:"message"`                                  // 错误信息或成功信息
	StartedAt   *time.Time `json:"startedAt"`                               // 开始时间
	CompletedAt *time.Time `json:"completedAt"`                             // 完成时间
}

// TableName 表名
func (AppImportTask) TableName() string {
	return "app_import_tasks"
}