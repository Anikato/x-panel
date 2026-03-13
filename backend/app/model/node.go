package model

type Node struct {
	BaseModel
	Name     string `gorm:"not null" json:"name"`
	Address  string `gorm:"not null" json:"address"` // IP:Port
	Token    string `json:"token"`
	Status   string `gorm:"default:offline" json:"status"` // online / offline
	GroupID  uint   `json:"groupID"`
	OS       string `json:"os"`
	Hostname string `json:"hostname"`
	CpuCores int    `json:"cpuCores"`
	MemTotal uint64 `json:"memTotal"`
}
