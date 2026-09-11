package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"xpanel/app/model"
	"xpanel/global"
	"xpanel/utils/accessticket"
)

type sessionStore struct {
	mu   sync.Mutex
	live map[string]string
}

var sessions = newSessionStore()

func newSessionStore() *sessionStore {
	return &sessionStore{live: make(map[string]string)}
}

func ResetSessionsForTest() {
	sessions = newSessionStore()
	accessticket.Default.SetSessionChecker(SessionValid)
}

func CreateSession(userName string) string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return ""
	}
	id := hex.EncodeToString(raw[:])
	if global.DB != nil {
		if err := global.DB.Create(&model.PanelSession{ID: id, UserName: userName}).Error; err != nil {
			return ""
		}
	}
	sessions.mu.Lock()
	sessions.live[id] = userName
	sessions.mu.Unlock()
	return id
}

func SessionValid(id string) bool {
	if id == "" {
		return false
	}
	sessions.mu.Lock()
	_, ok := sessions.live[id]
	sessions.mu.Unlock()
	return ok
}

func RevokeSession(id string) error {
	if id == "" {
		return fmt.Errorf("missing session")
	}
	sessions.mu.Lock()
	delete(sessions.live, id)
	sessions.mu.Unlock()
	accessticket.Default.RevokeSession(id)
	if global.DB != nil {
		if err := global.DB.Delete(&model.PanelSession{}, "id = ?", id).Error; err != nil {
			return err
		}
	}
	return nil
}

func RevokeOtherSessions(userName, keepID string) error {
	revoked := map[string]struct{}{}
	sessions.mu.Lock()
	for id, user := range sessions.live {
		if user == userName && id != keepID {
			delete(sessions.live, id)
			revoked[id] = struct{}{}
		}
	}
	sessions.mu.Unlock()
	accessticket.Default.RevokeSessions(revoked)
	if global.DB != nil {
		if err := global.DB.Where("user_name = ? AND id <> ?", userName, keepID).Delete(&model.PanelSession{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func RevokeUserSessions(userName string) error {
	revoked := map[string]struct{}{}
	sessions.mu.Lock()
	for id, user := range sessions.live {
		if user == userName {
			delete(sessions.live, id)
			revoked[id] = struct{}{}
		}
	}
	sessions.mu.Unlock()
	accessticket.Default.RevokeSessions(revoked)
	if global.DB != nil {
		if err := global.DB.Where("user_name = ?", userName).Delete(&model.PanelSession{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func LoadSessionsFromDB() {
	if global.DB == nil {
		return
	}
	var rows []model.PanelSession
	if err := global.DB.Find(&rows).Error; err != nil {
		return
	}
	sessions.mu.Lock()
	sessions.live = make(map[string]string, len(rows))
	for _, row := range rows {
		sessions.live[row.ID] = row.UserName
	}
	sessions.mu.Unlock()
}

func init() {
	accessticket.Default.SetSessionChecker(SessionValid)
}
