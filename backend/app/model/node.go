package model

type Node struct {
	BaseModel
	Name     string `gorm:"not null" json:"name"`
	Address  string `gorm:"not null" json:"address"` // IP:Port for X-Panel API
	Token    string `json:"token"`
	Status   string `gorm:"default:offline" json:"status"` // online / offline
	GroupID  uint   `json:"groupID"`
	OS       string `json:"os"`
	Hostname string `json:"hostname"`
	CpuCores int    `json:"cpuCores"`
	MemTotal uint64 `json:"memTotal"`
	// SSH credentials for remote agent management
	SSHHost     string `json:"sshHost"`
	SSHPort     uint   `gorm:"default:22" json:"sshPort"`
	SSHUser     string `json:"sshUser"`
	SSHPassword string `json:"sshPassword"`
}
