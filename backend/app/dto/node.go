package dto

import "time"

type NodeCreate struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address" binding:"required"`
	Token   string `json:"token" binding:"required"`
	GroupID uint   `json:"groupID"`
}

type NodeUpdate struct {
	ID      uint   `json:"id" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Address string `json:"address" binding:"required"`
	Token   string `json:"token"`
	GroupID uint   `json:"groupID"`
}

type NodeInfo struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Status    string    `json:"status"`
	GroupID   uint      `json:"groupID"`
	OS        string    `json:"os"`
	Hostname  string    `json:"hostname"`
	CpuCores  int       `json:"cpuCores"`
	MemTotal  uint64    `json:"memTotal"`
}
