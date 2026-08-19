package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"xpanel/app/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openSnapDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Website{}, &model.Certificate{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestBuildSitesSnapshotMapsSourcesAndHidesSecrets(t *testing.T) {
	db := openSnapDB(t)
	expire := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	localExpire := time.Date(2026, 11, 2, 8, 0, 0, 0, shanghai)

	if err := db.Create(&model.Certificate{
		BaseModel:     model.BaseModel{ID: 9},
		PrimaryDomain: "example.com",
		Provider:      "manual",
		Type:          "synced",
		SourceType:    "synced",
		SourceName:    "master-panel",
		ExpireDate:    expire,
		Pem:           "-----BEGIN CERTIFICATE-----\nSECRET\n",
		PrivateKey:    "-----BEGIN PRIVATE KEY-----\nSECRET\n",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Certificate{
		BaseModel:     model.BaseModel{ID: 10},
		PrimaryDomain: "upload.example.com",
		Provider:      "manual",
		Type:          "upload",
		SourceType:    "upload",
		ExpireDate:    expire,
		Pem:           "-----BEGIN CERTIFICATE-----\nUPLOAD\n",
		PrivateKey:    "-----BEGIN PRIVATE KEY-----\nUPLOAD\n",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Certificate{
		BaseModel:     model.BaseModel{ID: 11},
		PrimaryDomain: "acme.example.com",
		Provider:      "dns",
		Type:          "autoApply",
		SourceType:    "acme",
		ExpireDate:    expire,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Certificate{
		BaseModel:     model.BaseModel{ID: 12},
		PrimaryDomain: "http.example.com",
		Provider:      "http",
		Type:          "autoApply",
		ExpireDate:    expire,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Certificate{
		BaseModel:     model.BaseModel{ID: 13},
		PrimaryDomain: "zero.example.com",
		Provider:      "dns",
		Type:          "autoApply",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Certificate{
		BaseModel:     model.BaseModel{ID: 14},
		PrimaryDomain: "tz.example.com",
		Provider:      "dns",
		Type:          "autoApply",
		ExpireDate:    localExpire,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&model.Website{
		PrimaryDomain: "example.com",
		Domains:       "www.example.com",
		Alias:         "blog",
		Type:          "reverse_proxy",
		Status:        "running",
		SSLEnable:     true,
		CertificateID: 9,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Website{
		PrimaryDomain: "plain.example.com",
		Alias:         "plain",
		Type:          "static",
		Status:        "stopped",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Website{
		PrimaryDomain: "upload.example.com",
		Alias:         "upload-site",
		Type:          "static",
		Status:        "running",
		SSLEnable:     true,
		CertificateID: 10,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Website{
		PrimaryDomain: "acme.example.com",
		Alias:         "acme-site",
		Type:          "static",
		Status:        "running",
		SSLEnable:     true,
		CertificateID: 11,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Website{
		PrimaryDomain: "http.example.com",
		Alias:         "http-site",
		Type:          "static",
		Status:        "running",
		SSLEnable:     true,
		CertificateID: 12,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Website{
		PrimaryDomain: "offssl.example.com",
		Alias:         "offssl",
		Type:          "static",
		Status:        "running",
		SSLEnable:     false,
		CertificateID: 9,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Website{
		PrimaryDomain: "missing.example.com",
		Alias:         "missing",
		Type:          "static",
		Status:        "running",
		SSLEnable:     true,
		CertificateID: 999,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Website{
		PrimaryDomain: "dupe.example.com",
		Domains:       "dupe.example.com, www.dupe.example.com, ,dupe.example.com",
		Alias:         "dupe",
		Type:          "static",
		Status:        "stopped",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Website{
		PrimaryDomain: "zero.example.com",
		Alias:         "zero",
		Type:          "static",
		Status:        "running",
		SSLEnable:     true,
		CertificateID: 13,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.Website{
		PrimaryDomain: "tz.example.com",
		Alias:         "tz",
		Type:          "static",
		Status:        "running",
		SSLEnable:     true,
		CertificateID: 14,
	}).Error; err != nil {
		t.Fatal(err)
	}

	data, err := BuildSitesSnapshot(db)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(data)
	if bytes.Contains(raw, []byte("BEGIN CERTIFICATE")) || bytes.Contains(raw, []byte("PRIVATE KEY")) {
		t.Fatalf("secrets leaked: %s", raw)
	}
	if bytes.Contains(raw, []byte(`"pem"`)) || bytes.Contains(raw, []byte(`"privateKey"`)) || bytes.Contains(raw, []byte(`"private_key"`)) {
		t.Fatalf("secret fields present: %s", raw)
	}
	if data.Truncated || len(data.Sites) != 10 {
		t.Fatalf("%+v", data)
	}

	synced := data.Sites[0]
	if synced.SourceType != "synced" || synced.SourceName != "master-panel" || synced.ExpireAt == nil {
		t.Fatalf("%+v", synced)
	}
	if synced.Alias != "blog" || synced.PrimaryDomain != "example.com" || synced.Type != "reverse_proxy" || synced.Status != "running" || !synced.SSLEnable {
		t.Fatalf("%+v", synced)
	}
	if synced.ExpireAt.Location() != time.UTC || !synced.ExpireAt.Equal(expire) {
		t.Fatalf("expire_at=%v loc=%v", synced.ExpireAt, synced.ExpireAt.Location())
	}
	if len(synced.Domains) != 2 || synced.Domains[0] != "example.com" || synced.Domains[1] != "www.example.com" {
		t.Fatalf("domains=%v", synced.Domains)
	}

	plain := data.Sites[1]
	if plain.SourceType != "none" || plain.ExpireAt != nil || plain.SourceName != "" {
		t.Fatalf("%+v", plain)
	}

	if data.Sites[2].SourceType != "upload" {
		t.Fatalf("upload=%+v", data.Sites[2])
	}
	if data.Sites[3].SourceType != "acme" {
		t.Fatalf("dns/autoApply=%+v", data.Sites[3])
	}
	if data.Sites[4].SourceType != "acme" {
		t.Fatalf("http/autoApply=%+v", data.Sites[4])
	}
	if data.Sites[5].SourceType != "none" || data.Sites[5].ExpireAt != nil {
		t.Fatalf("sslEnable=false still bound: %+v", data.Sites[5])
	}
	if data.Sites[6].SourceType != "none" || data.Sites[6].ExpireAt != nil {
		t.Fatalf("missing cert: %+v", data.Sites[6])
	}
	if got := data.Sites[7].Domains; len(got) != 2 || got[0] != "dupe.example.com" || got[1] != "www.dupe.example.com" {
		t.Fatalf("deduped domains=%v", got)
	}
	if data.Sites[8].SourceType != "acme" || data.Sites[8].ExpireAt != nil {
		t.Fatalf("zero expire: %+v", data.Sites[8])
	}
	tzSite := data.Sites[9]
	if tzSite.ExpireAt == nil || tzSite.ExpireAt.Location() != time.UTC || !tzSite.ExpireAt.Equal(localExpire.UTC()) {
		t.Fatalf("tz expire_at=%v", tzSite.ExpireAt)
	}
}

func TestBuildSitesSnapshotTruncatesAt500(t *testing.T) {
	db := openSnapDB(t)
	sites := make([]model.Website, 501)
	for i := range sites {
		sites[i] = model.Website{
			PrimaryDomain: fmt.Sprintf("site%d.example.com", i),
			Alias:         fmt.Sprintf("site%d", i),
			Type:          "static",
			Status:        "stopped",
		}
	}
	if err := db.Create(&sites).Error; err != nil {
		t.Fatal(err)
	}
	data, err := BuildSitesSnapshot(db)
	if err != nil {
		t.Fatal(err)
	}
	if !data.Truncated || len(data.Sites) != 500 {
		t.Fatalf("truncated=%v len=%d", data.Truncated, len(data.Sites))
	}
}

func TestMapCertificateSourceType(t *testing.T) {
	tests := []struct {
		name  string
		cert  *model.Certificate
		bound bool
		want  string
	}{
		{name: "unbound", bound: false, want: "none"},
		{name: "unbound ignores cert", cert: &model.Certificate{Type: "synced", SourceType: "synced"}, bound: false, want: "none"},
		{name: "type synced", cert: &model.Certificate{Type: "synced"}, bound: true, want: "synced"},
		{name: "source_type synced", cert: &model.Certificate{SourceType: "synced"}, bound: true, want: "synced"},
		{name: "type upload", cert: &model.Certificate{Type: "upload"}, bound: true, want: "upload"},
		{name: "source_type upload", cert: &model.Certificate{SourceType: "upload"}, bound: true, want: "upload"},
		{name: "autoApply", cert: &model.Certificate{Type: "autoApply"}, bound: true, want: "acme"},
		{name: "dns", cert: &model.Certificate{Type: "autoApply", Provider: "dns", SourceType: "acme"}, bound: true, want: "acme"},
		{name: "http", cert: &model.Certificate{Type: "autoApply", Provider: "http"}, bound: true, want: "acme"},
		{name: "nil cert bound", cert: nil, bound: true, want: "none"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapCertificateSourceType(tt.cert, tt.bound); got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
