package dto

import "time"

type NodeCreate struct {
	Name        string `json:"name" binding:"required"`
	SSHHost     string `json:"sshHost" binding:"required"`
	SSHPort     uint   `json:"sshPort"`
	SSHUser     string `json:"sshUser" binding:"required"`
	SSHPassword string `json:"sshPassword" binding:"required"`
	PanelPort   string `json:"panelPort"`
	AgentToken  string `json:"agentToken"`
	GroupID     uint   `json:"groupID"`
}

type NodeUpdate struct {
	ID          uint   `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	SSHHost     string `json:"sshHost"`
	SSHPort     uint   `json:"sshPort"`
	SSHUser     string `json:"sshUser"`
	SSHPassword string `json:"sshPassword"`
	PanelPort   string `json:"panelPort"`
	AgentToken  string `json:"agentToken"`
	GroupID     uint   `json:"groupID"`
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
	SSHHost   string    `json:"sshHost"`
	SSHPort   uint      `json:"sshPort"`
	SSHUser   string    `json:"sshUser"`
}

type NodeSSHTest struct {
	SSHHost     string `json:"sshHost" binding:"required"`
	SSHPort     uint   `json:"sshPort"`
	SSHUser     string `json:"sshUser" binding:"required"`
	SSHPassword string `json:"sshPassword" binding:"required"`
}

type NodeInstallAgent struct {
	ID uint `json:"id" binding:"required"`
}

type NodeAgentAction struct {
	ID     uint   `json:"id" binding:"required"`
	Action string `json:"action" binding:"required"` // install / uninstall / update
}
