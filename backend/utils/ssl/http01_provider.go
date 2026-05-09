package ssl

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"xpanel/global"
)

const HTTP01ChallengePrefix = "/.well-known/acme-challenge/"

type HTTP01Provider struct{}

var http01Store = struct {
	sync.RWMutex
	tokens map[string]string
}{
	tokens: make(map[string]string),
}

func NewHTTP01Provider() *HTTP01Provider {
	return &HTTP01Provider{}
}

func (p *HTTP01Provider) Present(domain, token, keyAuth string) error {
	if token == "" || keyAuth == "" {
		return fmt.Errorf("invalid HTTP-01 challenge token")
	}
	if err := WriteHTTP01KeyAuth(token, keyAuth); err != nil {
		return err
	}
	http01Store.Lock()
	http01Store.tokens[token] = keyAuth
	http01Store.Unlock()
	return nil
}

func (p *HTTP01Provider) CleanUp(domain, token, keyAuth string) error {
	http01Store.Lock()
	delete(http01Store.tokens, token)
	http01Store.Unlock()
	_ = os.Remove(HTTP01ChallengeFile(token))
	return nil
}

func (p *HTTP01Provider) Timeout() (timeout, interval time.Duration) {
	return 2 * time.Minute, 2 * time.Second
}

func GetHTTP01KeyAuth(token string) (string, bool) {
	http01Store.RLock()
	keyAuth, ok := http01Store.tokens[token]
	http01Store.RUnlock()
	if ok {
		return keyAuth, true
	}
	data, err := os.ReadFile(HTTP01ChallengeFile(token))
	if err == nil {
		return string(data), true
	}
	return keyAuth, ok
}

func HTTP01WebRoot() string {
	dataDir := "."
	if global.CONF.System.DataDir != "" {
		dataDir = global.CONF.System.DataDir
	}
	return filepath.Join(dataDir, "acme-http")
}

func HTTP01ChallengeDir() string {
	return filepath.Join(HTTP01WebRoot(), ".well-known", "acme-challenge")
}

func HTTP01ChallengeFile(token string) string {
	return filepath.Join(HTTP01ChallengeDir(), filepath.Base(token))
}

func WriteHTTP01KeyAuth(token, keyAuth string) error {
	if err := os.MkdirAll(HTTP01ChallengeDir(), 0755); err != nil {
		return err
	}
	return os.WriteFile(HTTP01ChallengeFile(token), []byte(keyAuth), 0644)
}
