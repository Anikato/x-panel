package model

// Host SSH 主机
type Host struct {
	BaseModel
	GroupID     uint   `gorm:"default:0" json:"groupID"`
	Name        string `gorm:"not null" json:"name"`
	Addr        string `gorm:"not null" json:"addr"`
	Port        int    `gorm:"not null;default:22" json:"port"`
	User        string `gorm:"not null;default:root" json:"user"`
	AuthMode    string `gorm:"not null;default:password" json:"authMode"` // password | key
	Password    string `json:"-"`
	PrivateKey  string `json:"-"`
	PassPhrase  string `json:"-"`
	Description string `json:"description"`
}

// Command 快速命令
type Command struct {
	BaseModel
	GroupID uint   `gorm:"default:0" json:"groupID"`
	Name    string `gorm:"not null" json:"name"`
	Command string `gorm:"not null" json:"command"`
}

// Group 通用分组
type Group struct {
	BaseModel
	Name string `gorm:"not null" json:"name"`
	Type string `gorm:"not null" json:"type"` // host | command
}
