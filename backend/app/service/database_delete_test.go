package service

import (
	"errors"
	"testing"

	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func installDatabaseServiceDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.DatabaseServer{}, &model.DatabaseInstance{}); err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
}

func TestDeleteInstanceKeepsRecordWhenServerLookupFails(t *testing.T) {
	installDatabaseServiceDB(t)
	instance := model.DatabaseInstance{ServerID: 99, Name: "appdb", Username: "app"}
	if err := global.DB.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}

	err := NewIDatabaseService().DeleteInstance(instance.ID)
	if err == nil {
		t.Fatal("expected server lookup error")
	}
	var stored model.DatabaseInstance
	if dbErr := global.DB.First(&stored, instance.ID).Error; dbErr != nil {
		t.Fatalf("panel record removed: %v", dbErr)
	}
}

func TestDeleteInstanceKeepsRecordWhenDropFails(t *testing.T) {
	installDatabaseServiceDB(t)
	server := model.DatabaseServer{Name: "local", Type: "mysql"}
	if err := global.DB.Create(&server).Error; err != nil {
		t.Fatal(err)
	}
	instance := model.DatabaseInstance{ServerID: server.ID, Name: "appdb", Username: "app"}
	if err := global.DB.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}

	svc := &DatabaseService{
		repo: repo.NewIDatabaseRepo(),
		dropRemote: func(*model.DatabaseServer, *model.DatabaseInstance) error {
			return errors.New("drop failed")
		},
	}
	if err := svc.DeleteInstance(instance.ID); err == nil {
		t.Fatal("expected drop error")
	}
	var stored model.DatabaseInstance
	if err := global.DB.First(&stored, instance.ID).Error; err != nil {
		t.Fatalf("panel record removed: %v", err)
	}
}

func TestDeleteInstanceRemovesRecordAfterSuccessfulDrop(t *testing.T) {
	installDatabaseServiceDB(t)
	server := model.DatabaseServer{Name: "local", Type: "mysql"}
	if err := global.DB.Create(&server).Error; err != nil {
		t.Fatal(err)
	}
	instance := model.DatabaseInstance{ServerID: server.ID, Name: "appdb", Username: "app"}
	if err := global.DB.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}

	svc := &DatabaseService{
		repo: repo.NewIDatabaseRepo(),
		dropRemote: func(*model.DatabaseServer, *model.DatabaseInstance) error {
			return nil
		},
	}
	if err := svc.DeleteInstance(instance.ID); err != nil {
		t.Fatal(err)
	}
	if err := global.DB.First(&model.DatabaseInstance{}, instance.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("panel record still present: %v", err)
	}
}

func TestDropRemoteInstanceDoesNotDeleteUserAfterDatabaseFailure(t *testing.T) {
	fake := &fakeDBAdmin{deleteDBErr: errors.New("cannot drop")}
	err := dropRemoteInstance(&model.DatabaseServer{Type: "mysql"}, &model.DatabaseInstance{
		Name:     "appdb",
		Username: "app",
	}, nil, fake)
	if err == nil {
		t.Fatal("expected drop error")
	}
	if fake.deletedUser != "" {
		t.Fatalf("user deleted after drop failure: %s", fake.deletedUser)
	}
}

func TestDropRemoteInstanceReportsPartialFailureWhenUserCleanupFails(t *testing.T) {
	fake := &fakeDBAdmin{deleteUserErr: errors.New("cannot drop user")}
	err := dropRemoteInstance(&model.DatabaseServer{Type: "mysql"}, &model.DatabaseInstance{
		Name:     "appdb",
		Username: "app",
	}, nil, fake)
	if err == nil {
		t.Fatal("expected partial failure")
	}
	if fake.deletedDB != "appdb" {
		t.Fatalf("database not dropped: %s", fake.deletedDB)
	}
	if !errors.Is(err, fake.deleteUserErr) {
		t.Fatalf("error = %v", err)
	}
}

func TestDropRemoteInstanceSkipsSharedUsername(t *testing.T) {
	fake := &fakeDBAdmin{}
	instance := model.DatabaseInstance{BaseModel: model.BaseModel{ID: 1}, Name: "appdb", Username: "shared"}
	others := []model.DatabaseInstance{
		{BaseModel: model.BaseModel{ID: 2}, Name: "otherdb", Username: "shared"},
	}
	if err := dropRemoteInstance(&model.DatabaseServer{Type: "mysql"}, &instance, others, fake); err != nil {
		t.Fatal(err)
	}
	if fake.deletedDB != "appdb" {
		t.Fatalf("database not dropped: %s", fake.deletedDB)
	}
	if fake.deletedUser != "" {
		t.Fatalf("shared user deleted: %s", fake.deletedUser)
	}
}

type fakeDBAdmin struct {
	deleteDBErr   error
	deleteUserErr error
	deletedDB     string
	deletedUser   string
}

func (f *fakeDBAdmin) DeleteDatabase(name string) error {
	if f.deleteDBErr != nil {
		return f.deleteDBErr
	}
	f.deletedDB = name
	return nil
}

func (f *fakeDBAdmin) DeleteUser(username, _ string) error {
	if f.deleteUserErr != nil {
		return f.deleteUserErr
	}
	f.deletedUser = username
	return nil
}

func (f *fakeDBAdmin) Close() {}
