package service

import (
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestFindLocalCertAdoptsLegacySyncedCertificateWithoutSourceID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })
	if err := db.AutoMigrate(&model.Setting{}, &model.Certificate{}, &model.Website{}, &model.HAProxyLB{}, &model.GostService{}); err != nil {
		t.Fatal(err)
	}
	legacy := model.Certificate{
		PrimaryDomain: "example.com",
		Domains:       "www.example.com",
		Fingerprint:   "legacy-fingerprint",
		Type:          "synced",
		SourceType:    "synced",
		SourceID:      0,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}

	svc := NewICertSourceService().(*CertSourceService)
	cert, found, err := svc.findLocalCert(dto.CertServerItem{
		LineageUID:  "550e8400-e29b-41d4-a716-446655440000",
		Fingerprint: "legacy-fingerprint",
	}, 17)
	if err != nil {
		t.Fatal(err)
	}
	if !found || cert.ID != legacy.ID {
		t.Fatalf("expected legacy certificate %d to be adopted, got found=%t id=%d", legacy.ID, found, cert.ID)
	}
}
