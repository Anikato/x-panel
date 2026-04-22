package service

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// FileTaskStatus 文件操作任务状态
type FileTaskStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`              // 任务描述（如 "复制 3 个文件到 /data"）
	Type      string `json:"type"`              // move, compress, decompress
	Status    string `json:"status"`            // running, success, failed
	Message   string `json:"message,omitempty"` // 错误信息
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime,omitempty"`
}

var (
	fileTasksMu sync.RWMutex
	fileTasks   = make(map[string]*FileTaskStatus)
	taskSeq     int64
)

// newFileTask 创建异步文件任务
func newFileTask(taskType, name string) *FileTaskStatus {
	fileTasksMu.Lock()
	defer fileTasksMu.Unlock()
	taskSeq++
	task := &FileTaskStatus{
		ID:        fmt.Sprintf("ft-%d-%d", time.Now().UnixMilli(), taskSeq),
		Name:      name,
		Type:      taskType,
		Status:    "running",
		StartTime: time.Now().Unix(),
	}
	fileTasks[task.ID] = task

	// 限制最多保留 150 个任务，防止内存泄漏
	if len(fileTasks) > 150 {
		cleanOldTasks()
	}

	return task
}

// completeFileTask 标记任务完成
func completeFileTask(task *FileTaskStatus, err error) {
	fileTasksMu.Lock()
	defer fileTasksMu.Unlock()
	task.EndTime = time.Now().Unix()
	if err != nil {
		task.Status = "failed"
		task.Message = err.Error()
	} else {
		task.Status = "success"
	}
}

// GetFileTask 获取单个任务状态
func GetFileTask(id string) *FileTaskStatus {
	fileTasksMu.RLock()
	defer fileTasksMu.RUnlock()
	if t, ok := fileTasks[id]; ok {
		return t
	}
	return nil
}

// ListFileTasks 获取所有任务列表（按开始时间倒序）
func ListFileTasks() []*FileTaskStatus {
	fileTasksMu.RLock()
	defer fileTasksMu.RUnlock()
	result := make([]*FileTaskStatus, 0, len(fileTasks))
	cutoff := time.Now().Unix() - 3600 // 只返回 1 小时内的任务
	for _, t := range fileTasks {
		if t.Status == "running" || t.StartTime > cutoff {
			result = append(result, t)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartTime > result[j].StartTime
	})
	return result
}

// StartFileTask 启动异步文件任务
func StartFileTask(taskType, name string, fn func() error) *FileTaskStatus {
	task := newFileTask(taskType, name)
	go func() {
		err := fn()
		completeFileTask(task, err)
	}()
	return task
}

// cleanOldTasks 清理已完成超过 10 分钟的任务
func cleanOldTasks() {
	cutoff := time.Now().Unix() - 600
	for id, task := range fileTasks {
		if task.Status != "running" && task.EndTime < cutoff {
			delete(fileTasks, id)
		}
	}
}
