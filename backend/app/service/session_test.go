package service

import (
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"xpanel/app/model"
	"xpanel/global"
	"xpanel/utils/accessticket"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupSessionDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "session.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.PanelSession{}); err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = db
	ResetSessionsForTest()
	t.Cleanup(func() {
		global.DB = previous
		ResetSessionsForTest()
	})
}

func TestSessionValidDoesNotResurrectRevokedSessionFromDB(t *testing.T) {
	setupSessionDB(t)
	id := CreateSession("admin")
	if id == "" || !SessionValid(id) {
		t.Fatal("session should be valid after create")
	}
	if err := RevokeSession(id); err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Create(&model.PanelSession{ID: id, UserName: "admin"}).Error; err != nil {
		t.Fatal(err)
	}
	if SessionValid(id) {
		t.Fatal("revoked session was resurrected from database")
	}
	next := CreateSession("admin")
	if next == "" || !SessionValid(next) {
		t.Fatal("new session should be valid after revoke")
	}
	if SessionValid(id) {
		t.Fatal("revoked session became valid after creating a new one")
	}
}

func TestConcurrentRevokeMakesLaterValidationFail(t *testing.T) {
	setupSessionDB(t)
	id := CreateSession("admin")
	var validAfter atomic.Int32
	var wg sync.WaitGroup
	revoked := make(chan struct{})
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-revoked:
					if SessionValid(id) {
						validAfter.Add(1)
					}
					return
				default:
					_ = SessionValid(id)
				}
			}
		}()
	}
	if err := RevokeSession(id); err != nil {
		t.Fatal(err)
	}
	close(revoked)
	wg.Wait()
	if SessionValid(id) {
		t.Fatal("session still valid after revoke returned")
	}
	if validAfter.Load() != 0 {
		t.Fatalf("validation succeeded %d times after revoke returned", validAfter.Load())
	}
	next := CreateSession("admin")
	if !SessionValid(next) {
		t.Fatal("new session should be valid after concurrent revoke")
	}
	if SessionValid(id) {
		t.Fatal("revoked session became valid after creating a new one")
	}
}

func TestTicketAuthorizeAndSessionRevokeStayConsistent(t *testing.T) {
	setupSessionDB(t)
	id := CreateSession("admin")
	ticketID, err := accessticket.Default.Issue(accessticket.DownloadSpec("admin", id, "/tmp/a.txt"))
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for i := 0; i < 16; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					_, _ = accessticket.Default.Authorize(ticketID, http.MethodGet, "/api/v1/files/download", "/tmp/a.txt")
				}
			}()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = RevokeSession(id)
		}()
		wg.Wait()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock between ticket authorize and session revoke")
	}
	if SessionValid(id) {
		t.Fatal("session still valid after concurrent revoke")
	}
	if _, authErr := accessticket.Default.Authorize(ticketID, http.MethodGet, "/api/v1/files/download", "/tmp/a.txt"); authErr == nil {
		t.Fatal("ticket still authorized after session revoke")
	}

	next := CreateSession("admin")
	nextTicket, err := accessticket.Default.Issue(accessticket.DownloadSpec("admin", next, "/tmp/b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := accessticket.Default.Authorize(nextTicket, http.MethodGet, "/api/v1/files/download", "/tmp/b.txt"); err != nil {
		t.Fatalf("new session ticket should authorize: %v", err)
	}
	if SessionValid(id) {
		t.Fatal("revoked session became valid after issuing a new ticket")
	}
	if _, authErr := accessticket.Default.Authorize(ticketID, http.MethodGet, "/api/v1/files/download", "/tmp/a.txt"); authErr == nil {
		t.Fatal("old ticket became valid after creating a new session")
	}
}
