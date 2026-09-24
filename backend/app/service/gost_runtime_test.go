package service

import (
	"fmt"
	"sync"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/global"
	gostutil "xpanel/utils/gost"
)

type fakeGostRuntime struct {
	mu          sync.Mutex
	services    map[string]bool
	saveErr     error
	createFailN int
	creates     int
}

func newFakeGost(names ...string) *fakeGostRuntime {
	f := &fakeGostRuntime{services: map[string]bool{}}
	for _, name := range names {
		f.services[name] = true
	}
	return f
}

func (f *fakeGostRuntime) Ping() bool { return true }

func (f *fakeGostRuntime) CreateService(cfg gostutil.ServiceConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creates++
	if f.createFailN > 0 && f.creates >= f.createFailN {
		return fmt.Errorf("create failed")
	}
	f.services[cfg.Name] = true
	return nil
}

func (f *fakeGostRuntime) UpdateService(name string, cfg gostutil.ServiceConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.services[cfg.Name] = true
	return nil
}

func (f *fakeGostRuntime) DeleteService(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.services, name)
	return nil
}

func (f *fakeGostRuntime) SaveConfig() error {
	return f.saveErr
}

func (f *fakeGostRuntime) has(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.services[name]
}

func TestToggleRestoresRuntimeWhenSaveConfigFails(t *testing.T) {
	setupGostTestDB(t)
	original := model.GostService{
		Name: "fwd", Type: "tcp_forward", ListenAddr: ":8080",
		TargetAddr: "127.0.0.1:80", ListenerType: "tcp", Enabled: true,
	}
	if err := globalDBCreateGost(t, original); err != nil {
		t.Fatal(err)
	}
	fake := newFakeGost("fwd")
	fake.saveErr = fmt.Errorf("save failed")
	previous := newGostRuntime
	newGostRuntime = func() gostRuntime { return fake }
	t.Cleanup(func() { newGostRuntime = previous })

	storedID := gostID(t, "fwd")
	svc := NewIGostService().(*GostService)
	err := svc.ToggleService(dto.GostServiceToggle{ID: storedID, Enabled: false})
	if err == nil {
		t.Fatal("expected save failure")
	}
	var row model.GostService
	if dbErr := global.DB.First(&row, storedID).Error; dbErr != nil {
		t.Fatal(dbErr)
	}
	if !row.Enabled {
		t.Fatal("database enabled flag was not restored")
	}
	if !fake.has("fwd") {
		t.Fatal("runtime service was not restored after save failure")
	}
}

func TestUpdateCleansPartialCreateWhenLaterCreateFails(t *testing.T) {
	setupGostTestDB(t)
	original := model.GostService{
		Name: "fwd", Type: "tcp_udp_forward", ListenAddr: ":8080",
		TargetAddr: "127.0.0.1:80", ListenerType: "tcp", Enabled: true,
	}
	if err := globalDBCreateGost(t, original); err != nil {
		t.Fatal(err)
	}
	fake := newFakeGost("fwd-tcp", "fwd-udp")
	fake.createFailN = 2
	previous := newGostRuntime
	newGostRuntime = func() gostRuntime { return fake }
	t.Cleanup(func() { newGostRuntime = previous })

	storedID := gostID(t, "fwd")
	svc := NewIGostService().(*GostService)
	err := svc.UpdateService(dto.GostServiceUpdate{
		ID: storedID, Name: "fwd2", Type: "tcp_udp_forward",
		ListenAddr: ":9090", TargetAddr: "127.0.0.1:80", ListenerType: "tcp",
	})
	if err == nil {
		t.Fatal("expected create failure")
	}
	var row model.GostService
	if dbErr := global.DB.First(&row, storedID).Error; dbErr != nil {
		t.Fatal(dbErr)
	}
	if row.Name != "fwd" || row.ListenAddr != ":8080" {
		t.Fatalf("database rolled back incorrectly: %+v", row)
	}
	if fake.has("fwd2-tcp") || fake.has("fwd2-udp") {
		t.Fatal("partial new services were not cleaned")
	}
	if !fake.has("fwd-tcp") || !fake.has("fwd-udp") {
		t.Fatal("original runtime services were not restored")
	}
}

func globalDBCreateGost(t *testing.T, item model.GostService) error {
	t.Helper()
	return global.DB.Create(&item).Error
}

func gostID(t *testing.T, name string) uint {
	t.Helper()
	var row model.GostService
	if err := global.DB.Where("name = ?", name).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	return row.ID
}
