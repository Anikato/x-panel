package model

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type oldNotification struct {
	ID        uint `gorm:"primarykey;autoIncrement"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Type      string `gorm:"not null;index"`
	Title     string `gorm:"not null"`
	Content   string
	Source    string `gorm:"index"`
	TargetURL string
	ReadAt    *time.Time `gorm:"index"`
}

func (oldNotification) TableName() string {
	return "notifications"
}

func TestNotificationAutoMigrateFromOldSchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&oldNotification{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&oldNotification{Type: "info", Title: "old"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Notification{}); err != nil {
		t.Fatal(err)
	}

	var item Notification
	if err := db.First(&item).Error; err != nil {
		t.Fatal(err)
	}
	if !item.ShowBadge {
		t.Fatal("expected show_badge to default true for existing rows")
	}
	if item.Popup {
		t.Fatal("expected popup to default false for existing rows")
	}
}
