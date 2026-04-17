package service

import (
	"context"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/global"
)

type IAppImportProgressService interface {
	StartImportWithProgress(req dto.AppImportReq) error
	GetImportProgress(name string) (*model.AppImportTask, error)
	GetImportTasks() ([]model.AppImportTask, error)
}

type AppImportProgressService struct {
	taskRepo      repo.IAppImportTaskRepo
	importService IAppImportService
}

func NewIAppImportProgressService() IAppImportProgressService {
	return &AppImportProgressService{
		taskRepo:      repo.NewIAppImportTaskRepo(),
		importService: NewIAppImportService(),
	}
}

// StartImportWithProgress 启动带进度的导入任务
func (s *AppImportProgressService) StartImportWithProgress(req dto.AppImportReq) error {
	// 检查是否已有同名任务
	existing, _ := s.taskRepo.GetByName(req.Name)
	if existing.ID > 0 && (existing.Status == "pending" || existing.Status == "running") {
		return buserr.New("ErrImportTaskRunning")
	}

	// 创建导入任务记录
	now := time.Now()
	task := model.AppImportTask{
		Name:        req.Name,
		BackupPath:  req.BackupPath,
		AppKey:      req.AppKey,
		Version:     req.Version,
		Status:      "pending",
		Progress:    0,
		CurrentStep: "准备导入...",
		StartedAt:   &now,
	}

	if err := s.taskRepo.Create(context.Background(), &task); err != nil {
		return buserr.WithDetail("ErrCreateImportTask", err.Error(), err)
	}

	// 异步执行导入
	go s.executeImportWithProgress(&task, req)

	return nil
}

// executeImportWithProgress 执行带进度反馈的导入
func (s *AppImportProgressService) executeImportWithProgress(task *model.AppImportTask, req dto.AppImportReq) {
	// 更新任务状态为运行中
	task.Status = "running"
	task.Progress = 5
	task.CurrentStep = "开始导入..."
	s.taskRepo.Update(context.Background(), task)

	// 执行实际导入
	err := s.importService.ImportFromBackup(req)
	
	if err != nil {
		// 导入失败
		now := time.Now()
		task.Status = "failed"
		task.Progress = 0
		task.CurrentStep = "导入失败"
		task.Message = err.Error()
		task.CompletedAt = &now
		s.taskRepo.Update(context.Background(), task)
		global.LOG.Errorf("Import task %s failed: %v", task.Name, err)
		return
	}

	// 导入成功
	now := time.Now()
	task.Status = "success"
	task.Progress = 100
	task.CurrentStep = "导入完成"
	task.Message = "应用导入成功"
	task.CompletedAt = &now
	s.taskRepo.Update(context.Background(), task)
	
	global.LOG.Infof("Import task %s completed successfully", task.Name)
}

// validateBackupFile 验证备份文件
func (s *AppImportProgressService) validateBackupFile(backupPath string) error {
	// 这里可以添加更详细的文件验证逻辑
	return nil
}

// GetImportProgress 获取导入进度
func (s *AppImportProgressService) GetImportProgress(name string) (*model.AppImportTask, error) {
	task, err := s.taskRepo.GetByName(name)
	if err != nil {
		return nil, buserr.New("ErrImportTaskNotFound")
	}
	return &task, nil
}

// GetImportTasks 获取所有导入任务
func (s *AppImportProgressService) GetImportTasks() ([]model.AppImportTask, error) {
	tasks, err := s.taskRepo.GetList(repo.WithOrderBy("created_at", true))
	if err != nil {
		return nil, err
	}
	return tasks, nil
}