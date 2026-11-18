package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/global"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	composeTaskPrefix   = "ct-"
	composeHistoryLimit = 150
)

func (s *ComposeService) StartCreateCompose(req dto.ComposeCreate) (*FileTaskStatus, error) {
	return s.startComposeTask(req.Name, "create", func(service *ComposeService) error {
		return service.CreateCompose(req)
	})
}

func (s *ComposeService) StartOperateCompose(req dto.ComposeOperate) (*FileTaskStatus, error) {
	project, err := s.resolveProject(req)
	if err != nil {
		return nil, err
	}
	return s.startComposeTask(project.Name, req.Operation, func(service *ComposeService) error {
		return service.OperateCompose(req)
	})
}

func (s *ComposeService) startComposeTask(name, operation string, run func(*ComposeService) error) (*FileTaskStatus, error) {
	task := model.ComposeTask{
		ID:        composeTaskPrefix + uuid.NewString(),
		Name:      "Compose · " + name + " · " + operation,
		Status:    "running",
		Phase:     "preparing",
		StartedAt: time.Now().UTC(),
	}
	db := global.DB
	if err := db.Create(&task).Error; err != nil {
		return nil, fmt.Errorf("record Compose task: %w", err)
	}
	go s.runComposeTask(db, task, name, run)
	return composeFileTask(task), nil
}

func (s *ComposeService) runComposeTask(db *gorm.DB, task model.ComposeTask, name string, run func(*ComposeService) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	output := &composeLog{limit: 8192}
	row := db.Model(&task).Where("status = ?", "running").Session(&gorm.Session{})
	checkpoint := func(progress int, phase string) {
		if err := row.Updates(map[string]any{
			"progress": progress, "phase": phase, "log": output.String(),
		}).Error; err != nil && global.LOG != nil {
			global.LOG.Warnf("save Compose task checkpoint %s: %v", task.ID, err)
		}
	}
	checkpoint(10, "preparing")
	taskService := *s
	taskService.run = func(name, path string, args ...string) (string, error) {
		phase := args[0]
		checkpoint(40, phase)
		var lastWrite time.Time
		output.notify = func() {
			now := time.Now()
			if now.Sub(lastWrite) >= time.Second {
				lastWrite = now
				checkpoint(40, phase)
			}
		}
		err := runComposeCommand(ctx, name, path, output, args...)
		output.notify = nil
		return output.String(), err
	}
	err := run(&taskService)

	checkpoint(85, "checking")
	projects, checkErr := s.list()
	if checkErr != nil {
		_, _ = fmt.Fprintln(output, "\n[project state unconfirmed]", checkErr)
		if err == nil {
			err = fmt.Errorf("operation completed, but project state could not be confirmed: %w", checkErr)
		}
	} else {
		state := "not listed"
		for _, item := range projects {
			if item.Name == name {
				state = item.Status
				break
			}
		}
		_, _ = fmt.Fprintln(output, "\n[project state]", state)
	}
	status := composeFileTask(task)
	status.Progress = 95
	notice := finishFileTask(status, err, FileTaskNotification{
		Source: "container", TargetURL: "/container", SuccessContent: "Compose 操作完成，已检查项目状态",
	})
	result := row.Updates(map[string]any{
		"status": status.Status, "message": status.Message, "ended_at": time.Unix(status.EndTime, 0).UTC(),
		"progress": status.Progress, "phase": "checked", "log": output.String(),
	})
	if result.Error != nil {
		if global.LOG != nil {
			global.LOG.Errorf("finish Compose task %s: %v", task.ID, result.Error)
		}
		return
	}
	if result.RowsAffected == 0 {
		return
	}
	CreateNotification(notice)
}

func composeFileTask(task model.ComposeTask) *FileTaskStatus {
	status := &FileTaskStatus{
		ID: task.ID, Type: "compose", Name: task.Name, Status: task.Status,
		StartTime: task.StartedAt.Unix(), Progress: task.Progress,
		CurrentFile: task.Phase, Message: task.Message, Log: task.Log,
	}
	if task.EndedAt != nil {
		status.EndTime = task.EndedAt.Unix()
	}
	return status
}

func findComposeTask(ctx context.Context, id string) (*FileTaskStatus, error) {
	var task model.ComposeTask
	err := global.DB.WithContext(ctx).First(&task, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return composeFileTask(task), nil
}

func appendComposeTasks(ctx context.Context, tasks []*FileTaskStatus) ([]*FileTaskStatus, error) {
	db := global.DB.WithContext(ctx)
	var rows []model.ComposeTask
	if err := db.Where("status = ? OR id IN (?)", "running", recentCompletedComposeTaskIDs(db)).
		Order("started_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		tasks = append(tasks, composeFileTask(row))
	}
	return tasks, nil
}

func recentCompletedComposeTaskIDs(db *gorm.DB) *gorm.DB {
	return db.Model(&model.ComposeTask{}).Select("id").
		Where("status <> ?", "running").Order("started_at DESC, id DESC").Limit(composeHistoryLimit)
}

// RecoverInterruptedComposeTasks runs before the HTTP server starts. Docker may
// have applied part or all of an interrupted operation, so the outcome is unknown.
func RecoverInterruptedComposeTasks() error {
	return global.DB.Model(&model.ComposeTask{}).Where("status = ?", "running").Updates(map[string]any{
		"status":   "interrupted",
		"message":  "面板已重启，操作结果未确认。请刷新 Compose 列表核对项目状态。",
		"ended_at": time.Now().UTC(),
	}).Error
}
