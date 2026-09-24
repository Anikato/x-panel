package service

import (
	"path/filepath"
	"sync"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
	"xpanel/security/credentials"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInitUserClaimsAdminOnlyOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "auth.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
		t.Fatal(err)
	}
	manager, _, err := credentials.LoadOrCreate(filepath.Join(t.TempDir(), "keyring.json"), true)
	if err != nil {
		t.Fatal(err)
	}
	previousDB, previousCredentials := global.DB, global.CREDENTIALS
	global.DB = db
	global.CREDENTIALS = manager
	t.Cleanup(func() {
		global.DB = previousDB
		global.CREDENTIALS = previousCredentials
	})
	if err := repo.NewISettingRepo().Create(&model.Setting{Key: "Password", Value: ""}); err != nil {
		t.Fatal(err)
	}
	if err := repo.NewISettingRepo().Create(&model.Setting{Key: "UserName", Value: "admin"}); err != nil {
		t.Fatal(err)
	}

	svc := NewIAuthService()
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			name := "admin-a"
			if i == 1 {
				name = "admin-b"
			}
			errs[i] = svc.InitUser(dto.InitUser{Name: name, Password: "secret-pass"})
		}(i)
	}
	close(start)
	wg.Wait()
	success := 0
	for _, initErr := range errs {
		if initErr == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("successful inits = %d, errors = %v", success, errs)
	}
	if err := svc.InitUser(dto.InitUser{Name: "other", Password: "secret-pass"}); err == nil {
		t.Fatal("second initialization should be rejected")
	}
	name, err := repo.NewISettingRepo().GetValueByKey("UserName")
	if err != nil {
		t.Fatal(err)
	}
	if name != "admin-a" && name != "admin-b" {
		t.Fatalf("username = %q", name)
	}
}
