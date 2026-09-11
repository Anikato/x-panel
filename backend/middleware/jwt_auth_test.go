package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"xpanel/app/service"
	"xpanel/global"
	"xpanel/utils/accessticket"
	jwtUtil "xpanel/utils/jwt"

	"github.com/gin-gonic/gin"
)

func setupAuthTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previous := global.CONF
	global.CONF.System.JwtSecret = "ticket-test-secret"
	accessticket.Default = accessticket.NewStore()
	service.ResetSessionsForTest()
	t.Cleanup(func() {
		global.CONF = previous
		accessticket.Default = accessticket.NewStore()
		service.ResetSessionsForTest()
	})
}

func authEngine() *gin.Engine {
	r := gin.New()
	r.Use(JWTAuth())
	r.POST("/api/v1/auth/access-ticket", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/api/v1/terminal", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/api/v1/files/download", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.POST("/api/v1/files/download", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	r.GET("/api/v1/settings", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return r
}

func request(engine *gin.Engine, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, nil)
	engine.ServeHTTP(rec, req)
	return rec
}

func TestTerminalTicketCannotIssueNewTicket(t *testing.T) {
	setupAuthTest(t)
	sessionID := service.CreateSession("admin")
	id, err := accessticket.Default.Issue(accessticket.Spec{
		User: "admin", SessionID: sessionID, Method: http.MethodGet,
		Route: "/api/v1/terminal", Once: true, TTL: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := request(authEngine(), http.MethodPost, "/api/v1/auth/access-ticket?ticket="+id)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, ticket must not mint another ticket", rec.Code)
	}
}

func TestTerminalTicketCannotAccessSettings(t *testing.T) {
	setupAuthTest(t)
	sessionID := service.CreateSession("admin")
	id, _ := accessticket.Default.Issue(accessticket.Spec{
		User: "admin", SessionID: sessionID, Method: http.MethodGet,
		Route: "/api/v1/terminal", Once: true, TTL: time.Minute,
	})
	rec := request(authEngine(), http.MethodGet, "/api/v1/settings?ticket="+id)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestDownloadTicketRejectsWrongMethodAndRoute(t *testing.T) {
	setupAuthTest(t)
	sessionID := service.CreateSession("admin")
	id, _ := accessticket.Default.Issue(accessticket.Spec{
		User: "admin", SessionID: sessionID, Method: http.MethodGet,
		Route: "/api/v1/files/download", Resource: "/tmp/a.txt", TTL: time.Minute,
	})
	engine := authEngine()
	if rec := request(engine, http.MethodPost, "/api/v1/files/download?ticket="+id+"&path=/tmp/a.txt"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("POST download status=%d", rec.Code)
	}
	if rec := request(engine, http.MethodGet, "/api/v1/terminal?ticket="+id); rec.Code != http.StatusUnauthorized {
		t.Fatalf("terminal with download ticket status=%d", rec.Code)
	}
	if rec := request(engine, http.MethodGet, "/api/v1/files/download?ticket="+id+"&path=/tmp/other.txt"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong path status=%d", rec.Code)
	}
	if rec := request(engine, http.MethodGet, "/api/v1/files/download?ticket="+id+"&path=/tmp/a.txt"); rec.Code != http.StatusNoContent {
		t.Fatalf("matching download status=%d", rec.Code)
	}
}

func TestTerminalTicketIsConsumedOnHandshake(t *testing.T) {
	setupAuthTest(t)
	sessionID := service.CreateSession("admin")
	id, _ := accessticket.Default.Issue(accessticket.Spec{
		User: "admin", SessionID: sessionID, Method: http.MethodGet,
		Route: "/api/v1/terminal", Once: true, TTL: time.Minute,
	})
	engine := authEngine()
	if rec := request(engine, http.MethodGet, "/api/v1/terminal?ticket="+id); rec.Code != http.StatusNoContent {
		t.Fatalf("first handshake status=%d", rec.Code)
	}
	if rec := request(engine, http.MethodGet, "/api/v1/terminal?ticket="+id); rec.Code != http.StatusUnauthorized {
		t.Fatalf("reused terminal ticket status=%d", rec.Code)
	}
}

func TestTicketDiesWithSession(t *testing.T) {
	setupAuthTest(t)
	sessionID := service.CreateSession("admin")
	id, _ := accessticket.Default.Issue(accessticket.Spec{
		User: "admin", SessionID: sessionID, Method: http.MethodGet,
		Route: "/api/v1/files/download", Resource: "/tmp/a.txt", TTL: time.Minute,
	})
	if err := service.RevokeSession(sessionID); err != nil {
		t.Fatal(err)
	}
	rec := request(authEngine(), http.MethodGet, "/api/v1/files/download?ticket="+id+"&path=/tmp/a.txt")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d after session revoke", rec.Code)
	}
}

func TestLogoutCurrentSessionKeepsOtherSession(t *testing.T) {
	setupAuthTest(t)
	first := service.CreateSession("admin")
	second := service.CreateSession("admin")
	token1, err := jwtUtil.GenerateSessionToken("admin", first, 60)
	if err != nil {
		t.Fatal(err)
	}
	token2, err := jwtUtil.GenerateSessionToken("admin", second, 60)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RevokeSession(first); err != nil {
		t.Fatal(err)
	}
	engine := authEngine()
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	req1.Header.Set("Authorization", "Bearer "+token1)
	rec1 := httptest.NewRecorder()
	engine.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session status=%d", rec1.Code)
	}
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)
	rec2 := httptest.NewRecorder()
	engine.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNoContent {
		t.Fatalf("other session status=%d", rec2.Code)
	}
}

func TestJWTCanReachIssueTicketRoute(t *testing.T) {
	setupAuthTest(t)
	sessionID := service.CreateSession("admin")
	token, err := jwtUtil.GenerateSessionToken("admin", sessionID, 60)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/access-ticket", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	authEngine().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("jwt issue-ticket status=%d", rec.Code)
	}
}
