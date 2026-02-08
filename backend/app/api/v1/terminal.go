package v1

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
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

// WsTerminal WebSocket 终端（支持本地 PTY 和远程 SSH）
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

	if hostID > 0 {
		a.handleSSHTerminal(conn, uint(hostID))
	} else {
		a.handleLocalTerminal(conn)
	}
}

// handleLocalTerminal 本地 PTY 终端
func (a *TerminalAPI) handleLocalTerminal(conn *websocket.Conn) {
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
	cmd := exec.Command(shell)
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

	// PTY → WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				once.Do(func() { close(done) })
				return
			}
			if n > 0 {
				if err := sc.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
					once.Do(func() { close(done) })
					return
				}
			}
		}
	}()

	// WebSocket → PTY
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				once.Do(func() { close(done) })
				return
			}
			if len(msg) == 0 {
				continue
			}
			if msg[0] == 1 {
				var resize resizeMsg
				if err := json.Unmarshal(msg[1:], &resize); err == nil {
					pty.Setsize(ptmx, &pty.Winsize{Rows: resize.Rows, Cols: resize.Cols})
					continue
				}
			}
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
			_, msg, err := conn.ReadMessage()
			if err != nil {
				once.Do(func() { close(done) })
				return
			}
			if len(msg) == 0 {
				continue
			}
			if msg[0] == 1 {
				var resize resizeMsg
				if err := json.Unmarshal(msg[1:], &resize); err == nil {
					session.WindowChange(int(resize.Rows), int(resize.Cols))
					continue
				}
			}
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
	if err := w.sc.WriteMessage(websocket.TextMessage, p); err != nil {
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
