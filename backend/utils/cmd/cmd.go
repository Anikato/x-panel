package cmd

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// ExecWithOutput 执行命令并返回标准输出
func ExecWithOutput(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return stderr.String(), err
		}
		return "", err
	}
	return stdout.String(), nil
}

// ExecWithTimeoutAndOutput 带自定义超时的命令执行
func ExecWithTimeoutAndOutput(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if stderr.Len() > 0 {
			return stderr.String(), err
		}
		return "", err
	}
	return stdout.String(), nil
}

// Exec 执行命令，不关心输出
func Exec(name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Run()
}
