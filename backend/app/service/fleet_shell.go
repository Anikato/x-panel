package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"xpanel/global"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type fleetShellResize struct {
	Rows uint16 `json:"rows"`
	Cols uint16 `json:"cols"`
}

func (s *FleetReporterService) openFleetShell(endpoint, token, sessionID string) {
	wsURL, err := fleetShellURL(endpoint, sessionID)
	if err != nil {
		global.LOG.Warnf("fleet shell build url failed: %v", err)
		return
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+token)
	header.Set("User-Agent", "X-Panel Fleet Shell")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		global.LOG.Warnf("fleet shell connect failed: %v", err)
		return
	}
	defer conn.Close()

	if err := runFleetShellPTY(conn); err != nil {
		global.LOG.Warnf("fleet shell closed: %v", err)
	}
}

func fleetShellURL(endpoint, sessionID string) (string, error) {
	parsed, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported endpoint scheme: %s", parsed.Scheme)
	}
	parsed.Path = "/api/v1/fleet/shell/connect"
	query := parsed.Query()
	query.Set("sessionId", sessionID)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func runFleetShellPTY(conn *websocket.Conn) error {
	shell := fleetShellPath()
	cmd := exec.Command(shell, "-l")
	home := os.Getenv("HOME")
	if home == "" {
		home = "/root"
	}
	cmd.Dir = home
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "LANG=en_US.UTF-8")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\n终端启动失败: "+err.Error()+"\r\n"))
		return err
	}
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}()

	var once sync.Once
	var writeMu sync.Mutex
	done := make(chan struct{})
	closeDone := func() { once.Do(func() { close(done) }) }
	writeMessage := func(messageType int, data []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(messageType, data)
	}

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				closeDone()
				return
			}
			if n > 0 {
				if err := writeMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					closeDone()
					return
				}
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := writeMessage(websocket.PingMessage, nil); err != nil {
					closeDone()
					return
				}
			}
		}
	}()

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			closeDone()
			return err
		}
		if len(msg) == 0 {
			continue
		}
		if messageType == websocket.BinaryMessage && msg[0] == 1 {
			var resize fleetShellResize
			if err := json.Unmarshal(msg[1:], &resize); err == nil {
				_ = pty.Setsize(ptmx, &pty.Winsize{Rows: resize.Rows, Cols: resize.Cols})
			}
			continue
		}
		if _, err := ptmx.Write(msg); err != nil {
			closeDone()
			return err
		}
	}
}

func fleetShellPath() string {
	for _, shell := range []string{"/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}
	return "/bin/sh"
}
