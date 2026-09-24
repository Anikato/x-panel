package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
	"xpanel/security/credentials"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSecurityEntranceBlocksAPIUntilCookieIsPresented(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "entrance.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
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
	if err := repo.NewISettingRepo().CreateOrUpdate("SecurityEntrance", "secret-entry"); err != nil {
		t.Fatal(err)
	}

	engine := gin.New()
	engine.Use(SecurityEntrance())
	engine.POST("/api/v1/auth/login", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	engine.GET("/api/v1/version", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	engine.GET("/api/v1/cert-server/certs", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	engine.GET("/api/v1/settings", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	if rec := httptest.NewRecorder(); true {
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("login without entrance cookie = %d, want 404", rec.Code)
		}
	}
	version := httptest.NewRecorder()
	engine.ServeHTTP(version, httptest.NewRequest(http.MethodGet, "/api/v1/version", nil))
	if version.Code != http.StatusNoContent {
		t.Fatalf("version = %d, want 204", version.Code)
	}
	certs := httptest.NewRecorder()
	engine.ServeHTTP(certs, httptest.NewRequest(http.MethodGet, "/api/v1/cert-server/certs", nil))
	if certs.Code != http.StatusNoContent {
		t.Fatalf("cert server = %d, want 204", certs.Code)
	}

	authed := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	authed.AddCookie(&http.Cookie{Name: entranceCookieName, Value: "secret-entry"})
	settings := httptest.NewRecorder()
	engine.ServeHTTP(settings, authed)
	if settings.Code != http.StatusNoContent {
		t.Fatalf("settings with entrance cookie = %d, want 204", settings.Code)
	}
}

func TestCertServerIgnoresTokenInQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cert.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
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
	settings := repo.NewISettingRepo()
	if err := settings.CreateOrUpdate("CertServerEnabled", "enable"); err != nil {
		t.Fatal(err)
	}
	if err := settings.CreateOrUpdate("CertServerToken", "cert-token"); err != nil {
		t.Fatal(err)
	}

	engine := gin.New()
	engine.Use(CertServerAuth())
	engine.GET("/certs", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	query := httptest.NewRecorder()
	engine.ServeHTTP(query, httptest.NewRequest(http.MethodGet, "/certs?token=cert-token", nil))
	if query.Code != http.StatusUnauthorized {
		t.Fatalf("query token = %d, want 401", query.Code)
	}
	headerReq := httptest.NewRequest(http.MethodGet, "/certs", nil)
	headerReq.Header.Set("X-Cert-Token", "cert-token")
	header := httptest.NewRecorder()
	engine.ServeHTTP(header, headerReq)
	if header.Code != http.StatusNoContent {
		t.Fatalf("header token = %d, want 204", header.Code)
	}
}
