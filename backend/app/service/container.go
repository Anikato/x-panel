package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"xpanel/app/dto"
	dockerUtil "xpanel/utils/docker"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
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

	DockerStatus() dto.DockerStatusResp

	// 新增功能
	Inspect(req dto.InspectReq) (string, error)
	Prune(req dto.PruneReq) (dto.PruneReport, error)
	RenameContainer(req dto.ContainerRename) error
	CleanContainerLog(req dto.ContainerLogClean) error
	CommitContainer(req dto.ContainerCommit) error
	LoadDockerMirrors() ([]string, error)
	UpdateDockerMirrors(mirrors []string) error
	ControlDockerService(action string) error
}

func NewIContainerService() IContainerService {
	return &ContainerService{}
}

type ContainerService struct{}

func (s *ContainerService) DockerStatus() dto.DockerStatusResp {
	resp := dto.DockerStatusResp{}
	resp.IsExist = dockerUtil.IsDockerInstalled()
	if !resp.IsExist {
		return resp
	}
	resp.IsActive = dockerUtil.IsDockerAvailable()
	if resp.IsActive {
		resp.Version = dockerUtil.GetDockerVersion()
	}
	return resp
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

		info := dto.ContainerInfo{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			State:   c.State,
			Status:  c.Status,
			Created: c.Created,
			Ports:   formatPorts(c.Ports),
			RunTime: c.Status,
		}

		// Extract IP from network settings
		if c.NetworkSettings != nil && c.NetworkSettings.Networks != nil {
			for _, net := range c.NetworkSettings.Networks {
				if net.IPAddress != "" {
					info.IPAddress = net.IPAddress
					break
				}
			}
		}

		items = append(items, info)
	}

	// Batch collect stats for running containers
	statsMap := s.batchContainerStats(cli, items)
	for i, item := range items {
		if st, ok := statsMap[item.ID]; ok {
			items[i].CPUPercent = st.cpuPercent
			items[i].MemUsage = st.memUsage
			items[i].MemLimit = st.memLimit
			items[i].MemPercent = st.memPercent
		}
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

type containerStats struct {
	cpuPercent float64
	memUsage   int64
	memLimit   int64
	memPercent float64
}

func (s *ContainerService) batchContainerStats(cli *client.Client, items []dto.ContainerInfo) map[string]containerStats {
	result := make(map[string]containerStats)
	for _, item := range items {
		if item.State != "running" {
			continue
		}
		st, err := s.getOneContainerStats(cli, item.ID)
		if err == nil {
			result[item.ID] = st
		}
	}
	return result
}

func (s *ContainerService) getOneContainerStats(cli *client.Client, id string) (containerStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := cli.ContainerStatsOneShot(ctx, id)
	if err != nil {
		return containerStats{}, err
	}
	defer resp.Body.Close()

	var stats struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs     int    `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
		} `json:"memory_stats"`
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return containerStats{}, err
	}
	if err := json.Unmarshal(data, &stats); err != nil {
		return containerStats{}, err
	}

	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)
	cpuPercent := 0.0
	if sysDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / sysDelta) * float64(stats.CPUStats.OnlineCPUs) * 100.0
	}

	memPercent := 0.0
	if stats.MemoryStats.Limit > 0 {
		memPercent = float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	}

	return containerStats{
		cpuPercent: cpuPercent,
		memUsage:   int64(stats.MemoryStats.Usage),
		memLimit:   int64(stats.MemoryStats.Limit),
		memPercent: memPercent,
	}, nil
}

func formatPorts(ports []container.Port) string {
	if len(ports) == 0 {
		return ""
	}
	var parts []string
	for _, p := range ports {
		if p.PublicPort > 0 {
			parts = append(parts, fmt.Sprintf("%s:%d->%d/%s", p.IP, p.PublicPort, p.PrivatePort, p.Type))
		} else {
			parts = append(parts, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
		}
	}
	return strings.Join(parts, ", ")
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

// ======================== 新增功能 ========================

func (s *ContainerService) Inspect(req dto.InspectReq) (string, error) {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return "", err
	}
	defer cli.Close()
	ctx := context.Background()

	var inspectInfo interface{}
	switch req.Type {
	case "container":
		inspectInfo, err = cli.ContainerInspect(ctx, req.ID)
	case "image":
		inspectInfo, _, err = cli.ImageInspectWithRaw(ctx, req.ID)
	case "network":
		inspectInfo, err = cli.NetworkInspect(ctx, req.ID, network.InspectOptions{})
	case "volume":
		inspectInfo, err = cli.VolumeInspect(ctx, req.ID)
	default:
		return "", fmt.Errorf("unknown inspect type: %s", req.Type)
	}
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(inspectInfo, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *ContainerService) Prune(req dto.PruneReq) (dto.PruneReport, error) {
	report := dto.PruneReport{}
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return report, err
	}
	defer cli.Close()
	ctx := context.Background()
	pruneFilters := filters.NewArgs()

	switch req.PruneType {
	case "container":
		rep, e := cli.ContainersPrune(ctx, pruneFilters)
		if e != nil {
			return report, e
		}
		report.DeletedCount = len(rep.ContainersDeleted)
		report.SpaceReclaimed = int64(rep.SpaceReclaimed)
	case "image":
		if !req.WithAll {
			pruneFilters.Add("dangling", "true")
		}
		rep, e := cli.ImagesPrune(ctx, pruneFilters)
		if e != nil {
			return report, e
		}
		report.DeletedCount = len(rep.ImagesDeleted)
		report.SpaceReclaimed = int64(rep.SpaceReclaimed)
	case "network":
		rep, e := cli.NetworksPrune(ctx, pruneFilters)
		if e != nil {
			return report, e
		}
		report.DeletedCount = len(rep.NetworksDeleted)
	case "volume":
		rep, e := cli.VolumesPrune(ctx, pruneFilters)
		if e != nil {
			return report, e
		}
		report.DeletedCount = len(rep.VolumesDeleted)
		report.SpaceReclaimed = int64(rep.SpaceReclaimed)
	case "buildcache":
		rep, e := cli.BuildCachePrune(ctx, types.BuildCachePruneOptions{All: true})
		if e != nil {
			return report, e
		}
		report.DeletedCount = len(rep.CachesDeleted)
		report.SpaceReclaimed = int64(rep.SpaceReclaimed)
	default:
		return report, fmt.Errorf("unknown prune type: %s", req.PruneType)
	}
	return report, nil
}

func (s *ContainerService) RenameContainer(req dto.ContainerRename) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return cli.ContainerRename(context.Background(), req.ContainerID, req.NewName)
}

func (s *ContainerService) CleanContainerLog(req dto.ContainerLogClean) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	ctx := context.Background()

	info, err := cli.ContainerInspect(ctx, req.ContainerID)
	if err != nil {
		return err
	}
	logPath := info.LogPath
	if logPath == "" {
		return fmt.Errorf("container log path not found")
	}

	// stop → truncate → start
	_ = cli.ContainerStop(ctx, req.ContainerID, container.StopOptions{})

	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		// 尝试重启容器
		_ = cli.ContainerStart(ctx, req.ContainerID, container.StartOptions{})
		return fmt.Errorf("failed to open log file: %v", err)
	}
	_ = f.Truncate(0)
	f.Close()

	// 清理轮转日志
	rotated, _ := filepath.Glob(logPath + ".*")
	for _, r := range rotated {
		_ = os.Remove(r)
	}

	return cli.ContainerStart(ctx, req.ContainerID, container.StartOptions{})
}

func (s *ContainerService) CommitContainer(req dto.ContainerCommit) error {
	cli, err := dockerUtil.NewClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	_, err = cli.ContainerCommit(context.Background(), req.ContainerID, container.CommitOptions{
		Reference: req.NewImageName,
		Comment:   req.Comment,
		Pause:     req.Pause,
	})
	return err
}

const daemonJSONPath = "/etc/docker/daemon.json"

func (s *ContainerService) LoadDockerMirrors() ([]string, error) {
	data, err := os.ReadFile(daemonJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var conf map[string]interface{}
	if err := json.Unmarshal(data, &conf); err != nil {
		return nil, fmt.Errorf("parse daemon.json failed: %v", err)
	}
	mirrors, ok := conf["registry-mirrors"]
	if !ok {
		return []string{}, nil
	}
	arr, ok := mirrors.([]interface{})
	if !ok {
		return []string{}, nil
	}
	var result []string
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result, nil
}

func (s *ContainerService) UpdateDockerMirrors(mirrors []string) error {
	var conf map[string]interface{}

	data, err := os.ReadFile(daemonJSONPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		conf = make(map[string]interface{})
	} else {
		if err := json.Unmarshal(data, &conf); err != nil {
			return fmt.Errorf("parse daemon.json failed: %v", err)
		}
	}

	if len(mirrors) == 0 {
		delete(conf, "registry-mirrors")
	} else {
		conf["registry-mirrors"] = mirrors
	}

	out, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll("/etc/docker", 0755); err != nil {
		return err
	}
	if err := os.WriteFile(daemonJSONPath, out, 0644); err != nil {
		return err
	}

	// reload + restart docker
	if output, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload failed: %s", string(output))
	}
	if output, err := exec.Command("systemctl", "restart", "docker").CombinedOutput(); err != nil {
		return fmt.Errorf("restart docker failed: %s", string(output))
	}
	return nil
}

// ControlDockerService controls the Docker systemd service (start / stop / restart)
func (s *ContainerService) ControlDockerService(action string) error {
	switch action {
	case "start", "stop", "restart":
		out, err := exec.Command("systemctl", action, "docker").CombinedOutput()
		if err != nil {
			return fmt.Errorf("systemctl %s docker failed: %s", action, string(out))
		}
		return nil
	default:
		return fmt.Errorf("unsupported docker service action: %s (use start/stop/restart)", action)
	}
}
