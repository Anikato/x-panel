package service

import (
	"path/filepath"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
	"xpanel/security/credentials"

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

func TestDeleteLBKeepsRowWhenRuntimeApplyFails(t *testing.T) {
	setupHAProxyTestDB(t)
	lb := model.HAProxyLB{Name: "web", Mode: "http", BindPort: 8080, BindAddr: "0.0.0.0", Enabled: true}
	if err := global.DB.Create(&lb).Error; err != nil {
		t.Fatal(err)
	}
	acl := model.HAProxyACLRule{LBID: lb.ID, MatchType: "host", MatchValue: "example.com", TargetBackendID: 1, Enabled: true}
	if err := global.DB.Create(&acl).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewIHAProxyService().(*HAProxyService)
	if err := svc.DeleteLB(lb.ID, "test"); err == nil {
		t.Fatal("expected runtime apply failure")
	}
	var lbCount, aclCount int64
	global.DB.Model(&model.HAProxyLB{}).Count(&lbCount)
	global.DB.Model(&model.HAProxyACLRule{}).Count(&aclCount)
	if lbCount != 1 || aclCount != 1 {
		t.Fatalf("lb=%d acl=%d, want both restored", lbCount, aclCount)
	}
}

func TestDeleteBackendAndServerKeepRowsWhenRuntimeApplyFails(t *testing.T) {
	setupHAProxyTestDB(t)
	backend := model.HAProxyBackend{Name: "pool", Mode: "http", Balance: "roundrobin"}
	if err := global.DB.Create(&backend).Error; err != nil {
		t.Fatal(err)
	}
	server := model.HAProxyServer{BackendID: backend.ID, Name: "n1", Address: "127.0.0.1", Port: 80}
	if err := global.DB.Create(&server).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewIHAProxyService().(*HAProxyService)
	if err := svc.DeleteServer(server.ID, "test"); err == nil {
		t.Fatal("expected server delete failure")
	}
	if err := svc.DeleteBackend(backend.ID, "test"); err == nil {
		t.Fatal("expected backend delete failure")
	}
	var backends, servers int64
	global.DB.Model(&model.HAProxyBackend{}).Count(&backends)
	global.DB.Model(&model.HAProxyServer{}).Count(&servers)
	if backends != 1 || servers != 1 {
		t.Fatalf("backends=%d servers=%d, want both restored", backends, servers)
	}
}

func useTestCredentials(t *testing.T) {
	t.Helper()
	manager, _, err := credentials.LoadOrCreate(filepath.Join(t.TempDir(), "credential-keyring.json"), true)
	if err != nil {
		t.Fatal(err)
	}
	previous := global.CREDENTIALS
	global.CREDENTIALS = manager
	t.Cleanup(func() { global.CREDENTIALS = previous })
}

func TestCreateChainDoesNotLeaveRowWhenRuntimeUnreachable(t *testing.T) {
	setupGostTestDB(t)
	useTestCredentials(t)
	svc := NewIGostService().(*GostService)
	err := svc.CreateChain(dto.GostChainCreate{Name: "chain", Hops: `[{"name":"hop","addr":"127.0.0.1:1080"}]`})
	if err == nil {
		t.Fatal("expected runtime failure")
	}
	var count int64
	global.DB.Model(&model.GostChain{}).Count(&count)
	if count != 0 {
		t.Fatalf("chain rows = %d, want 0", count)
	}
}

func TestUpdateAndDeleteChainKeepRecordWhenRuntimeUnreachable(t *testing.T) {
	setupGostTestDB(t)
	useTestCredentials(t)
	original := model.GostChain{Name: "chain", Hops: `[{"name":"hop","addr":"127.0.0.1:1080"}]`}
	if err := global.DB.Create(&original).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewIGostService().(*GostService)
	err := svc.UpdateChain(dto.GostChainUpdate{
		ID: original.ID, Name: "chain", Hops: `[{"name":"hop","addr":"127.0.0.1:1081"}]`,
	})
	if err == nil {
		t.Fatal("expected update failure")
	}
	stored, dbErr := repo.NewIGostChainRepo().Get(repo.WithByID(original.ID))
	if dbErr != nil {
		t.Fatal(dbErr)
	}
	if stored.Hops != original.Hops {
		t.Fatalf("hops = %s, want original", stored.Hops)
	}
	if err := svc.DeleteChain(original.ID); err == nil {
		t.Fatal("expected delete failure")
	}
	if _, dbErr = repo.NewIGostChainRepo().Get(repo.WithByID(original.ID)); dbErr != nil {
		t.Fatalf("chain deleted despite runtime failure: %v", dbErr)
	}
}

func TestDeleteGostServiceKeepsRowWhenRuntimeUnreachable(t *testing.T) {
	setupGostTestDB(t)
	original := model.GostService{
		Name: "fwd", Type: "tcp_forward", ListenAddr: ":8080",
		TargetAddr: "127.0.0.1:80", ListenerType: "tcp", Enabled: true,
	}
	if err := global.DB.Create(&original).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewIGostService().(*GostService)
	if err := svc.DeleteService(original.ID); err == nil {
		t.Fatal("expected runtime failure")
	}
	var count int64
	global.DB.Model(&model.GostService{}).Count(&count)
	if count != 1 {
		t.Fatalf("service rows = %d, want 1", count)
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
