package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"xpanel/global"

	"github.com/creack/pty"
)

const (
	MaxRunningTerminalSessions  = 8
	terminalSessionBufferBytes  = 1 << 20
	terminalSessionChunkBacklog = 64
	terminalReconnectGrace      = 30 * time.Second
)

var (
	ErrTooManyTerminalSessions = errors.New("too many background terminals")
	ErrTerminalSessionNotFound = errors.New("terminal session not found")
	ErrTerminalSessionExited   = errors.New("terminal session exited")
)

// TerminalSessionInfo is one background shell kept by the panel process.
type TerminalSessionInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Cwd       string    `json:"cwd"`
	StartedAt time.Time `json:"startedAt"`
	Status    string    `json:"status"`
	Attached  bool      `json:"attached"`
	Locked    bool      `json:"locked"`
}

// TerminalSubscription receives live output for one attached browser.
type TerminalSubscription struct {
	Chunks   chan []byte
	Events   chan string
	done     chan struct{}
	stopOnce sync.Once
}

// BackgroundTerminalsError refuses panel upgrade or restart while shells are running.
type BackgroundTerminalsError struct {
	Sessions []TerminalSessionInfo
}

func (e *BackgroundTerminalsError) Error() string {
	names := make([]string, 0, len(e.Sessions))
	for _, item := range e.Sessions {
		names = append(names, item.Name)
	}
	return fmt.Sprintf("还有 %d 个后台终端在运行（%s）。请先终止或等它们结束后再升级或重启面板", len(names), strings.Join(names, "、"))
}

type terminalRing struct {
	buf []byte
	max int
}

func newTerminalRing(max int) *terminalRing {
	return &terminalRing{max: max}
}

func (r *terminalRing) write(p []byte) {
	if r.max <= 0 || len(p) == 0 {
		return
	}
	if len(p) >= r.max {
		p = p[len(p)-r.max:]
		r.buf = append([]byte(nil), p...)
		return
	}
	r.buf = append(r.buf, p...)
	if len(r.buf) > r.max {
		r.buf = append([]byte(nil), r.buf[len(r.buf)-r.max:]...)
	}
}

func (r *terminalRing) snapshot() []byte {
	if len(r.buf) == 0 {
		return nil
	}
	return append([]byte(nil), r.buf...)
}

type terminalSession struct {
	id          string
	name        string
	cwd         string
	startedAt   time.Time
	ptmx        *os.File
	cmd         *exec.Cmd
	ring        *terminalRing
	sub         *TerminalSubscription
	exited      bool
	locked      bool
	mu          sync.Mutex
	waitOnce    sync.Once
	detachTimer *time.Timer
	debugBytes  uint64
	debugChunks uint64
	debugLast   time.Time
}

type terminalRegistry struct {
	mu    sync.Mutex
	items map[string]*terminalSession
}

var backgroundTerminals = terminalRegistry{items: map[string]*terminalSession{}}

// RegisterTerminalSession takes ownership of a local shell that dies when its
// browser disconnects. On error the caller still owns ptmx and cmd.
func RegisterTerminalSession(ptmx *os.File, cmd *exec.Cmd, name, cwd string) (TerminalSessionInfo, error) {
	return registerTerminalSession(ptmx, cmd, name, cwd, false)
}

// AdoptTerminalSession takes ownership of a shell that already stays after disconnect.
// On error the caller still owns ptmx and cmd.
func AdoptTerminalSession(ptmx *os.File, cmd *exec.Cmd, name, cwd string) (TerminalSessionInfo, error) {
	return registerTerminalSession(ptmx, cmd, name, cwd, true)
}

func registerTerminalSession(ptmx *os.File, cmd *exec.Cmd, name, cwd string, locked bool) (TerminalSessionInfo, error) {
	if ptmx == nil || cmd == nil || cmd.Process == nil {
		return TerminalSessionInfo{}, errors.New("terminal session is not running")
	}
	backgroundTerminals.mu.Lock()
	defer backgroundTerminals.mu.Unlock()
	if locked && runningTerminalCountLocked() >= MaxRunningTerminalSessions {
		return TerminalSessionInfo{}, ErrTooManyTerminalSessions
	}
	session := &terminalSession{
		id:        newTerminalSessionID(),
		name:      normalizeTerminalSessionName(name, cwd),
		cwd:       cwd,
		startedAt: time.Now(),
		ptmx:      ptmx,
		cmd:       cmd,
		ring:      newTerminalRing(terminalSessionBufferBytes),
		locked:    locked,
	}
	backgroundTerminals.items[session.id] = session
	go session.readLoop()
	return session.view(), nil
}

func (s *terminalSession) view() TerminalSessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := "running"
	if s.exited {
		status = "exited"
	}
	return TerminalSessionInfo{
		ID:        s.id,
		Name:      s.name,
		Cwd:       s.cwd,
		StartedAt: s.startedAt,
		Status:    status,
		Attached:  s.sub != nil && !s.exited,
		Locked:    s.locked,
	}
}

func runningTerminalCountLocked() int {
	count := 0
	for _, session := range backgroundTerminals.items {
		session.mu.Lock()
		locked := session.locked
		exited := session.exited
		session.mu.Unlock()
		if locked && !exited {
			count++
		}
	}
	return count
}

func runningLockedCountExcept(skip *terminalSession) int {
	count := 0
	for _, session := range backgroundTerminals.items {
		if session == skip {
			continue
		}
		session.mu.Lock()
		locked := session.locked
		exited := session.exited
		session.mu.Unlock()
		if locked && !exited {
			count++
		}
	}
	return count
}

// ListTerminalSessions returns running sessions first, newest first within each status.
func ListTerminalSessions() []TerminalSessionInfo {
	backgroundTerminals.mu.Lock()
	defer backgroundTerminals.mu.Unlock()
	items := make([]TerminalSessionInfo, 0, len(backgroundTerminals.items))
	for _, session := range backgroundTerminals.items {
		view := session.view()
		if !view.Locked {
			continue
		}
		items = append(items, view)
	}
	sortTerminalSessions(items)
	return items
}

func sortTerminalSessions(items []TerminalSessionInfo) {
	for i := 1; i < len(items); i++ {
		item := items[i]
		j := i
		for j > 0 && terminalSessionLess(item, items[j-1]) {
			items[j] = items[j-1]
			j--
		}
		items[j] = item
	}
}

func terminalSessionLess(a, b TerminalSessionInfo) bool {
	if (a.Status == "running") != (b.Status == "running") {
		return a.Status == "running"
	}
	return a.StartedAt.After(b.StartedAt)
}

// RunningTerminalSessions is the set that dies when the panel process restarts.
func RunningTerminalSessions() []TerminalSessionInfo {
	running := make([]TerminalSessionInfo, 0)
	for _, item := range ListTerminalSessions() {
		if item.Status == "running" {
			running = append(running, item)
		}
	}
	return running
}

// GuardPanelProcessRestart reports running background shells.
func GuardPanelProcessRestart() error {
	running := RunningTerminalSessions()
	if len(running) == 0 {
		return nil
	}
	return &BackgroundTerminalsError{Sessions: running}
}

// RenameTerminalSession changes the label shown in the session list.
func RenameTerminalSession(id, name string) error {
	session := lookupTerminalSession(id)
	if session == nil {
		return ErrTerminalSessionNotFound
	}
	session.mu.Lock()
	session.name = normalizeTerminalSessionName(name, session.cwd)
	session.mu.Unlock()
	return nil
}

// CloseTerminalSession kills a running shell and drops the record.
func CloseTerminalSession(id string) error {
	backgroundTerminals.mu.Lock()
	session := backgroundTerminals.items[id]
	if session == nil {
		backgroundTerminals.mu.Unlock()
		return ErrTerminalSessionNotFound
	}
	delete(backgroundTerminals.items, id)
	session.mu.Lock()
	if session.detachTimer != nil {
		session.detachTimer.Stop()
		session.detachTimer = nil
	}
	session.mu.Unlock()
	backgroundTerminals.mu.Unlock()
	terminalLog("session %s deleted: explicit close", id)
	session.halt()
	return nil
}

// SubscribeTerminalSession replays the retained tail and attaches one browser.
// A later subscribe takes the previous browser's place.
func SubscribeTerminalSession(id string) ([]byte, *TerminalSubscription, error) {
	backgroundTerminals.mu.Lock()
	session := backgroundTerminals.items[id]
	if session == nil {
		backgroundTerminals.mu.Unlock()
		return nil, nil, ErrTerminalSessionNotFound
	}
	session.mu.Lock()
	backgroundTerminals.mu.Unlock()
	defer session.mu.Unlock()
	if session.detachTimer != nil {
		session.detachTimer.Stop()
		session.detachTimer = nil
	}
	terminalLog("session %s attached", id)
	replay := session.ring.snapshot()
	if session.sub != nil {
		notifyTerminalEvent(session.sub, "taken")
		session.sub.stop()
		session.sub = nil
	}
	sub := newTerminalSubscription()
	session.sub = sub
	if session.exited {
		notifyTerminalEvent(sub, "exited")
		return replay, sub, nil
	}
	return replay, sub, nil
}

// UnsubscribeTerminalSession allows an unlocked session a short reconnect window.
func UnsubscribeTerminalSession(id string, sub *TerminalSubscription) {
	if sub == nil {
		return
	}
	backgroundTerminals.mu.Lock()
	session := backgroundTerminals.items[id]
	if session == nil {
		backgroundTerminals.mu.Unlock()
		return
	}
	session.mu.Lock()
	if session.sub == sub {
		sub.stop()
		session.sub = nil
		if !session.locked {
			if session.detachTimer != nil {
				session.detachTimer.Stop()
			}
			var timer *time.Timer
			timer = time.AfterFunc(terminalReconnectGrace, func() { expireDetachedTerminal(id, session, timer) })
			session.detachTimer = timer
			terminalLog("session %s detached: unlocked; reconnect grace=%s", id, terminalReconnectGrace)
		} else {
			terminalLog("session %s detached: retained in background", id)
		}
	}
	session.mu.Unlock()
	backgroundTerminals.mu.Unlock()
}

func expireDetachedTerminal(id string, session *terminalSession, timer *time.Timer) {
	backgroundTerminals.mu.Lock()
	session.mu.Lock()
	drop := backgroundTerminals.items[id] == session && session.detachTimer == timer && session.sub == nil && !session.locked
	if drop {
		delete(backgroundTerminals.items, id)
		session.detachTimer = nil
	}
	session.mu.Unlock()
	backgroundTerminals.mu.Unlock()
	if drop {
		terminalLog("session %s deleted: reconnect grace expired", id)
		session.halt()
	}
}

func terminalLog(format string, args ...interface{}) {
	if global.LOG != nil {
		global.LOG.Infof("Terminal "+format, args...)
	}
}

// SetTerminalSessionLocked keeps a shell after disconnect, or returns it to
// ending when the browser leaves. The 8-session limit counts locked shells only.
func SetTerminalSessionLocked(id string, locked bool, name string) error {
	backgroundTerminals.mu.Lock()
	session := backgroundTerminals.items[id]
	if session == nil {
		backgroundTerminals.mu.Unlock()
		return ErrTerminalSessionNotFound
	}
	session.mu.Lock()
	if locked && !session.locked && !session.exited && runningLockedCountExcept(session) >= MaxRunningTerminalSessions {
		session.mu.Unlock()
		backgroundTerminals.mu.Unlock()
		return ErrTooManyTerminalSessions
	}
	session.locked = locked
	if locked && session.detachTimer != nil {
		session.detachTimer.Stop()
		session.detachTimer = nil
	}
	if strings.TrimSpace(name) != "" {
		session.name = normalizeTerminalSessionName(name, session.cwd)
	}
	drop := !session.locked && session.sub == nil
	if drop {
		delete(backgroundTerminals.items, id)
	}
	session.mu.Unlock()
	backgroundTerminals.mu.Unlock()
	if drop {
		terminalLog("session %s deleted: unlocked while detached", id)
		session.halt()
	}
	return nil
}

// WriteTerminalSession sends keyboard input to the shell.
func WriteTerminalSession(id string, p []byte) error {
	session := lookupTerminalSession(id)
	if session == nil {
		return ErrTerminalSessionNotFound
	}
	session.mu.Lock()
	exited := session.exited
	ptmx := session.ptmx
	session.mu.Unlock()
	if exited || ptmx == nil {
		return ErrTerminalSessionExited
	}
	_, err := ptmx.Write(p)
	return err
}

// ResizeTerminalSession updates the PTY size for the attached browser.
func ResizeTerminalSession(id string, rows, cols uint16) error {
	session := lookupTerminalSession(id)
	if session == nil {
		return ErrTerminalSessionNotFound
	}
	session.mu.Lock()
	exited := session.exited
	ptmx := session.ptmx
	session.mu.Unlock()
	if exited || ptmx == nil {
		return ErrTerminalSessionExited
	}
	if rows == 0 || cols == 0 {
		return nil
	}
	return pty.Setsize(ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

func (s *terminalSession) readLoop() {
	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			s.publish(buf[:n])
		}
		if err != nil {
			terminalLog("session %s PTY read ended: %v", s.id, err)
			s.markExited()
			return
		}
	}
}

func (s *terminalSession) publish(p []byte) {
	chunk := append([]byte(nil), p...)
	s.mu.Lock()
	s.ring.write(chunk)
	if os.Getenv("XPANEL_TERMINAL_DEBUG") == "1" {
		s.debugBytes += uint64(len(chunk))
		s.debugChunks++
		if time.Since(s.debugLast) >= 5*time.Second {
			depth := 0
			if s.sub != nil {
				depth = len(s.sub.Chunks)
			}
			terminalLog("session %s debug: output_bytes=%d chunks=%d queue=%d/%d retained_bytes=%d", s.id, s.debugBytes, s.debugChunks, depth, terminalSessionChunkBacklog, len(s.ring.buf))
			s.debugLast = time.Now()
		}
	}
	sub := s.sub
	s.mu.Unlock()
	if sub == nil {
		return
	}
	select {
	case sub.Chunks <- chunk:
		return
	case <-sub.done:
		return
	default:
	}
	started := time.Now()
	if os.Getenv("XPANEL_TERMINAL_DEBUG") == "1" {
		terminalLog("session %s debug: output backpressure queue=%d/%d", s.id, len(sub.Chunks), terminalSessionChunkBacklog)
	}
	select {
	case sub.Chunks <- chunk:
	case <-sub.done:
	}
	if os.Getenv("XPANEL_TERMINAL_DEBUG") == "1" {
		terminalLog("session %s debug: output backpressure ended duration=%s", s.id, time.Since(started))
	}
}

func (s *terminalSession) markExited() {
	s.mu.Lock()
	if s.exited {
		s.mu.Unlock()
		return
	}
	s.exited = true
	ptmx := s.ptmx
	sub := s.sub
	s.mu.Unlock()
	if ptmx != nil {
		_ = ptmx.Close()
	}
	s.waitProc()
	notifyTerminalEvent(sub, "exited")
}

func (s *terminalSession) halt() {
	s.mu.Lock()
	cmd := s.cmd
	ptmx := s.ptmx
	exited := s.exited
	if s.sub != nil {
		s.sub.stop()
	}
	s.mu.Unlock()
	if !exited && cmd != nil && cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	if ptmx != nil {
		_ = ptmx.Close()
	}
	s.waitProc()
}

func (s *terminalSession) waitProc() {
	s.waitOnce.Do(func() {
		if s.cmd != nil {
			_ = s.cmd.Wait()
		}
	})
}

func newTerminalSubscription() *TerminalSubscription {
	return &TerminalSubscription{
		Chunks: make(chan []byte, terminalSessionChunkBacklog),
		Events: make(chan string, 4),
		done:   make(chan struct{}),
	}
}

func (sub *TerminalSubscription) stop() {
	sub.stopOnce.Do(func() { close(sub.done) })
}

func notifyTerminalEvent(sub *TerminalSubscription, event string) {
	if sub == nil {
		return
	}
	select {
	case sub.Events <- event:
	default:
	}
}

func lookupTerminalSession(id string) *terminalSession {
	backgroundTerminals.mu.Lock()
	defer backgroundTerminals.mu.Unlock()
	return backgroundTerminals.items[id]
}

func newTerminalSessionID() string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000000")))[:16]
	}
	return hex.EncodeToString(raw[:])
}

func normalizeTerminalSessionName(name, cwd string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	if name == "" {
		base := filepath.Base(cwd)
		if base == "" || base == "." || base == "/" || base == string(filepath.Separator) {
			return "终端"
		}
		name = base
	}
	runes := []rune(name)
	if len(runes) > 64 {
		name = string(runes[:64])
	}
	return name
}
