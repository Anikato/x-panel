package dto

import "time"

// ---- Nginx 状态 ----

type NginxStatus struct {
	IsInstalled      bool      `json:"isInstalled"`
	IsRunning        bool      `json:"isRunning"`
	Version          string    `json:"version"`
	PID              int       `json:"pid"`
	InstallDir       string    `json:"installDir"`
	StartedAt        time.Time `json:"startedAt"`
	ConfigOK         bool      `json:"configOK"`
	AutoStart        bool      `json:"autoStart"`
	SystemMode       bool      `json:"systemMode"`
	HasBothInstalled bool      `json:"hasBothInstalled"`
}

type NginxAutoStartReq struct {
	Enable bool `json:"enable"`
}

// ---- Nginx 安装 ----

type NginxInstallReq struct {
	Version string `json:"version" binding:"required"` // 目标版本号 (如 "1.26.2")
}

type NginxInstallProgress struct {
	Phase   string `json:"phase"`   // 当前阶段: download/verify/install/done/error
	Message string `json:"message"` // 阶段描述
	Percent int    `json:"percent"` // 总体进度百分比 0-100
}

// ---- Nginx 可用版本 ----

type NginxVersionInfo struct {
	Version     string `json:"version"`     // 版本号 (如 "1.26.2")
	Tag         string `json:"tag"`         // Git 标签 (如 "v1.26.2")
	PublishedAt string `json:"publishedAt"` // 发布时间
}

// ---- Nginx 操作 ----

type NginxOperateReq struct {
	Operation string `json:"operation" binding:"required,oneof=start stop reload reopen quit"` // 操作类型
}

type NginxConfigTestResult struct {
	Success bool   `json:"success"` // 测试是否通过
	Output  string `json:"output"`  // nginx -t 原始输出
}
