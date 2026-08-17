package service

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const backupHookTimeout = 5 * time.Minute

type hookResult struct {
	Name     string
	Skipped  bool
	OK       bool
	Duration time.Duration
	Output   string
	Err      error
}

func (r hookResult) Line() string {
	status := "OK"
	switch {
	case r.Skipped:
		status = "SKIP"
	case !r.OK:
		status = "FAIL"
	}
	line := fmt.Sprintf("[%s] %s %s", r.Name, status, r.Duration.Round(time.Millisecond))
	if r.Skipped {
		return line + " (empty)"
	}
	if r.Output != "" {
		line += "\n" + indentHookOutput(r.Output)
	}
	if r.Err != nil {
		line += "\n  error: " + r.Err.Error()
	}
	return line
}

type directoryHookResult struct {
	OK  bool
	Log string
	Err error
}

func runDirectoryBackupHooks(preCommand, postCommand string, pack func() error) directoryHookResult {
	var lines []string
	appendLine := func(line string) {
		lines = append(lines, line)
	}

	pre := runBackupHook("pre", preCommand, backupHookTimeout)
	appendLine(pre.Line())
	if !pre.OK {
		return directoryHookResult{Log: strings.Join(lines, "\n"), Err: hookError("pre", pre)}
	}

	packStart := time.Now()
	packErr := pack()
	packRes := hookResult{Name: "pack", OK: packErr == nil, Duration: time.Since(packStart), Err: packErr}
	if packErr != nil {
		packRes.Output = packErr.Error()
		packRes.Err = packErr
	}
	appendLine(packRes.Line())

	post := runBackupHook("post", postCommand, backupHookTimeout)
	appendLine(post.Line())

	log := strings.Join(lines, "\n")
	if packErr != nil {
		return directoryHookResult{Log: log, Err: packErr}
	}
	if !post.OK {
		return directoryHookResult{Log: log, Err: hookError("post", post)}
	}
	return directoryHookResult{OK: true, Log: log}
}

func runBackupHook(name, script string, timeout time.Duration) hookResult {
	result := hookResult{Name: name}
	script = strings.TrimSpace(script)
	if script == "" {
		result.Skipped = true
		result.OK = true
		return result
	}
	if timeout <= 0 {
		timeout = backupHookTimeout
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", script)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	result.Duration = time.Since(start)
	result.Output = strings.TrimSpace(buf.String())
	if ctx.Err() == context.DeadlineExceeded {
		result.Err = fmt.Errorf("timeout after %s", timeout)
		return result
	}
	if err != nil {
		result.Err = err
		return result
	}
	result.OK = true
	return result
}

func hookError(name string, result hookResult) error {
	if result.Err != nil {
		return fmt.Errorf("%s command failed: %w", name, result.Err)
	}
	return fmt.Errorf("%s command failed", name)
}

func indentHookOutput(output string) string {
	lines := strings.Split(output, "\n")
	const maxLines = 40
	if len(lines) > maxLines {
		lines = append(lines[:maxLines], fmt.Sprintf("... (%d more lines)", len(lines)-maxLines))
	}
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}
