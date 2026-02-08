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
