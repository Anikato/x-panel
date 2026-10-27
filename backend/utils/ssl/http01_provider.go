package ssl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	if dir := strings.TrimSpace(global.CONF.System.DataDir); dir != "" {
		dataDir = dir
	}
	dataDir = filepath.Clean(dataDir)
	parent := filepath.Dir(dataDir)
	// The data directory is mode 0700. Challenge files must sit beside it so
	// Nginx's worker can traverse to them without opening the private tree.
	if parent == dataDir {
		return filepath.Join(dataDir, "acme-http")
	}
	return filepath.Join(parent, "acme-http")
}

func HTTP01ChallengeDir() string {
	return filepath.Join(HTTP01WebRoot(), ".well-known", "acme-challenge")
}

func HTTP01ChallengeFile(token string) string {
	return filepath.Join(HTTP01ChallengeDir(), filepath.Base(token))
}

func WriteHTTP01KeyAuth(token, keyAuth string) error {
	dir := HTTP01ChallengeDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// The panel process uses umask 0077, so the mode passed to MkdirAll and
	// WriteFile is not what lands on disk. Nginx workers run as another user.
	if err := chmodHTTP01Directories(dir); err != nil {
		return err
	}
	path := HTTP01ChallengeFile(token)
	if err := os.WriteFile(path, []byte(keyAuth), 0644); err != nil {
		return err
	}
	return os.Chmod(path, 0644)
}

func chmodHTTP01Directories(dir string) error {
	root := filepath.Clean(HTTP01WebRoot())
	dir = filepath.Clean(dir)
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("http-01 directory %s is outside %s", dir, root)
	}
	current := root
	if err := os.Chmod(current, 0755); err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		if err := os.Chmod(current, 0755); err != nil {
			return err
		}
	}
	return nil
}
