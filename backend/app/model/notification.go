package model

import "time"

type Notification struct {
	BaseModel
	Type      string     `gorm:"not null;index" json:"type"` // info / success / warning / error
	Title     string     `gorm:"not null" json:"title"`
	Content   string     `json:"content"`
	Source    string     `gorm:"index" json:"source"` // file / cronjob / system / security
	TargetURL string     `json:"targetUrl"`
	ReadAt    *time.Time `gorm:"index" json:"readAt,omitempty"`
}
