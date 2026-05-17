package log

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"
	"xpanel/app/service"
	"xpanel/global"

	"github.com/sirupsen/logrus"
)

// Init 初始化日志模块
func Init() {
	logger := logrus.New()

	// 解析日志级别
	level, err := logrus.ParseLevel(global.CONF.Log.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// 设置日志格式
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 确保日志目录存在
	logPath := global.CONF.Log.Path
	if logPath != "" {
		if err := os.MkdirAll(logPath, 0755); err != nil {
			logger.Warnf("Failed to create log directory %s: %v", logPath, err)
		} else {
			logFile := filepath.Join(logPath, "xpanel.log")
			f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				logger.Warnf("Failed to open log file %s: %v", logFile, err)
			} else {
				// 同时输出到文件和控制台
				multiWriter := io.MultiWriter(os.Stdout, f)
				logger.SetOutput(multiWriter)
			}
		}
	}

	global.LOG = logger
	global.LOG.AddHook(newNotificationHook())
	global.LOG.Info("Logger initialized")
}

type notificationHook struct {
	mu       sync.Mutex
	lastSeen map[string]time.Time
}

func newNotificationHook() *notificationHook {
	return &notificationHook{lastSeen: map[string]time.Time{}}
}

func (h *notificationHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel}
}

func (h *notificationHook) Fire(entry *logrus.Entry) error {
	if global.DB == nil {
		return nil
	}
	msg := strings.TrimSpace(entry.Message)
	if msg == "" {
		msg = "面板记录了一条错误日志"
	}
	if len(msg) > 500 {
		msg = msg[:500] + "\n...(truncated)"
	}
	key := entry.Level.String() + ":" + msg

	h.mu.Lock()
	if last, ok := h.lastSeen[key]; ok && time.Since(last) < time.Minute {
		h.mu.Unlock()
		return nil
	}
	h.lastSeen[key] = time.Now()
	h.mu.Unlock()

	go service.CreateNotification(dto.NotificationCreate{
		Type:      "error",
		Event:     "system.log.error",
		Title:     "面板错误日志",
		Content:   msg,
		Source:    "system",
		TargetURL: "/logs/system",
	})
	return nil
}
