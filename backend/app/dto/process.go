package dto

// ProcessInfo 进程信息
type ProcessInfo struct {
	PID        int32   `json:"pid"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	Username   string  `json:"username"`
	CPUPercent float64 `json:"cpuPercent"`
	MemPercent float32 `json:"memPercent"`
	MemRSS     uint64  `json:"memRSS"` // bytes
	StartTime  int64   `json:"startTime"`
	NumThreads int32   `json:"numThreads"`
	CmdLine    string  `json:"cmdLine"`
	PPID       int32   `json:"ppid"`
}

// ProcessSearchReq 进程搜索请求
type ProcessSearchReq struct {
	PID      int32  `json:"pid"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Status   string `json:"status"` // running, sleeping, stopped, zombie
	SortBy   string `json:"sortBy"` // cpu, mem, pid, name
	SortDesc bool   `json:"sortDesc"`
}

// ProcessStopReq 停止进程请求
type ProcessStopReq struct {
	PID    int32  `json:"pid" binding:"required"`
	Signal string `json:"signal"` // kill, term, stop
}

// NetworkConnInfo 网络连接信息
type NetworkConnInfo struct {
	PID        int32  `json:"pid"`
	Name       string `json:"name"`
	LocalAddr  string `json:"localAddr"`
	LocalPort  uint32 `json:"localPort"`
	RemoteAddr string `json:"remoteAddr"`
	RemotePort uint32 `json:"remotePort"`
	Status     string `json:"status"`
	Protocol   string `json:"protocol"` // tcp, tcp6, udp, udp6
}
