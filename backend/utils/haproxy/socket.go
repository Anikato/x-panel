package haproxy

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// DefaultSocketPath HAProxy admin socket 默认路径（apt 安装的 haproxy 包对应路径）
const DefaultSocketPath = "/run/haproxy/admin.sock"

// Socket 封装与 HAProxy admin.sock 的通信
type Socket struct {
	Path    string
	Timeout time.Duration
}

// NewSocket 创建 socket 客户端
func NewSocket(path string) *Socket {
	if path == "" {
		path = DefaultSocketPath
	}
	return &Socket{Path: path, Timeout: 5 * time.Second}
}

// Exec 发送一条命令，返回原始文本输出
func (s *Socket) Exec(cmd string) (string, error) {
	conn, err := net.DialTimeout("unix", s.Path, 2*time.Second)
	if err != nil {
		return "", fmt.Errorf("dial %s: %w", s.Path, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(s.Timeout))

	if !strings.HasSuffix(cmd, "\n") {
		cmd += "\n"
	}
	if _, err := conn.Write([]byte(cmd)); err != nil {
		return "", err
	}
	data, err := io.ReadAll(conn)
	if err != nil && err != io.EOF {
		return "", err
	}
	return string(data), nil
}

// Ping 测试 socket 是否可达
func (s *Socket) Ping() bool {
	out, err := s.Exec("show info")
	if err != nil {
		return false
	}
	return strings.Contains(out, "Name:") || strings.Contains(out, "Version:")
}

// ShowStat 返回 "show stat" 的原始 CSV 文本
func (s *Socket) ShowStat() (string, error) {
	return s.Exec("show stat")
}

// ShowInfo 返回 "show info" 的原始文本
func (s *Socket) ShowInfo() (string, error) {
	return s.Exec("show info")
}

// DisableServer 禁用某个 backend/server
func (s *Socket) DisableServer(backend, server string) error {
	out, err := s.Exec(fmt.Sprintf("disable server %s/%s", backend, server))
	if err != nil {
		return err
	}
	if strings.Contains(out, "No such") || strings.Contains(out, "denied") || strings.Contains(out, "error") {
		return fmt.Errorf("disable failed: %s", strings.TrimSpace(out))
	}
	return nil
}

// EnableServer 启用某个 backend/server
func (s *Socket) EnableServer(backend, server string) error {
	out, err := s.Exec(fmt.Sprintf("enable server %s/%s", backend, server))
	if err != nil {
		return err
	}
	if strings.Contains(out, "No such") || strings.Contains(out, "denied") || strings.Contains(out, "error") {
		return fmt.Errorf("enable failed: %s", strings.TrimSpace(out))
	}
	return nil
}

// SetWeight 调整 server 权重（0-256）
func (s *Socket) SetWeight(backend, server string, weight int) error {
	out, err := s.Exec(fmt.Sprintf("set weight %s/%s %d", backend, server, weight))
	if err != nil {
		return err
	}
	out = strings.TrimSpace(out)
	if out != "" && !strings.Contains(out, "done") {
		if strings.Contains(out, "No such") || strings.Contains(out, "denied") || strings.Contains(out, "error") {
			return fmt.Errorf("set weight failed: %s", out)
		}
	}
	return nil
}

// ShutdownSessions 断开某个 server 的全部会话
func (s *Socket) ShutdownSessions(backend, server string) error {
	_, err := s.Exec(fmt.Sprintf("shutdown sessions server %s/%s", backend, server))
	return err
}

// ClearCounters 清零所有计数器
func (s *Socket) ClearCounters() error {
	_, err := s.Exec("clear counters all")
	return err
}
