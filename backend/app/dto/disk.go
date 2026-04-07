package dto

// DiskInfo 物理磁盘信息
type DiskInfo struct {
	Device     string          `json:"device"`
	Model      string          `json:"model"`
	Size       uint64          `json:"size"` // bytes
	Type       string          `json:"type"` // HDD, SSD, unknown
	Partitions []PartitionInfo `json:"partitions"`
}

// PartitionInfo 分区信息
type PartitionInfo struct {
	Device      string  `json:"device"`
	MountPoint  string  `json:"mountPoint"`
	FSType      string  `json:"fsType"`
	Total       uint64  `json:"total"` // bytes
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
	InodesTotal uint64  `json:"inodesTotal"`
	InodesUsed  uint64  `json:"inodesUsed"`
	InodesFree  uint64  `json:"inodesFree"`
}

// RemoteMountRequest 远程挂载请求
type RemoteMountRequest struct {
	Protocol   string `json:"protocol" validate:"required,oneof=smb nfs cifs"`
	Server     string `json:"server" validate:"required"`
	SharePath  string `json:"sharePath" validate:"required"`
	MountPoint string `json:"mountPoint" validate:"required"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Options    string `json:"options"`
	Preset     string `json:"preset"`
	Persist    bool   `json:"persist"`
}

// RemoteUnmountRequest 卸载远程挂载
type RemoteUnmountRequest struct {
	MountPoint string `json:"mountPoint" validate:"required"`
	RemoveFstab bool  `json:"removeFstab"`
}

// RemoteMountInfo 远程挂载信息
type RemoteMountInfo struct {
	Device     string  `json:"device"`
	MountPoint string  `json:"mountPoint"`
	FSType     string  `json:"fsType"`
	Options    string  `json:"options"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Free       uint64  `json:"free"`
	Percent    float64 `json:"percent"`
	InFstab    bool    `json:"inFstab"`
}
