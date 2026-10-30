package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"xpanel/app/model"
	"xpanel/app/service"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDecodeInvokeRequestRejectsEmptyAndUnknown(t *testing.T) {
	if _, err := decodeInvokeRequest(strings.NewReader("")); err == nil {
		t.Fatal("empty stdin accepted")
	}
	if _, err := decodeInvokeRequest(strings.NewReader("{")); err == nil {
		t.Fatal("truncated json accepted")
	}
	req, err := decodeInvokeRequest(strings.NewReader(`{"capability":"ssl.renew"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp := dispatchInvoke(req, nil)
	if resp.OK || resp.Error != "unknown_capability" || resp.Capability != "ssl.renew" {
		t.Fatalf("%+v", resp)
	}
}

func TestDispatchSitesSnapshotRequiresDatabase(t *testing.T) {
	req, err := decodeInvokeRequest(bytes.NewReader([]byte(`{"capability":"sites.snapshot","payload":{}}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp := dispatchInvoke(req, nil)
	if resp.OK || resp.Error != "database_unreadable" {
		t.Fatalf("%+v", resp)
	}
}

func TestDispatchSitesSnapshotRejectsNonEmptyPayload(t *testing.T) {
	req, err := decodeInvokeRequest(strings.NewReader(`{"capability":"sites.snapshot","payload":{"foo":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	resp := dispatchInvoke(req, nil)
	if resp.OK || resp.Error != "invalid_request" {
		t.Fatalf("%+v", resp)
	}

	req, err = decodeInvokeRequest(strings.NewReader(`{"capability":"sites.snapshot"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp = dispatchInvoke(req, nil)
	if resp.OK || resp.Error != "database_unreadable" {
		t.Fatalf("omitted payload: %+v", resp)
	}
}

func TestDispatchHealthSnapshotRejectsNonEmptyPayloadAndRequiresDatabase(t *testing.T) {
	resp := dispatchInvoke(invokeRequest{Capability: "health.snapshot", Payload: json.RawMessage(`{"unexpected":true}`)}, nil)
	if resp.OK || resp.Error != "invalid_request" {
		t.Fatalf("non-empty payload: %+v", resp)
	}
	resp = dispatchInvoke(invokeRequest{Capability: "health.snapshot", Payload: json.RawMessage(`{ }`)}, nil)
	if resp.OK || resp.Error != "database_unreadable" {
		t.Fatalf("empty payload: %+v", resp)
	}
}

func TestDispatchHealthSnapshotReturnsStableDatabaseError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:healthinvoke-missing-schema?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	resp := dispatchInvoke(invokeRequest{Capability: "health.snapshot", Payload: json.RawMessage(`{}`)}, db)
	if resp.OK || resp.Error != "database_unreadable" {
		t.Fatalf("resp=%+v", resp)
	}
}

func TestDispatchHealthSnapshotUsesCollectorAndDoesNotLeakSecrets(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:healthinvoke?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.Website{},
		&model.Certificate{},
		&model.BackupRecord{},
		&model.Cronjob{},
		&model.CronjobRecord{},
	); err != nil {
		t.Fatal(err)
	}
	secret := "HEALTH-INVOKE-SECRET"
	if err := db.Create(&model.Certificate{PrimaryDomain: "secret.example", Pem: secret, PrivateKey: secret}).Error; err != nil {
		t.Fatal(err)
	}
	called := false
	resp := dispatchInvokeWithHealthCollector(invokeRequest{Capability: "health.snapshot", Payload: json.RawMessage(`{}`)}, db, func(data *service.HealthSnapshotData) {
		called = true
		data.Panel.Version = "v-test"
		data.Services = []service.HealthServiceSnapshot{{Name: "xpanel", Unit: "xpanel.service", ActiveState: "active", UnitFileState: "enabled"}}
	})
	if !resp.OK || resp.Error != "" || !called {
		t.Fatalf("resp=%+v called=%v", resp, called)
	}
	data, ok := resp.Data.(*service.HealthSnapshotData)
	if !ok || data.SchemaVersion != service.HealthSnapshotSchemaVersion || data.Panel.Version != "v-test" {
		t.Fatalf("data=%#v", resp.Data)
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte(secret)) || bytes.Contains(raw, []byte("PRIVATE KEY")) {
		t.Fatalf("secret leaked: %s", raw)
	}
}

func TestEncodeInvokeResponseIsJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := encodeInvokeResponse(&buf, invokeResponse{OK: true, Capability: "sites.snapshot"}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["ok"] != true || got["capability"] != "sites.snapshot" {
		t.Fatalf("%s", buf.String())
	}
}

func TestOpenInvokeDatabaseIsQueryOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "xpanel.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Website{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Website{PrimaryDomain: "a.example", Alias: "a"}).Error; err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatal(err)
	}

	ro, err := openInvokeDatabase(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ro.Create(&model.Website{PrimaryDomain: "b.example", Alias: "b"}).Error; err == nil {
		t.Fatal("create on query-only db succeeded")
	}
}

func TestRunInvokeWritesSnapshotJSON(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:runinvoke?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Website{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Website{PrimaryDomain: "site.example", Alias: "site"}).Error; err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	if err := runInvokeWith(nil, strings.NewReader(`{"capability":"sites.snapshot"}`), &stdout, db); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout=%s err=%v", stdout.String(), err)
	}
	if got["ok"] != true || got["capability"] != "sites.snapshot" {
		t.Fatalf("%s", stdout.String())
	}
	generatedAt, _ := got["generated_at"].(string)
	if generatedAt == "" {
		t.Fatal("generated_at empty")
	}
	if _, err := time.Parse(time.RFC3339, generatedAt); err != nil {
		t.Fatalf("generated_at not RFC3339: %q", generatedAt)
	}
	data, _ := got["data"].(map[string]any)
	if _, ok := data["sites"]; !ok {
		t.Fatalf("data.sites missing: %s", stdout.String())
	}

	stdout.Reset()
	err = runInvokeWith(nil, strings.NewReader(`{"capability":"ssl.renew"}`), &stdout, db)
	if err == nil {
		t.Fatal("unknown capability returned nil error")
	}
	var fail map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &fail); err != nil {
		t.Fatalf("stdout=%s err=%v", stdout.String(), err)
	}
	if fail["ok"] != false || fail["error"] != "unknown_capability" {
		t.Fatalf("%s", stdout.String())
	}

	stdout.Reset()
	err = runInvokeWith(nil, strings.NewReader("{"), &stdout, db)
	if err == nil {
		t.Fatal("invalid json returned nil error")
	}
	var invalid map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &invalid); err != nil {
		t.Fatalf("stdout=%s err=%v", stdout.String(), err)
	}
	if invalid["ok"] != false || invalid["error"] != "invalid_request" {
		t.Fatalf("%s", stdout.String())
	}
}

func TestRunInvokeWithExtraArgsReturnsUsageError(t *testing.T) {
	err := runInvokeWith([]string{"--json"}, strings.NewReader(`{"capability":"sites.snapshot"}`), os.Stdout, nil)
	if err == nil {
		t.Fatal("extra args accepted")
	}
}
