package dto

// Container
type ContainerSearch struct {
	PageInfo
	State string `json:"state"`
	Name  string `json:"name"`
}

type ContainerInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	State   string `json:"state"`
	Status  string `json:"status"`
	Created int64  `json:"created"`
}

type ContainerCreate struct {
	Name          string            `json:"name" binding:"required"`
	Image         string            `json:"image" binding:"required"`
	Ports         []PortBinding     `json:"ports"`
	Volumes       []VolumeBinding   `json:"volumes"`
	Env           []string          `json:"env"`
	Cmd           []string          `json:"cmd"`
	RestartPolicy string            `json:"restartPolicy"`
	Labels        map[string]string `json:"labels"`
	NanoCPUs      int64             `json:"nanoCPUs"`
	Memory        int64             `json:"memory"`
}

type PortBinding struct {
	Host      string `json:"host"`
	Container string `json:"container"`
	Protocol  string `json:"protocol"`
}

type VolumeBinding struct {
	Host      string `json:"host"`
	Container string `json:"container"`
}

type ContainerOperate struct {
	ContainerID string `json:"containerID" binding:"required"`
	Operation   string `json:"operation" binding:"required"`
}

type ContainerLog struct {
	ContainerID string `json:"containerID" binding:"required"`
	Since       string `json:"since"`
	Tail        string `json:"tail"`
}

// Image
type ImageInfo struct {
	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
	Size    int64    `json:"size"`
	Created int64    `json:"created"`
}

type ImagePull struct {
	ImageName string `json:"imageName" binding:"required"`
}

// Network
type NetworkInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Driver  string `json:"driver"`
	Subnet  string `json:"subnet"`
	Gateway string `json:"gateway"`
	Created string `json:"created"`
}

type NetworkCreate struct {
	Name    string `json:"name" binding:"required"`
	Driver  string `json:"driver"`
	Subnet  string `json:"subnet"`
	Gateway string `json:"gateway"`
}

// Volume
type VolumeInfo struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	MountPoint string `json:"mountPoint"`
	Created    string `json:"created"`
}

type VolumeCreate struct {
	Name   string `json:"name" binding:"required"`
	Driver string `json:"driver"`
}

// Compose
type ComposeInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Status  string `json:"status"`
	Created string `json:"created"`
}

type ComposeCreate struct {
	Name    string `json:"name" binding:"required"`
	Path    string `json:"path" binding:"required"`
	Content string `json:"content"`
}

type ComposeOperate struct {
	Name      string `json:"name" binding:"required"`
	Path      string `json:"path" binding:"required"`
	Operation string `json:"operation" binding:"required"`
}
