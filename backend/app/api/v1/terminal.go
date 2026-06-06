package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"xpanel/app/service"
	"xpanel/global"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

type TerminalAPI struct{}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type resizeMsg struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

// WsTerminal WebSocket 终端（支持本地 PTY、远程 SSH 和容器终端）
func (a *TerminalAPI) WsTerminal(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		global.LOG.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// 判断是本地终端还是远程 SSH
	hostIDStr := c.Query("id")
	hostID, _ := strconv.ParseUint(hostIDStr, 10, 64)
	containerID := c.Query("containerID")
	cwd := c.Query("cwd")

	if containerID != "" {
		a.handleContainerTerminal(conn, containerID, c.Query("command"), c.Query("user"))
	} else if hostID > 0 {
		a.handleSSHTerminal(conn, uint(hostID))
	} else {
		a.handleLocalTerminal(conn, cwd)
	}
}

// handleLocalTerminal 本地 PTY 终端
func (a *TerminalAPI) handleLocalTerminal(conn *websocket.Conn, cwd string) {
	sc := &safeConn{conn: conn}

	// 检查 PTY 设备是否可用
	if _, err := os.Stat("/dev/ptmx"); err != nil {
		global.LOG.Errorf("PTY device not available: %v", err)
		sc.WriteMessage(websocket.TextMessage, []byte(
			"\r\n\x1b[31m终端启动失败：/dev/ptmx 设备不可用\x1b[0m\r\n"+
				"\x1b[33m可能的原因：\r\n"+
				"  1. 当前进程运行在受限环境（沙箱/容器）中\r\n"+
				"  2. devpts 文件系统未挂载\r\n"+
				"解决方法：确保后端进程以完整权限运行\x1b[0m\r\n"))
		return
	}

	shell := getShell()
	cmd := exec.Command(shell, "-l")
	cmd.Dir = resolveLocalTerminalDir(cwd)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "LANG=en_US.UTF-8")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		global.LOG.Errorf("PTY start failed: %v", err)
		sc.WriteMessage(websocket.TextMessage, []byte(
			"\r\n\x1b[31m终端启动失败："+err.Error()+"\x1b[0m\r\n"+
				"\x1b[33m请检查 /dev/ptmx 设备权限，或以 root 权限运行后端\x1b[0m\r\n"))
		return
	}
	defer func() {
		ptmx.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	global.LOG.Info("Local terminal session started")
	var once sync.Once
	done := make(chan struct{})

	// PTY → WebSocket (BinaryMessage to avoid UTF-8 validation issues with raw terminal data)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				once.Do(func() { close(done) })
				return
			}
			if n > 0 {
				if err := sc.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					once.Do(func() { close(done) })
					return
				}
			}
		}
	}()

	// WebSocket → PTY
	go func() {
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				once.Do(func() { close(done) })
				return
			}
			if len(msg) == 0 {
				continue
			}
			// Binary frames with prefix byte 1 are resize commands
			if msgType == websocket.BinaryMessage && msg[0] == 1 {
				var resize resizeMsg
				if err := json.Unmarshal(msg[1:], &resize); err == nil {
					pty.Setsize(ptmx, &pty.Winsize{Rows: resize.Rows, Cols: resize.Cols})
				}
				continue
			}
			// Text frames are terminal input
			if _, err := ptmx.Write(msg); err != nil {
				once.Do(func() { close(done) })
				return
			}
		}
	}()

	// 心跳
	go wsHeartbeat(sc, done, &once)
	<-done
	global.LOG.Info("Local terminal session closed")
}

func resolveLocalTerminalDir(cwd string) string {
	fallback := localTerminalHomeDir()
	requested := strings.TrimSpace(cwd)
	if requested == "" {
		return fallback
	}

	cleaned := filepath.Clean(requested)
	if !filepath.IsAbs(cleaned) {
		return fallback
	}

	info, err := os.Stat(cleaned)
	if err != nil || !info.IsDir() {
		return fallback
	}
	return cleaned
}

func localTerminalHomeDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		return "/root"
	}
	return home
}

// handleSSHTerminal 远程 SSH 终端
func (a *TerminalAPI) handleSSHTerminal(conn *websocket.Conn, hostID uint) {
	sc := &safeConn{conn: conn}

	svc := service.NewIHostService()
	sshClient, err := svc.ConnSSH(hostID)
	if err != nil {
		global.LOG.Errorf("SSH connect failed: %v", err)
		sc.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31mSSH 连接失败: "+err.Error()+"\x1b[0m\r\n"))
		return
	}
	defer sshClient.Close()

	session, err := sshClient.NewSession()
	if err != nil {
		global.LOG.Errorf("SSH session failed: %v", err)
		sc.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31m创建 SSH 会话失败\x1b[0m\r\n"))
		return
	}
	defer session.Close()

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return
	}

	// stdout 和 stderr 共享同一个 safeConn，避免并发写
	session.Stdout = &wsWriter{sc: sc}
	session.Stderr = &wsWriter{sc: sc}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", 30, 120, modes); err != nil {
		sc.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31mPTY 分配失败\x1b[0m\r\n"))
		return
	}
	if err := session.Shell(); err != nil {
		sc.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31mShell 启动失败\x1b[0m\r\n"))
		return
	}

	global.LOG.Infof("SSH terminal session started for host %d", hostID)
	var once sync.Once
	done := make(chan struct{})

	// WebSocket → SSH stdin
	go func() {
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				once.Do(func() { close(done) })
				return
			}
			if len(msg) == 0 {
				continue
			}
			// Binary frames with prefix byte 1 are resize commands
			if msgType == websocket.BinaryMessage && msg[0] == 1 {
				var resize resizeMsg
				if err := json.Unmarshal(msg[1:], &resize); err == nil {
					session.WindowChange(int(resize.Rows), int(resize.Cols))
				}
				continue
			}
			// Text frames are terminal input
			if _, err := stdinPipe.Write(msg); err != nil {
				once.Do(func() { close(done) })
				return
			}
		}
	}()

	// 等待 session 结束
	go func() {
		session.Wait()
		once.Do(func() { close(done) })
	}()

	// 心跳
	go wsHeartbeat(sc, done, &once)
	<-done
	global.LOG.Infof("SSH terminal session closed for host %d", hostID)
}

// handleContainerTerminal 容器 PTY 终端
func (a *TerminalAPI) handleContainerTerminal(conn *websocket.Conn, containerID, command, user string) {
	sc := &safeConn{conn: conn}

	args, err := buildDockerExecArgs(containerID, command, user)
	if err != nil {
		sc.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31m容器终端参数错误: "+err.Error()+"\x1b[0m\r\n"))
		return
	}
	if _, err := exec.LookPath("docker"); err != nil {
		sc.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31m未找到 docker 命令，请确认 Docker 已安装\x1b[0m\r\n"))
		return
	}

	cmd := exec.Command("docker", args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "LANG=en_US.UTF-8")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		global.LOG.Errorf("Container terminal start failed: %v", err)
		sc.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31m容器终端启动失败: "+err.Error()+"\x1b[0m\r\n"))
		return
	}
	defer func() {
		ptmx.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	global.LOG.Infof("Container terminal session started: %s", containerID)
	var once sync.Once
	done := make(chan struct{})

	go pipePTYToWS(ptmx, sc, done, &once)
	go pipeWSToPTY(conn, ptmx, done, &once)
	go wsHeartbeat(sc, done, &once)

	<-done
	global.LOG.Infof("Container terminal session closed: %s", containerID)
}

func pipePTYToWS(ptmx *os.File, sc *safeConn, done chan struct{}, once *sync.Once) {
	buf := make([]byte, 4096)
	for {
		n, err := ptmx.Read(buf)
		if err != nil {
			once.Do(func() { close(done) })
			return
		}
		if n > 0 {
			if err := sc.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				once.Do(func() { close(done) })
				return
			}
		}
	}
}

func pipeWSToPTY(conn *websocket.Conn, ptmx *os.File, done chan struct{}, once *sync.Once) {
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			once.Do(func() { close(done) })
			return
		}
		if len(msg) == 0 {
			continue
		}
		if msgType == websocket.BinaryMessage && msg[0] == 1 {
			var resize resizeMsg
			if err := json.Unmarshal(msg[1:], &resize); err == nil {
				pty.Setsize(ptmx, &pty.Winsize{Rows: resize.Rows, Cols: resize.Cols})
			}
			continue
		}
		if _, err := ptmx.Write(msg); err != nil {
			once.Do(func() { close(done) })
			return
		}
	}
}

var (
	containerIDPattern      = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:/-]{0,127}$`)
	containerUserPattern    = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.:-]{0,127}$`)
	containerCommandPattern = regexp.MustCompile(`^/?[a-zA-Z0-9_./-]{1,128}$`)
)

func buildDockerExecArgs(containerID, command, user string) ([]string, error) {
	if !containerIDPattern.MatchString(containerID) {
		return nil, fmt.Errorf("containerID 非法")
	}
	if command == "" {
		command = "/bin/sh"
	}
	if !containerCommandPattern.MatchString(command) {
		return nil, fmt.Errorf("command 非法")
	}
	args := []string{"exec", "-it"}
	if user != "" {
		if !containerUserPattern.MatchString(user) {
			return nil, fmt.Errorf("user 非法")
		}
		args = append(args, "-u", user)
	}
	args = append(args, containerID, command)
	return args, nil
}

// safeConn 封装 WebSocket 连接，保证并发写安全
type safeConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (sc *safeConn) WriteMessage(messageType int, data []byte) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.WriteMessage(messageType, data)
}

// wsWriter 将 SSH/PTY 输出写入 WebSocket（通过 safeConn 保证并发安全）
type wsWriter struct {
	sc *safeConn
}

func (w *wsWriter) Write(p []byte) (n int, err error) {
	if err := w.sc.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, io.EOF
	}
	return len(p), nil
}

func wsHeartbeat(sc *safeConn, done chan struct{}, once *sync.Once) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := sc.WriteMessage(websocket.PingMessage, nil); err != nil {
				once.Do(func() { close(done) })
				return
			}
		}
	}
}

func getShell() string {
	for _, sh := range []string{"/bin/zsh", "/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(sh); err == nil {
			return sh
		}
	}
	return "/bin/sh"
}
