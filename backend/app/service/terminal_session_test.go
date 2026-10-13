package service

import (
	"bytes"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestAdoptTerminalSessionKeepsProcessAndReplaysOutput(t *testing.T) {
	resetTerminalSessionsForTest(t)

	cmd := exec.Command("sh", "-c", "printf 'ready\\n'; sleep 60")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("pty start: %v", err)
	}
	pid := cmd.Process.Pid

	view, err := AdoptTerminalSession(ptmx, cmd, "编译", "/tmp/work")
	if err != nil {
		t.Fatal(err)
	}
	if view.Name != "编译" || view.Status != "running" || view.Cwd != "/tmp/work" {
		t.Fatalf("view = %+v", view)
	}
	waitForTerminalReplay(t, view.ID, "ready")

	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("process died after adopt: %v", err)
	}

	if err := CloseTerminalSession(view.ID); err != nil {
		t.Fatal(err)
	}
	waitForProcessExit(t, pid)
	if _, err := terminalSessionByID(view.ID); err == nil {
		t.Fatal("closed session still listed")
	}
}

func TestAdoptTerminalSessionRejectsNinthRunningSession(t *testing.T) {
	resetTerminalSessionsForTest(t)

	for i := 0; i < MaxRunningTerminalSessions; i++ {
		cmd := exec.Command("sleep", "60")
		ptmx, err := pty.Start(cmd)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := AdoptTerminalSession(ptmx, cmd, "", "/tmp"); err != nil {
			t.Fatal(err)
		}
	}

	extra := exec.Command("sleep", "60")
	ptmx, err := pty.Start(extra)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if extra.Process != nil {
			_ = syscall.Kill(-extra.Process.Pid, syscall.SIGKILL)
		}
		_ = ptmx.Close()
	})

	if _, err := AdoptTerminalSession(ptmx, extra, "多余", "/tmp"); err != ErrTooManyTerminalSessions {
		t.Fatalf("ninth adopt error = %v", err)
	}
	if err := syscall.Kill(extra.Process.Pid, 0); err != nil {
		t.Fatalf("rejected session process was killed: %v", err)
	}
	if got := len(RunningTerminalSessions()); got != MaxRunningTerminalSessions {
		t.Fatalf("running = %d", got)
	}
}

func TestExitedTerminalSessionStaysUntilDismissedAndDoesNotBlockRestart(t *testing.T) {
	resetTerminalSessionsForTest(t)

	cmd := exec.Command("sh", "-c", "printf 'done\\n'")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	view, err := AdoptTerminalSession(ptmx, cmd, "看结果", "/tmp")
	if err != nil {
		t.Fatal(err)
	}
	waitForTerminalStatus(t, view.ID, "exited")
	if err := GuardPanelProcessRestart(); err != nil {
		t.Fatalf("exited session blocked restart: %v", err)
	}
	replay := terminalReplayForTest(t, view.ID)
	if !bytes.Contains(replay, []byte("done")) {
		t.Fatalf("replay = %q", replay)
	}
	if err := CloseTerminalSession(view.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := terminalSessionByID(view.ID); err == nil {
		t.Fatal("dismissed session still listed")
	}
}

func TestGuardPanelProcessRestartNamesRunningSessions(t *testing.T) {
	resetTerminalSessionsForTest(t)

	cmd := exec.Command("sleep", "60")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptTerminalSession(ptmx, cmd, "编译前端", "/src"); err != nil {
		t.Fatal(err)
	}

	err = GuardPanelProcessRestart()
	blocked, ok := err.(*BackgroundTerminalsError)
	if !ok || len(blocked.Sessions) != 1 || blocked.Sessions[0].Name != "编译前端" {
		t.Fatalf("guard error = %#v", err)
	}
	if blocked.Error() == "" {
		t.Fatal("empty guard message")
	}
}

func TestRenameTerminalSession(t *testing.T) {
	resetTerminalSessionsForTest(t)

	cmd := exec.Command("sleep", "60")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	view, err := AdoptTerminalSession(ptmx, cmd, "", "/var/www")
	if err != nil {
		t.Fatal(err)
	}
	if view.Name != "www" {
		t.Fatalf("default name = %q", view.Name)
	}
	if err := RenameTerminalSession(view.ID, "  发布  "); err != nil {
		t.Fatal(err)
	}
	got, err := terminalSessionByID(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "发布" {
		t.Fatalf("name = %q", got.Name)
	}
}

func TestTerminalRingKeepsTail(t *testing.T) {
	ring := newTerminalRing(8)
	ring.write([]byte("abcdefghij"))
	if got := string(ring.snapshot()); got != "cdefghij" {
		t.Fatalf("snapshot = %q", got)
	}
}

func TestSecondSubscribeTakesOverTheSession(t *testing.T) {
	resetTerminalSessionsForTest(t)

	cmd := exec.Command("sleep", "60")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	view, err := AdoptTerminalSession(ptmx, cmd, "会话", "/tmp")
	if err != nil {
		t.Fatal(err)
	}
	_, first, err := SubscribeTerminalSession(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := SubscribeTerminalSession(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-first.Events:
		if ev != "taken" {
			t.Fatalf("event = %q", ev)
		}
	case <-time.After(time.Second):
		t.Fatal("first subscriber was not notified")
	}
	UnsubscribeTerminalSession(view.ID, second)
}

func terminalSessionByID(id string) (TerminalSessionInfo, error) {
	for _, item := range ListTerminalSessions() {
		if item.ID == id {
			return item, nil
		}
	}
	return TerminalSessionInfo{}, ErrTerminalSessionNotFound
}

func terminalReplayForTest(t *testing.T, id string) []byte {
	t.Helper()
	replay, sub, err := SubscribeTerminalSession(id)
	if err != nil {
		t.Fatal(err)
	}
	UnsubscribeTerminalSession(id, sub)
	return replay
}

func resetTerminalSessionsForTest(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		backgroundTerminals.mu.Lock()
		ids := make([]string, 0, len(backgroundTerminals.items))
		for id := range backgroundTerminals.items {
			ids = append(ids, id)
		}
		backgroundTerminals.mu.Unlock()
		for _, id := range ids {
			_ = CloseTerminalSession(id)
		}
	})
}

func TestUnlockedSessionIsHiddenAndEndsWhenDetached(t *testing.T) {
	resetTerminalSessionsForTest(t)

	cmd := exec.Command("sleep", "60")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	view, err := RegisterTerminalSession(ptmx, cmd, "", "/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if view.Locked {
		t.Fatal("new local session is locked")
	}
	if len(ListTerminalSessions()) != 0 {
		t.Fatal("unlocked session was listed")
	}
	if err := GuardPanelProcessRestart(); err != nil {
		t.Fatalf("unlocked session blocked restart: %v", err)
	}
	_, sub, err := SubscribeTerminalSession(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	UnsubscribeTerminalSession(view.ID, sub)
	waitForProcessExit(t, cmd.Process.Pid)
	if _, err := terminalSessionByID(view.ID); err == nil {
		t.Fatal("detached unlocked session still listed")
	}
}

func TestLockThenUnlockWithoutBrowserEndsTheShell(t *testing.T) {
	resetTerminalSessionsForTest(t)

	cmd := exec.Command("sleep", "60")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	view, err := RegisterTerminalSession(ptmx, cmd, "", "/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if err := SetTerminalSessionLocked(view.ID, true, "编译"); err != nil {
		t.Fatal(err)
	}
	got, err := terminalSessionByID(view.ID)
	if err != nil || got.Name != "编译" || !got.Locked {
		t.Fatalf("locked view = %+v %v", got, err)
	}
	_, sub, err := SubscribeTerminalSession(view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := SetTerminalSessionLocked(view.ID, false, ""); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(cmd.Process.Pid, 0); err != nil {
		t.Fatalf("unlock while attached killed the shell: %v", err)
	}
	UnsubscribeTerminalSession(view.ID, sub)
	waitForProcessExit(t, cmd.Process.Pid)
}

func TestNinthLockIsRejected(t *testing.T) {
	resetTerminalSessionsForTest(t)

	for i := 0; i < MaxRunningTerminalSessions; i++ {
		cmd := exec.Command("sleep", "60")
		ptmx, err := pty.Start(cmd)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := AdoptTerminalSession(ptmx, cmd, "", "/tmp"); err != nil {
			t.Fatal(err)
		}
	}
	extra := exec.Command("sleep", "60")
	ptmx, err := pty.Start(extra)
	if err != nil {
		t.Fatal(err)
	}
	view, err := RegisterTerminalSession(ptmx, extra, "", "/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if err := SetTerminalSessionLocked(view.ID, true, "多余"); err != ErrTooManyTerminalSessions {
		t.Fatalf("ninth lock error = %v", err)
	}
	if err := syscall.Kill(extra.Process.Pid, 0); err != nil {
		t.Fatalf("rejected lock killed the shell: %v", err)
	}
	if got := len(RunningTerminalSessions()); got != MaxRunningTerminalSessions {
		t.Fatalf("running = %d", got)
	}
}

func waitForTerminalReplay(t *testing.T, id, needle string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if bytes.Contains(terminalReplayForTest(t, id), []byte(needle)) {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("replay missing %q, have %q", needle, terminalReplayForTest(t, id))
}

func waitForTerminalStatus(t *testing.T, id, status string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		view, err := terminalSessionByID(id)
		if err == nil && view.Status == status {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	view, _ := terminalSessionByID(id)
	t.Fatalf("status = %+v, want %s", view, status)
}

func waitForProcessExit(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("pid %d still alive", pid)
}
