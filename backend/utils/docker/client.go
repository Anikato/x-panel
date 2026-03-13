package docker

import (
	"context"

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
