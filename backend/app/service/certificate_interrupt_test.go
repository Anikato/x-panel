package service

import (
	"path/filepath"
	"testing"

	"xpanel/app/model"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMarkInterruptedCertificateApplications(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cert.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Certificate{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Certificate{PrimaryDomain: "stuck.example.com", Provider: "dns", Status: "applying", KeyType: "P256"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Certificate{PrimaryDomain: "ok.example.com", Provider: "dns", Status: "applied", KeyType: "P256"}).Error; err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	MarkInterruptedCertificateApplications()

	var stuck, ok model.Certificate
	if err := db.Where("primary_domain = ?", "stuck.example.com").First(&stuck).Error; err != nil {
		t.Fatal(err)
	}
	if stuck.Status != "error" || stuck.Message == "" {
		t.Fatalf("stuck certificate = %#v", stuck)
	}
	if err := db.Where("primary_domain = ?", "ok.example.com").First(&ok).Error; err != nil {
		t.Fatal(err)
	}
	if ok.Status != "applied" {
		t.Fatalf("applied certificate changed to %s", ok.Status)
	}
}
