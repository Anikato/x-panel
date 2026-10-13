package v1

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"xpanel/app/service"
	"xpanel/global"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	global.LOG = logger
	os.Exit(m.Run())
}

func TestLockLocalTerminalKeepsSessionAndReattach(t *testing.T) {
	srv := startTerminalTestServer(t)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/terminal"
	before := childPIDs(os.Getpid())

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	pid := waitForShell(t, before)
	t.Cleanup(func() {
		for _, item := range service.ListTerminalSessions() {
			_ = service.CloseTerminalSession(item.ID)
		}
	})

	ack := readTerminalControl(t, conn)
	if ack["op"] != "session" || ack["id"] == "" || ack["message"] != "false" {
		t.Fatalf("ack = %#v", ack)
	}
	if err := service.SetTerminalSessionLocked(ack["id"], true, "编译"); err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()

	if !processExists(pid) {
		t.Fatalf("shell %d died after detach; sessions=%+v children=%s", pid, service.ListTerminalSessions(), debugChildren(os.Getpid()))
	}
	sessions := service.ListTerminalSessions()
	if len(sessions) != 1 || sessions[0].ID != ack["id"] || sessions[0].Name != "编译" || sessions[0].Status != "running" {
		t.Fatalf("sessions = %+v", sessions)
	}

	again, _, err := websocket.DefaultDialer.Dial(wsURL+"?session="+ack["id"], nil)
	if err != nil {
		t.Fatal(err)
	}
	defer again.Close()
	if err := again.WriteMessage(websocket.TextMessage, []byte("printf 'reattach-ok\\n'\n")); err != nil {
		t.Fatal(err)
	}
	if !readTerminalOutputContains(t, again, "reattach-ok") {
		t.Fatal("reattach did not show command output")
	}
}

func TestClosingLocalTerminalKillsTheShell(t *testing.T) {
	srv := startTerminalTestServer(t)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/terminal"
	before := childPIDs(os.Getpid())
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	pid := waitForShell(t, before)
	_ = conn.Close()
	deadline := time.Now().Add(3 * time.Second)
	for processExists(pid) && time.Now().Before(deadline) {
		time.Sleep(15 * time.Millisecond)
	}
	if processExists(pid) {
		t.Fatalf("shell %d still alive after close", pid)
	}
	if got := service.ListTerminalSessions(); len(got) != 0 {
		t.Fatalf("ephemeral close left sessions %+v", got)
	}
}

func startTerminalTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := &TerminalAPI{}
	r.GET("/terminal", api.WsTerminal)
	srv := httptest.NewServer(r)
	t.Cleanup(func() {
		srv.CloseClientConnections()
		srv.Close()
	})
	return srv
}

func readTerminalControl(t *testing.T, conn *websocket.Conn) map[string]string {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		kind, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		if kind != websocket.TextMessage {
			continue
		}
		var payload map[string]string
		if json.Unmarshal(msg, &payload) == nil && payload["op"] != "" {
			return payload
		}
	}
}

func readTerminalOutputContains(t *testing.T, conn *websocket.Conn, needle string) bool {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	var got strings.Builder
	for {
		kind, msg, err := conn.ReadMessage()
		if err != nil {
			t.Logf("output so far: %q", got.String())
			return false
		}
		if kind == websocket.TextMessage {
			continue
		}
		got.Write(msg)
		if strings.Contains(got.String(), needle) {
			return true
		}
	}
}

func childPIDs(parent int) map[int]struct{} {
	found := map[int]struct{}{}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return found
	}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		stat, err := os.ReadFile("/proc/" + entry.Name() + "/stat")
		if err != nil {
			continue
		}
		text := string(stat)
		end := strings.LastIndex(text, ")")
		if end < 0 || end+2 >= len(text) {
			continue
		}
		fields := strings.Fields(text[end+2:])
		if len(fields) < 2 {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err == nil && ppid == parent {
			found[pid] = struct{}{}
		}
	}
	return found
}

func waitForShell(t *testing.T, before map[int]struct{}) int {
	t.Helper()
	shell := getShell()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for pid := range childPIDs(os.Getpid()) {
			if _, ok := before[pid]; ok || !cmdlineIsShell(pid, shell) {
				continue
			}
			time.Sleep(80 * time.Millisecond)
			if processExists(pid) && cmdlineIsShell(pid, shell) {
				return pid
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("shell process did not start; children=%s", debugChildren(os.Getpid()))
	return 0
}

func cmdlineIsShell(pid int, shell string) bool {
	raw, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
	if err != nil {
		return false
	}
	text := strings.ReplaceAll(string(raw), "\x00", " ")
	return strings.Contains(text, shell)
}

func processExists(pid int) bool {
	_, err := os.Stat("/proc/" + strconv.Itoa(pid))
	return err == nil
}

func debugChildren(parent int) string {
	var b strings.Builder
	for pid := range childPIDs(parent) {
		cmd, _ := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
		fmt.Fprintf(&b, "%d:%q ", pid, strings.ReplaceAll(string(cmd), "\x00", " "))
	}
	return b.String()
}
