package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
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
	if err := engine.SetTrustedProxies(nil); err != nil {
		t.Fatal(err)
	}
	var events []model.PanelAccessKind
	engine.Use(securityEntrance(func(ip string, kind model.PanelAccessKind) {
		if ip != "192.0.2.1" {
			t.Errorf("recorded IP = %q, want unproxied request IP", ip)
		}
		events = append(events, kind)
	}))
	engine.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	engine.POST("/api/v1/auth/login", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	engine.GET("/api/v1/version", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	engine.GET("/api/v1/cert-server/certs", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	engine.GET("/api/v1/settings", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	if rec := httptest.NewRecorder(); true {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		engine.ServeHTTP(rec, req)
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

	matched := httptest.NewRecorder()
	engine.ServeHTTP(matched, httptest.NewRequest(http.MethodGet, "/secret-entry?private=value", nil))
	if matched.Code != http.StatusTemporaryRedirect || len(matched.Result().Cookies()) == 0 {
		t.Fatalf("matched entrance did not redirect with a cookie")
	}
	cookie := matched.Result().Cookies()[0]

	authed := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	authed.AddCookie(cookie)
	settings := httptest.NewRecorder()
	engine.ServeHTTP(settings, authed)
	if settings.Code != http.StatusNoContent {
		t.Fatalf("settings with entrance cookie = %d, want 204", settings.Code)
	}
	root := httptest.NewRequest(http.MethodGet, "/", nil)
	root.AddCookie(cookie)
	engine.ServeHTTP(httptest.NewRecorder(), root)
	want := []model.PanelAccessKind{model.PanelAccessEntranceBlocked, model.PanelAccessEntranceMatched, model.PanelAccessPageOpened}
	if !slices.Equal(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
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
