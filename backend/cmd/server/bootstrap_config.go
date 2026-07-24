package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"xpanel/app/repo"
)

const (
	securityEntranceEnv = "XPANEL_SECURITY_ENTRANCE"
	agentTokenEnv       = "XPANEL_AGENT_TOKEN"
)

func collectBootstrapSettings(getenv func(string) string) (map[string]string, error) {
	settings := make(map[string]string, 2)
	if value := strings.TrimSpace(getenv(securityEntranceEnv)); value != "" {
		settings["SecurityEntrance"] = strings.Trim(value, "/")
	}
	if value := strings.TrimSpace(getenv(agentTokenEnv)); value != "" {
		settings["AgentToken"] = value
	}
	if len(settings) == 0 {
		return nil, errors.New("未提供可写入的引导配置")
	}
	return settings, nil
}

func runBootstrapConfig(args []string) {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "错误: bootstrap-config 不接受参数，只从受限环境变量读取配置")
		os.Exit(1)
	}
	settings, err := collectBootstrapSettings(os.Getenv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	initializeOfflineDatabase()
	settingRepo := repo.NewISettingRepo()
	for key, value := range settings {
		if err := settingRepo.CreateOrUpdate(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "错误: 写入 %s 失败\n", key)
			os.Exit(1)
		}
	}
	fmt.Println("✓ 安全引导配置已写入")
}
