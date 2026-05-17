package model

import "time"

type Notification struct {
	BaseModel
	Type      string     `gorm:"not null;index" json:"type"` // info / success / warning / error
	Event     string     `gorm:"index" json:"event"`         // file.upload.completed / cronjob.failed / system.log.error
	Title     string     `gorm:"not null" json:"title"`
	Content   string     `json:"content"`
	Source    string     `gorm:"index" json:"source"` // file / cronjob / system / security
	TargetURL string     `json:"targetUrl"`
	ShowBadge bool       `gorm:"not null" json:"showBadge"`
	Popup     bool       `gorm:"not null" json:"popup"`
	ReadAt    *time.Time `gorm:"index" json:"readAt,omitempty"`
}
