package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunBackupHookSkipsEmptyCommand(t *testing.T) {
	result := runBackupHook("pre", "  \n  ", time.Second)
	if !result.Skipped || !result.OK {
		t.Fatalf("empty hook should skip as success: %#v", result)
	}
	if !strings.Contains(result.Line(), "[pre]") || !strings.Contains(result.Line(), "SKIP") {
		t.Fatalf("step line = %q", result.Line())
	}
}

func TestRunBackupHookReportsCommandSuccess(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "ok")
	result := runBackupHook("pre", "touch "+marker, 5*time.Second)
	if result.Skipped || !result.OK {
		t.Fatalf("hook should succeed: %#v", result)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("command did not run: %v", err)
	}
	if !strings.Contains(result.Line(), "OK") {
		t.Fatalf("step line = %q", result.Line())
	}
}

func TestRunBackupHookReportsCommandFailure(t *testing.T) {
	result := runBackupHook("pre", "exit 7", 5*time.Second)
	if result.OK || result.Skipped {
		t.Fatalf("hook should fail: %#v", result)
	}
	if !strings.Contains(result.Line(), "FAIL") {
		t.Fatalf("step line = %q", result.Line())
	}
}

func TestRunDirectoryHooksRunsPostAfterPackFailure(t *testing.T) {
	dir := t.TempDir()
	pre := filepath.Join(dir, "pre")
	post := filepath.Join(dir, "post")

	result := runDirectoryBackupHooks(
		"touch "+pre,
		"touch "+post,
		func() error { return os.ErrInvalid },
	)
	if result.OK {
		t.Fatal("pack failure should fail the job")
	}
	if _, err := os.Stat(pre); err != nil {
		t.Fatalf("pre command should run: %v", err)
	}
	if _, err := os.Stat(post); err != nil {
		t.Fatalf("post command should still run after pack failure: %v", err)
	}
	if !strings.Contains(result.Log, "[pack]") || !strings.Contains(result.Log, "FAIL") {
		t.Fatalf("log missing pack failure: %s", result.Log)
	}
}

func TestRunDirectoryHooksSkipsPackAndPostWhenPreFails(t *testing.T) {
	dir := t.TempDir()
	post := filepath.Join(dir, "post")
	packed := false

	result := runDirectoryBackupHooks(
		"exit 1",
		"touch "+post,
		func() error {
			packed = true
			return nil
		},
	)
	if result.OK {
		t.Fatal("pre failure should fail the job")
	}
	if packed {
		t.Fatal("pack should not run after pre failure")
	}
	if _, err := os.Stat(post); !os.IsNotExist(err) {
		t.Fatal("post should not run after pre failure")
	}
}

func TestRunDirectoryHooksFailsWhenPostFailsAfterSuccessfulPack(t *testing.T) {
	packed := false
	result := runDirectoryBackupHooks(
		"",
		"exit 3",
		func() error {
			packed = true
			return nil
		},
	)
	if !packed {
		t.Fatal("pack should run")
	}
	if result.OK {
		t.Fatal("post failure should fail the job")
	}
	if !strings.Contains(result.Log, "[post]") || !strings.Contains(result.Log, "FAIL") {
		t.Fatalf("log missing post failure: %s", result.Log)
	}
}
