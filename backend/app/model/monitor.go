package model

// MonitorBase CPU/内存/负载 基础监控数据
type MonitorBase struct {
	BaseModel
	Cpu       float64 `json:"cpu"`
	Memory    float64 `json:"memory"`
	LoadUsage float64 `json:"loadUsage"`
	CpuLoad1  float64 `json:"cpuLoad1"`
	CpuLoad5  float64 `json:"cpuLoad5"`
	CpuLoad15 float64 `json:"cpuLoad15"`
}

// MonitorIO 磁盘 IO 监控数据
type MonitorIO struct {
	BaseModel
	Name  string `json:"name"`
	Read  uint64 `json:"read"`  // bytes/s
	Write uint64 `json:"write"` // bytes/s
}

// MonitorNetwork 网络 IO 监控数据
type MonitorNetwork struct {
	BaseModel
	Name string  `json:"name"`
	Up   float64 `json:"up"`   // KB/s
	Down float64 `json:"down"` // KB/s
}
