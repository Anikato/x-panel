package model

import "time"

// ComposeTask records the latest checkpoint for a Compose operation.
type ComposeTask struct {
	ID        string `gorm:"primaryKey"`
	Name      string `gorm:"not null"`
	Status    string `gorm:"not null;index"`
	Message   string
	Log       string
	Phase     string
	Progress  int
	StartedAt time.Time `gorm:"not null;index"`
	EndedAt   *time.Time
}
