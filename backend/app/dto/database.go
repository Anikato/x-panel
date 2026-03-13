package dto

import "time"

// Database Server
type DatabaseServerCreate struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	From     string `json:"from" binding:"required"`
	Address  string `json:"address"`
	Port     uint   `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type DatabaseServerUpdate struct {
	ID       uint   `json:"id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Address  string `json:"address"`
	Port     uint   `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type DatabaseServerInfo struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	From      string    `json:"from"`
	Address   string    `json:"address"`
	Port      uint      `json:"port"`
	Username  string    `json:"username"`
	Status    string    `json:"status"`
}

type DatabaseServerSearch struct {
	PageInfo
	Type string `json:"type"`
	Info string `json:"info"`
}

// Database Instance
type DatabaseInstanceCreate struct {
	ServerID uint   `json:"serverID" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Charset  string `json:"charset"`
	Owner    string `json:"owner"`
	Password string `json:"password"`
}

type DatabaseInstanceInfo struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	ServerID  uint      `json:"serverID"`
	Name      string    `json:"name"`
	Charset   string    `json:"charset"`
	Owner     string    `json:"owner"`
}

type DatabaseInstanceSearch struct {
	PageInfo
	ServerID uint   `json:"serverID" binding:"required"`
	Info     string `json:"info"`
}

type DatabaseInstanceChangePassword struct {
	ID       uint   `json:"id" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type DatabaseInstanceBackup struct {
	ID uint `json:"id" binding:"required"`
}

type DatabaseInstanceRestore struct {
	ID   uint   `json:"id" binding:"required"`
	File string `json:"file" binding:"required"`
}
