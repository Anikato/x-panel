package docker

import (
	"context"
	"os/exec"

	"github.com/docker/docker/client"
)

func NewClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func IsDockerAvailable() bool {
	cli, err := NewClient()
	if err != nil {
		return false
	}
	defer cli.Close()
	_, err = cli.Ping(context.Background())
	return err == nil
}

func IsDockerInstalled() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func GetDockerVersion() string {
	cli, err := NewClient()
	if err != nil {
		return ""
	}
	defer cli.Close()
	info, err := cli.ServerVersion(context.Background())
	if err != nil {
		return ""
	}
	return info.Version
}
