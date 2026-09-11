package model

import "time"

type PanelSession struct {
	ID        string    `gorm:"primaryKey;size:64" json:"id"`
	UserName  string    `gorm:"index;not null" json:"userName"`
	CreatedAt time.Time `json:"createdAt"`
}
