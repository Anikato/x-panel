package migration

import (
	"testing"

	"xpanel/app/model"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMigrateCertificateLineageAndPause(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })

	if err := db.AutoMigrate(&model.Setting{}, &model.Certificate{}, &model.CertSource{}); err != nil {
		t.Fatal(err)
	}
	cert := model.Certificate{PrimaryDomain: "example.com", Provider: "manual", Type: "upload", KeyType: "2048"}
	syncedCert := model.Certificate{PrimaryDomain: "sync.example.com", Provider: "manual", Type: "synced", SourceType: "synced", KeyType: "2048"}
	source := model.CertSource{Name: "primary", ServerAddr: "https://upstream", Token: "token", Enabled: true}
	if err := db.Create(&cert).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&syncedCert).Error; err != nil {
		t.Fatal(err)
	}

	if err := migrateCertificateLineageAndPause(); err != nil {
		t.Fatal(err)
	}

	var gotCert model.Certificate
	var gotSyncedCert model.Certificate
	var gotSource model.CertSource
	if err := db.First(&gotCert, cert.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&gotSource, source.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&gotSyncedCert, syncedCert.ID).Error; err != nil {
		t.Fatal(err)
	}
	if gotCert.LineageUID == "" {
		t.Fatal("expected existing certificate to receive a lineage UID")
	}
	if gotSyncedCert.LineageUID != "" {
		t.Fatal("historical synced certificate must wait for upstream lineage adoption")
	}
	if gotSource.Enabled {
		t.Fatal("expected existing source to be disabled")
	}
	if !gotSource.ResumeRequired {
		t.Fatal("expected existing source to require manual resume")
	}

	if err := db.Model(&gotSource).Update("enabled", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateCertificateLineageAndPause(); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&gotSource, source.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !gotSource.Enabled {
		t.Fatal("migration must run only once")
	}
}
