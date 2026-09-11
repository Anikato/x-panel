package service

import (
	"path/filepath"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func setupHAProxyTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "haproxy.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.HAProxyLB{}, &model.HAProxyBackend{}, &model.HAProxyServer{}, &model.HAProxyACLRule{}, &model.HAProxyConfigVersion{}, &model.Setting{}); err != nil {
		t.Fatal(err)
	}
	previousDB, previousLog := global.DB, global.LOG
	global.DB = db
	global.LOG = logrus.New()
	t.Cleanup(func() {
		global.DB = previousDB
		global.LOG = previousLog
	})
}

func TestCreateLBDoesNotLeaveRowWhenRuntimeApplyFails(t *testing.T) {
	setupHAProxyTestDB(t)
	svc := NewIHAProxyService().(*HAProxyService)
	err := svc.CreateLB(dto.HAProxyLBCreate{
		Name:     "web",
		Mode:     "http",
		BindPort: 8080,
	}, "test")
	if err == nil {
		t.Fatal("expected runtime apply failure")
	}
	var count int64
	if dbErr := global.DB.Model(&model.HAProxyLB{}).Count(&count).Error; dbErr != nil {
		t.Fatal(dbErr)
	}
	if count != 0 {
		t.Fatalf("lb rows = %d, want 0", count)
	}
}

func TestUpdateLBKeepsOriginalRowWhenRuntimeApplyFails(t *testing.T) {
	setupHAProxyTestDB(t)
	original := model.HAProxyLB{Name: "web", Mode: "http", BindPort: 8080, BindAddr: "0.0.0.0", Enabled: true}
	if err := global.DB.Create(&original).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewIHAProxyService().(*HAProxyService)
	err := svc.UpdateLB(dto.HAProxyLBUpdate{
		ID: original.ID,
		HAProxyLBCreate: dto.HAProxyLBCreate{
			Name:     "web",
			Mode:     "http",
			BindPort: 9090,
		},
	}, "test")
	if err == nil {
		t.Fatal("expected runtime apply failure")
	}
	var stored model.HAProxyLB
	if dbErr := global.DB.First(&stored, original.ID).Error; dbErr != nil {
		t.Fatal(dbErr)
	}
	if stored.BindPort != 8080 {
		t.Fatalf("bind port = %d, want original 8080", stored.BindPort)
	}
}

func setupGostTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "gost.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.GostService{}, &model.GostChain{}, &model.Setting{}); err != nil {
		t.Fatal(err)
	}
	previousDB, previousLog := global.DB, global.LOG
	global.DB = db
	global.LOG = logrus.New()
	t.Cleanup(func() {
		global.DB = previousDB
		global.LOG = previousLog
	})
}

func TestUpdateGostServiceKeepsOriginalWhenRuntimeUnreachable(t *testing.T) {
	setupGostTestDB(t)
	original := model.GostService{
		Name:         "fwd",
		Type:         "tcp_forward",
		ListenAddr:   ":8080",
		TargetAddr:   "127.0.0.1:80",
		ListenerType: "tcp",
		Enabled:      true,
	}
	if err := global.DB.Create(&original).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewIGostService().(*GostService)
	err := svc.UpdateService(dto.GostServiceUpdate{
		ID:           original.ID,
		Name:         "fwd",
		Type:         "tcp_forward",
		ListenAddr:   ":9090",
		TargetAddr:   "127.0.0.1:80",
		ListenerType: "tcp",
	})
	if err == nil {
		t.Fatal("expected runtime failure")
	}
	var stored model.GostService
	if dbErr := global.DB.First(&stored, original.ID).Error; dbErr != nil {
		t.Fatal(dbErr)
	}
	if stored.ListenAddr != ":8080" {
		t.Fatalf("listen = %q, want original", stored.ListenAddr)
	}
}

func TestCreateBackendDoesNotLeaveRowWhenRuntimeApplyFails(t *testing.T) {
	setupHAProxyTestDB(t)
	svc := NewIHAProxyService().(*HAProxyService)
	err := svc.CreateBackend(dto.HAProxyBackendCreate{
		Name: "pool", Mode: "http", Balance: "roundrobin",
	}, "test")
	if err == nil {
		t.Fatal("expected runtime apply failure")
	}
	var count int64
	if dbErr := global.DB.Model(&model.HAProxyBackend{}).Count(&count).Error; dbErr != nil {
		t.Fatal(dbErr)
	}
	if count != 0 {
		t.Fatalf("backend rows = %d, want 0", count)
	}
}

func TestCreateServerDoesNotLeaveRowWhenRuntimeApplyFails(t *testing.T) {
	setupHAProxyTestDB(t)
	backend := model.HAProxyBackend{Name: "pool", Mode: "http", Balance: "roundrobin"}
	if err := global.DB.Create(&backend).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewIHAProxyService().(*HAProxyService)
	err := svc.CreateServer(dto.HAProxyServerCreate{
		BackendID: backend.ID, Name: "n1", Address: "127.0.0.1", Port: 80,
	}, "test")
	if err == nil {
		t.Fatal("expected runtime apply failure")
	}
	var count int64
	global.DB.Model(&model.HAProxyServer{}).Count(&count)
	if count != 0 {
		t.Fatalf("server rows = %d, want 0", count)
	}
}
