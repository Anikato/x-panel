package docker

import (
	"xpanel/utils/cmd"
)

// EnsureNetwork 确保 Docker 网络存在
func EnsureNetwork(networkName string) error {
	// 检查网络是否存在
	output, err := cmd.ExecWithOutput("docker", "network", "ls", "--filter", "name=^"+networkName+"$", "--format", "{{.Name}}")
	if err != nil {
		return err
	}

	// 如果网络不存在，创建它
	if output == "" {
		err := cmd.Exec("docker", "network", "create", networkName)
		return err
	}

	return nil
}
