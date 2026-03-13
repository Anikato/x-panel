package service

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"xpanel/app/dto"
	dockerUtil "xpanel/utils/docker"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
)

type IContainerService interface {
	ListContainers(req dto.ContainerSearch) (int64, []dto.ContainerInfo, error)
	CreateContainer(req dto.ContainerCreate) error
	OperateContainer(req dto.ContainerOperate) error
	ContainerLogs(req dto.ContainerLog) (string, error)
	RemoveContainer(containerID string) error

	ListImages() ([]dto.ImageInfo, error)
	PullImage(req dto.ImagePull) error
	RemoveImage(imageID string) error

	ListNetworks() ([]dto.NetworkInfo, error)
	CreateNetwork(req dto.NetworkCreate) error
	RemoveNetwork(networkID string) error

	ListVolumes() ([]dto.VolumeInfo, error)
	CreateVolume(req dto.VolumeCreate) error
	RemoveVolume(name string) error

	DockerAvailable() bool
}

func NewIContainerService() IContainerService {
	return &ContainerService{}
}

type ContainerService struct{}

func (s *ContainerService) DockerAvailable() bool {
	return dockerUtil.IsDockerAvailable()
}

func (s *ContainerService) ListContainers(req dto.ContainerSearch) (int64, []dto.ContainerInfo, error) {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return 0, nil, err
	}
	defer cli.Close()

	opts := container.ListOptions{All: true}
	if req.State != "" {
		opts.Filters = filters.NewArgs(filters.Arg("status", req.State))
	}
	containers, err := cli.ContainerList(context.Background(), opts)
	if err != nil {
		return 0, nil, err
	}

	var items []dto.ContainerInfo
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		if req.Name != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(req.Name)) {
			continue
		}
		items = append(items, dto.ContainerInfo{
			ID: c.ID[:12], Name: name, Image: c.Image,
			State: c.State, Status: c.Status, Created: c.Created,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Created > items[j].Created })
	total := int64(len(items))
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > int(total) {
		return total, nil, nil
	}
	if end > int(total) {
		end = int(total)
	}
	return total, items[start:end], nil
}

func (s *ContainerService) CreateContainer(req dto.ContainerCreate) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	config := &container.Config{
		Image:  req.Image,
		Env:    req.Env,
		Cmd:    req.Cmd,
		Labels: req.Labels,
	}

	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}
	for _, p := range req.Ports {
		proto := p.Protocol
		if proto == "" {
			proto = "tcp"
		}
		cp := nat.Port(fmt.Sprintf("%s/%s", p.Container, proto))
		exposedPorts[cp] = struct{}{}
		portBindings[cp] = []nat.PortBinding{{HostPort: p.Host}}
	}
	config.ExposedPorts = exposedPorts

	var binds []string
	for _, v := range req.Volumes {
		binds = append(binds, fmt.Sprintf("%s:%s", v.Host, v.Container))
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Binds:        binds,
		Resources: container.Resources{
			NanoCPUs: req.NanoCPUs,
			Memory:   req.Memory,
		},
	}
	if req.RestartPolicy != "" {
		hostConfig.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyMode(req.RestartPolicy)}
	}

	resp, err := cli.ContainerCreate(context.Background(), config, hostConfig, nil, nil, req.Name)
	if err != nil {
		return err
	}
	return cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{})
}

func (s *ContainerService) OperateContainer(req dto.ContainerOperate) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	ctx := context.Background()

	switch req.Operation {
	case "start":
		return cli.ContainerStart(ctx, req.ContainerID, container.StartOptions{})
	case "stop":
		return cli.ContainerStop(ctx, req.ContainerID, container.StopOptions{})
	case "restart":
		return cli.ContainerRestart(ctx, req.ContainerID, container.StopOptions{})
	case "pause":
		return cli.ContainerPause(ctx, req.ContainerID)
	case "unpause":
		return cli.ContainerUnpause(ctx, req.ContainerID)
	default:
		return fmt.Errorf("unknown operation: %s", req.Operation)
	}
}

func (s *ContainerService) ContainerLogs(req dto.ContainerLog) (string, error) {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return "", err
	}
	defer cli.Close()

	opts := container.LogsOptions{ShowStdout: true, ShowStderr: true}
	if req.Since != "" {
		opts.Since = req.Since
	}
	if req.Tail != "" {
		opts.Tail = req.Tail
	} else {
		opts.Tail = "200"
	}

	reader, err := cli.ContainerLogs(context.Background(), req.ContainerID, opts)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	data, _ := io.ReadAll(reader)
	return string(data), nil
}

func (s *ContainerService) RemoveContainer(containerID string) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true, RemoveVolumes: true})
}

func (s *ContainerService) ListImages() ([]dto.ImageInfo, error) {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	images, err := cli.ImageList(context.Background(), image.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var items []dto.ImageInfo
	for _, img := range images {
		items = append(items, dto.ImageInfo{
			ID: img.ID[7:19], Tags: img.RepoTags, Size: img.Size, Created: img.Created,
		})
	}
	return items, nil
}

func (s *ContainerService) PullImage(req dto.ImagePull) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	reader, err := cli.ImagePull(context.Background(), req.ImageName, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

func (s *ContainerService) RemoveImage(imageID string) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	_, err = cli.ImageRemove(context.Background(), imageID, image.RemoveOptions{Force: true, PruneChildren: true})
	return err
}

func (s *ContainerService) ListNetworks() ([]dto.NetworkInfo, error) {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	nets, err := cli.NetworkList(context.Background(), network.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []dto.NetworkInfo
	for _, n := range nets {
		subnet, gateway := "", ""
		if len(n.IPAM.Config) > 0 {
			subnet = n.IPAM.Config[0].Subnet
			gateway = n.IPAM.Config[0].Gateway
		}
		items = append(items, dto.NetworkInfo{
			ID: n.ID[:12], Name: n.Name, Driver: n.Driver,
			Subnet: subnet, Gateway: gateway, Created: n.Created.Format("2006-01-02 15:04:05"),
		})
	}
	return items, nil
}

func (s *ContainerService) CreateNetwork(req dto.NetworkCreate) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	opts := network.CreateOptions{Driver: req.Driver}
	if req.Subnet != "" || req.Gateway != "" {
		opts.IPAM = &network.IPAM{
			Config: []network.IPAMConfig{{Subnet: req.Subnet, Gateway: req.Gateway}},
		}
	}
	_, err = cli.NetworkCreate(context.Background(), req.Name, opts)
	return err
}

func (s *ContainerService) RemoveNetwork(networkID string) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return cli.NetworkRemove(context.Background(), networkID)
}

func (s *ContainerService) ListVolumes() ([]dto.VolumeInfo, error) {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	resp, err := cli.VolumeList(context.Background(), volume.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []dto.VolumeInfo
	for _, v := range resp.Volumes {
		items = append(items, dto.VolumeInfo{
			Name: v.Name, Driver: v.Driver, MountPoint: v.Mountpoint, Created: v.CreatedAt,
		})
	}
	return items, nil
}

func (s *ContainerService) CreateVolume(req dto.VolumeCreate) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	_, err = cli.VolumeCreate(context.Background(), volume.CreateOptions{Name: req.Name, Driver: req.Driver})
	return err
}

func (s *ContainerService) RemoveVolume(name string) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return cli.VolumeRemove(context.Background(), name, true)
}

// Compose operations use docker compose CLI
func (s *ContainerService) ComposeUp(path string) error {
	return nil // implemented in compose service
}
