package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"
)

// FileTaskStatus 文件操作任务状态
type FileTaskStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`              // 任务描述
	Type      string `json:"type"`              // move, compress, decompress
	Status    string `json:"status"`            // running, success, failed
	Message   string `json:"message,omitempty"` // 错误信息
	Log       string `json:"log,omitempty"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime,omitempty"`
	// 进度信息
	Progress    int    `json:"progress"` // 0-100
	BytesDone   int64  `json:"bytesDone"`
	BytesTotal  int64  `json:"bytesTotal"`
	Speed       int64  `json:"speed"`       // bytes/s 滑动平均
	CurrentFile string `json:"currentFile"` // 正在处理的文件名
}

// ProgressTracker 内部速度计算器（导出，供 API 层传递）
type ProgressTracker struct {
	task      *FileTaskStatus
	mu        sync.Mutex
	lastTime  time.Time
	lastBytes int64
}

type FileTaskNotification struct {
	Source             string
	TargetURL          string
	Event              string
	SuccessTitle       string
	SuccessContent     string
	SuccessContentFunc func() string
	FailedTitle        string
}

func newProgressTracker(task *FileTaskStatus) *ProgressTracker {
	return &ProgressTracker{task: task, lastTime: time.Now()}
}

// SetTotal 更新任务总字节数，适用于下载开始后才能拿到 Content-Length 的场景。
func (pt *ProgressTracker) SetTotal(total int64) {
	if total > 0 {
		fileTasksMu.Lock()
		pt.task.BytesTotal = total
		fileTasksMu.Unlock()
	}
}

// SetCurrentFile 更新当前处理文件名。
func (pt *ProgressTracker) SetCurrentFile(name string) {
	fileTasksMu.Lock()
	defer fileTasksMu.Unlock()
	pt.task.CurrentFile = name
}

// AddBytes 原子累加已完成字节，并按500ms采样更新速度
func (pt *ProgressTracker) AddBytes(n int64) {
	fileTasksMu.Lock()
	defer fileTasksMu.Unlock()
	pt.task.BytesDone += n
	done, total := pt.task.BytesDone, pt.task.BytesTotal
	if total > 0 {
		pct := int(done * 100 / total)
		if pct > 99 {
			pct = 99
		}
		pt.task.Progress = pct
	}

	pt.mu.Lock()
	defer pt.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(pt.lastTime).Seconds()
	if elapsed >= 0.5 {
		bytesDelta := done - pt.lastBytes
		pt.task.Speed = int64(float64(bytesDelta) / elapsed)
		pt.lastBytes = done
		pt.lastTime = now
	}
}

var (
	fileTasksMu     sync.RWMutex
	fileTasks       = make(map[string]*FileTaskStatus)
	fileTaskCancels = make(map[string]context.CancelFunc)
	taskSeq         int64
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
func completeFileTask(task *FileTaskStatus, err error, notify FileTaskNotification) {
	fileTasksMu.Lock()
	delete(fileTaskCancels, task.ID)
	notification := finishFileTask(task, err, notify)
	fileTasksMu.Unlock()
	CreateNotification(notification)
}

func finishFileTask(task *FileTaskStatus, err error, notify FileTaskNotification) dto.NotificationCreate {
	task.EndTime = time.Now().Unix()
	notificationType := "success"
	notificationTitle := notify.SuccessTitle
	notificationContent := notify.SuccessContent
	if notificationTitle == "" {
		notificationTitle = task.Name + "完成"
	}
	if notificationContent == "" {
		notificationContent = "后台任务已完成"
	}
	if errors.Is(err, context.Canceled) {
		task.Status = "cancelled"
		task.Message = "任务已取消"
		notificationType = "warning"
		notificationTitle = task.Name + "已取消"
		notificationContent = "后台任务已取消"
	} else if err != nil {
		task.Status = "failed"
		task.Message = err.Error()
		notificationType = "error"
		notificationTitle = notify.FailedTitle
		if notificationTitle == "" {
			notificationTitle = task.Name + "失败"
		}
		notificationContent = err.Error()
	} else {
		task.Status = "success"
		task.Progress = 100
		if notify.SuccessContentFunc != nil {
			if content := notify.SuccessContentFunc(); content != "" {
				notificationContent = content
			}
		}
	}
	if notify.Source == "" {
		notify.Source = "file"
	}
	if notify.TargetURL == "" {
		notify.TargetURL = "/host/files"
	}
	event := notify.Event
	if event == "" {
		switch task.Status {
		case "success":
			event = notify.Source + ".task.success"
		case "cancelled":
			event = notify.Source + ".task.cancelled"
		default:
			event = notify.Source + ".task.failed"
		}
	}
	return dto.NotificationCreate{
		Type:      notificationType,
		Event:     event,
		Title:     notificationTitle,
		Content:   notificationContent,
		Source:    notify.Source,
		TargetURL: notify.TargetURL,
	}
}

// GetFileTask 获取单个任务状态
func GetFileTask(ctx context.Context, id string) (*FileTaskStatus, error) {
	if strings.HasPrefix(id, composeTaskPrefix) {
		return findComposeTask(ctx, id)
	}
	return getMemoryFileTask(id), nil
}

func getMemoryFileTask(id string) *FileTaskStatus {
	fileTasksMu.RLock()
	defer fileTasksMu.RUnlock()
	if t, ok := fileTasks[id]; ok {
		snapshot := *t
		return &snapshot
	}
	return nil
}

// RegisterFileTaskCancel 为运行中的任务登记取消函数。
func RegisterFileTaskCancel(id string, cancel context.CancelFunc) {
	fileTasksMu.Lock()
	defer fileTasksMu.Unlock()
	task, ok := fileTasks[id]
	if !ok || task.Status != "running" {
		cancel()
		return
	}
	fileTaskCancels[id] = cancel
}

// CancelFileTask 取消支持取消的运行中任务。
func CancelFileTask(id string) error {
	if strings.HasPrefix(id, composeTaskPrefix) {
		task, err := findComposeTask(context.Background(), id)
		if err != nil {
			return err
		}
		if task == nil {
			return errors.New("task not found")
		}
		if task.Status != "running" {
			return errors.New("task is not running")
		}
		return errors.New("task cannot be cancelled")
	}
	fileTasksMu.Lock()
	task, ok := fileTasks[id]
	if !ok {
		fileTasksMu.Unlock()
		return fmt.Errorf("task not found")
	}
	if task.Status != "running" {
		fileTasksMu.Unlock()
		return fmt.Errorf("task is not running")
	}
	cancel, ok := fileTaskCancels[id]
	if !ok {
		fileTasksMu.Unlock()
		return fmt.Errorf("task cannot be cancelled")
	}
	delete(fileTaskCancels, id)
	fileTasksMu.Unlock()

	cancel()
	return nil
}

// ListFileTasks 获取所有任务列表（按开始时间倒序）
func ListFileTasks(ctx context.Context) ([]*FileTaskStatus, error) {
	fileTasksMu.RLock()
	result := make([]*FileTaskStatus, 0, len(fileTasks))
	cutoff := time.Now().Unix() - 3600 // 只返回 1 小时内的任务
	for _, t := range fileTasks {
		if t.Status == "running" || t.StartTime > cutoff {
			snapshot := *t
			result = append(result, &snapshot)
		}
	}
	fileTasksMu.RUnlock()

	result, err := appendComposeTasks(ctx, result)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].StartTime > result[j].StartTime
	})
	return result, nil
}

// StartFileTask 启动异步文件任务（不带进度）
func StartFileTask(taskType, name string, fn func() error) *FileTaskStatus {
	return StartFileTaskWithNotification(taskType, name, FileTaskNotification{
		SuccessContent: "文件后台任务已完成",
	}, fn)
}

// StartFileTaskWithNotification 启动异步任务并按指定来源写入通知
func StartFileTaskWithNotification(taskType, name string, notify FileTaskNotification, fn func() error) *FileTaskStatus {
	task := newFileTask(taskType, name)
	go func() {
		err := fn()
		completeFileTask(task, err, notify)
	}()
	return task
}

// StartFileTaskWithProgress 启动带进度追踪的异步文件任务
func StartFileTaskWithProgress(taskType, name string, totalBytes int64, fn func(*ProgressTracker) error) *FileTaskStatus {
	return startProgressTask(taskType, name, totalBytes, FileTaskNotification{
		SuccessContent: "文件后台任务已完成",
	}, fn)
}

func startProgressTask(taskType, name string, totalBytes int64, notify FileTaskNotification, fn func(*ProgressTracker) error) *FileTaskStatus {
	task := newFileTask(taskType, name)
	tracker := newProgressTracker(task)
	tracker.SetTotal(totalBytes)
	initial := getMemoryFileTask(task.ID)
	go func() {
		completeFileTask(task, fn(tracker), notify)
	}()
	return initial
}

// CalcDirBytes 递归统计目录总字节数（导出供 API 层使用）
func CalcDirBytes(root string) int64 {
	return calcDirBytes(root)
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
