package model

type DatabaseServer struct {
	BaseModel
	Name     string `gorm:"not null" json:"name"`
	Type     string `gorm:"not null" json:"type"` // mysql / postgresql
	From     string `gorm:"default:local" json:"from"` // local / remote
	Address  string `json:"address"`
	Port     uint   `gorm:"default:3306" json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type DatabaseInstance struct {
	BaseModel
	ServerID uint   `gorm:"index" json:"serverID"`
	Name     string `gorm:"not null" json:"name"`
	Charset  string `gorm:"default:utf8mb4" json:"charset"`
	Owner    string `json:"owner"`
}
