package service

import (
	"os"
	"path/filepath"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
)

func TestBackupResolvesOnlyUnambiguousTargets(t *testing.T) {
	installDatabaseServiceDB(t)
	db := repo.NewIDatabaseRepo()
	var ids []uint
	for _, typ := range []string{"mysql", "postgresql", "mysql"} {
		server := model.DatabaseServer{Name: typ, Type: typ}
		if err := global.DB.Create(&server).Error; err != nil {
			t.Fatal(err)
		}
		instance := model.DatabaseInstance{ServerID: server.ID, Name: "app"}
		if err := global.DB.Create(&instance).Error; err != nil {
			t.Fatal(err)
		}
		ids = append(ids, instance.ID)
	}
	id, err := db.UniqueInstanceID("app", "postgresql")
	if err != nil || id != ids[1] {
		t.Fatalf("resolved %d: %v", id, err)
	}
	for _, req := range []dto.BackupCreate{
		{Type: "database", Name: "app", DBType: "mysql"},
		{Type: "database", Name: "missing", DBType: "mysql"},
		{Type: "database", Name: "app", DBType: "mysql", DBInstanceID: ids[1]},
		{Type: "database", Name: "other", DBInstanceID: ids[0]},
		{Type: "database", Name: "app", DBInstanceID: 999},
	} {
		if err := (&BackupService{dbRepo: db}).Backup(req); err == nil {
			t.Fatalf("queued wrong target %#v", req)
		}
	}
}

func TestDatabaseBackupPathsAreUnique(t *testing.T) {
	dir := t.TempDir()
	paths := map[string]bool{}
	for _, id := range []uint{1, 2, 1} {
		instance := model.DatabaseInstance{BaseModel: model.BaseModel{ID: id}, Name: "app"}
		path, err := reserveDatabaseBackup(dir, &instance, "postgresql", "20261118200000")
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 || filepath.Ext(path) != ".dump" || paths[path] {
			t.Fatalf("invalid reserved path %q (%v)", path, err)
		}
		paths[path] = true
	}
}
