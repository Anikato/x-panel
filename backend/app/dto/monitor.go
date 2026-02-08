package dto

// SystemStats 系统实时状态
type SystemStats struct {
	Host       SystemHostInfo `json:"host"`
	CPU        CPUStats       `json:"cpu"`
	Memory     MemoryStats    `json:"memory"`
	Load       LoadStats      `json:"load"`
	Disks      []DiskStats    `json:"disks"`
	Network    NetworkStats   `json:"network"`
	NetIO      []NetIOStats   `json:"netIO"`
	TopProcess []ProcessBrief `json:"topProcess"`
	Uptime     uint64         `json:"uptime"` // seconds
}

// SystemHostInfo 系统基本信息
type SystemHostInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platformVersion"`
	KernelVersion   string `json:"kernelVersion"`
	KernelArch      string `json:"kernelArch"`
}

// CPUStats CPU 状态
type CPUStats struct {
	ModelName    string    `json:"modelName"`
	Cores        int       `json:"cores"`       // 物理核心
	LogicalCores int       `json:"logicalCores"` // 逻辑核心
	UsagePercent float64   `json:"usagePercent"`
	PerCPU       []float64 `json:"perCPU,omitempty"` // 每核使用率
}

// MemoryStats 内存状态
type MemoryStats struct {
	Total       uint64  `json:"total"`       // bytes
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"usedPercent"`
	SwapTotal   uint64  `json:"swapTotal"`
	SwapUsed    uint64  `json:"swapUsed"`
	SwapPercent float64 `json:"swapPercent"`
}

// LoadStats 系统负载
type LoadStats struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

// DiskStats 磁盘使用状态
type DiskStats struct {
	Device      string  `json:"device"`
	MountPoint  string  `json:"mountPoint"`
	FSType      string  `json:"fsType"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
	InodesTotal uint64  `json:"inodesTotal"`
	InodesUsed  uint64  `json:"inodesUsed"`
	InodesFree  uint64  `json:"inodesFree"`
	InodesPercent float64 `json:"inodesPercent"`
}

// NetworkStats 网络累计流量
type NetworkStats struct {
	BytesSent   uint64 `json:"bytesSent"`
	BytesRecv   uint64 `json:"bytesRecv"`
	PacketsSent uint64 `json:"packetsSent"`
	PacketsRecv uint64 `json:"packetsRecv"`
}

// NetIOStats 每个网卡的实时速率
type NetIOStats struct {
	Name      string  `json:"name"`
	BytesSent uint64  `json:"bytesSent"`
	BytesRecv uint64  `json:"bytesRecv"`
	SpeedUp   float64 `json:"speedUp"`   // bytes/s
	SpeedDown float64 `json:"speedDown"` // bytes/s
}

// ProcessBrief Top 进程简要信息
type ProcessBrief struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	CPUPercent float64 `json:"cpuPercent"`
	MemPercent float32 `json:"memPercent"`
	MemRSS     uint64  `json:"memRss"` // bytes
}
