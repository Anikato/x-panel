package credentials

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const keyringVersion = 1

type keyringDocument struct {
	Version     int               `json:"version"`
	ActiveKeyID string            `json:"activeKeyId"`
	Keys        map[string]string `json:"keys"`
}

type Manager struct {
	mu   sync.RWMutex
	path string
	doc  keyringDocument
}

func LoadOrCreate(path string, allowCreate bool) (*Manager, bool, error) {
	path = filepath.Clean(path)
	if path == "." || filepath.Base(path) == "." {
		return nil, false, errors.New("credential keyring path is invalid")
	}
	if err := ensureKeyringDirectory(filepath.Dir(path)); err != nil {
		return nil, false, err
	}

	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return nil, false, fmt.Errorf("credential keyring must be a regular file: %s", path)
		}
		if err := os.Chmod(path, 0600); err != nil {
			return nil, false, fmt.Errorf("harden credential keyring %s: %w", path, err)
		}
		doc, err := loadKeyringDocument(path)
		if err != nil {
			return nil, false, err
		}
		return &Manager{path: path, doc: doc}, false, nil
	}
	if !os.IsNotExist(err) {
		return nil, false, fmt.Errorf("inspect credential keyring %s: %w", path, err)
	}
	if !allowCreate {
		return nil, false, fmt.Errorf("credential keyring does not exist: %s", path)
	}

	doc, err := newKeyringDocument()
	if err != nil {
		return nil, false, err
	}
	if err := writeNewKeyring(path, doc); err != nil {
		if errors.Is(err, os.ErrExist) {
			loaded, loadErr := loadKeyringDocument(path)
			if loadErr != nil {
				return nil, false, loadErr
			}
			return &Manager{path: path, doc: loaded}, false, nil
		}
		return nil, false, err
	}
	return &Manager{path: path, doc: doc}, true, nil
}

func (m *Manager) Protect(scope, value string) (string, error) {
	if value == "" {
		return "", nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	plaintext := value
	if IsEncrypted(value) {
		keyID, err := EnvelopeKeyID(value)
		if err != nil {
			return "", err
		}
		key, err := decodedKey(m.doc, keyID)
		if err != nil {
			return "", err
		}
		plaintext, err = openEnvelope(key, scope, value)
		if err != nil {
			return "", err
		}
		if keyID == m.doc.ActiveKeyID {
			return value, nil
		}
	}

	activeKey, err := decodedKey(m.doc, m.doc.ActiveKeyID)
	if err != nil {
		return "", err
	}
	return sealEnvelope(m.doc.ActiveKeyID, activeKey, scope, plaintext)
}

func (m *Manager) Reveal(scope, value string) (string, error) {
	if value == "" || !IsEncrypted(value) {
		return value, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	keyID, err := EnvelopeKeyID(value)
	if err != nil {
		return "", err
	}
	key, err := decodedKey(m.doc, keyID)
	if err != nil {
		return "", err
	}
	return openEnvelope(key, scope, value)
}

func (m *Manager) Validate(scope, value string) error {
	if value == "" {
		return nil
	}
	if !IsEncrypted(value) {
		return errors.New("credential value is not encrypted")
	}
	_, err := m.Reveal(scope, value)
	return err
}

func (m *Manager) IsEncrypted(value string) bool {
	return IsEncrypted(value)
}

func (m *Manager) ActiveKeyID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.doc.ActiveKeyID
}

func (m *Manager) AddActiveKey() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	next := cloneKeyringDocument(m.doc)
	var (
		keyID string
		key   []byte
		err   error
	)
	for {
		keyID, key, err = generateKEK()
		if err != nil {
			return "", err
		}
		if _, exists := next.Keys[keyID]; !exists {
			break
		}
	}
	next.Keys[keyID] = base64.RawStdEncoding.EncodeToString(key)
	next.ActiveKeyID = keyID

	if err := persistKeyring(m.path, next); err != nil {
		return "", err
	}
	m.doc = next
	return keyID, nil
}

func (m *Manager) KeyIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.doc.Keys))
	for keyID := range m.doc.Keys {
		ids = append(ids, keyID)
	}
	sort.Strings(ids)
	return ids
}

func newKeyringDocument() (keyringDocument, error) {
	keyID, key, err := generateKEK()
	if err != nil {
		return keyringDocument{}, err
	}
	return keyringDocument{
		Version:     keyringVersion,
		ActiveKeyID: keyID,
		Keys: map[string]string{
			keyID: base64.RawStdEncoding.EncodeToString(key),
		},
	}, nil
}

func generateKEK() (string, []byte, error) {
	idBytes := make([]byte, 8)
	if _, err := io.ReadFull(rand.Reader, idBytes); err != nil {
		return "", nil, fmt.Errorf("generate credential key id: %w", err)
	}
	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", nil, fmt.Errorf("generate credential KEK: %w", err)
	}
	return "kek-" + hex.EncodeToString(idBytes), key, nil
}

func ensureKeyringDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create credential keyring directory %s: %w", dir, err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("inspect credential keyring directory %s: %w", dir, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("credential keyring directory must be a real directory: %s", dir)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("harden credential keyring directory %s: %w", dir, err)
	}
	return nil
}

func loadKeyringDocument(path string) (keyringDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return keyringDocument{}, fmt.Errorf("read credential keyring %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var doc keyringDocument
	if err := decoder.Decode(&doc); err != nil {
		return keyringDocument{}, fmt.Errorf("decode credential keyring %s: %w", path, err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return keyringDocument{}, fmt.Errorf("decode credential keyring %s: %w", path, err)
	}
	if err := validateKeyringDocument(doc); err != nil {
		return keyringDocument{}, fmt.Errorf("validate credential keyring %s: %w", path, err)
	}
	return doc, nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); errors.Is(err, io.EOF) {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("credential keyring contains trailing JSON")
}

func validateKeyringDocument(doc keyringDocument) error {
	if doc.Version != keyringVersion {
		return fmt.Errorf("unsupported keyring version %d", doc.Version)
	}
	if doc.ActiveKeyID == "" {
		return errors.New("active key id is empty")
	}
	if len(doc.Keys) == 0 {
		return errors.New("keyring has no keys")
	}
	for keyID := range doc.Keys {
		if keyID == "" {
			return errors.New("keyring contains an empty key id")
		}
		if _, err := decodedKey(doc, keyID); err != nil {
			return err
		}
	}
	if _, ok := doc.Keys[doc.ActiveKeyID]; !ok {
		return fmt.Errorf("active key %q is missing", doc.ActiveKeyID)
	}
	return nil
}

func decodedKey(doc keyringDocument, keyID string) ([]byte, error) {
	encoded, ok := doc.Keys[keyID]
	if !ok {
		return nil, fmt.Errorf("credential KEK %q is unavailable", keyID)
	}
	key, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode credential KEK %q: %w", keyID, err)
	}
	if len(key) != keySize {
		return nil, fmt.Errorf("credential KEK %q must be %d bytes", keyID, keySize)
	}
	return key, nil
}

func writeNewKeyring(path string, doc keyringDocument) error {
	data, err := marshalKeyring(doc)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create credential keyring %s: %w", path, err)
	}
	cleanup := true
	defer func() {
		_ = file.Close()
		if cleanup {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write credential keyring %s: %w", path, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync credential keyring %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close credential keyring %s: %w", path, err)
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func persistKeyring(path string, doc keyringDocument) error {
	if err := validateKeyringDocument(doc); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect credential keyring %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("credential keyring must be a regular file: %s", path)
	}
	data, err := marshalKeyring(doc)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".credential-keyring-*")
	if err != nil {
		return fmt.Errorf("create credential keyring temp file: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return fmt.Errorf("harden credential keyring temp file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write credential keyring temp file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync credential keyring temp file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close credential keyring temp file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace credential keyring %s: %w", path, err)
	}
	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return err
	}
	return nil
}

func marshalKeyring(doc keyringDocument) ([]byte, error) {
	if err := validateKeyringDocument(doc); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode credential keyring: %w", err)
	}
	return append(data, '\n'), nil
}

func syncDirectory(dir string) error {
	file, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open credential keyring directory %s: %w", dir, err)
	}
	defer file.Close()
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync credential keyring directory %s: %w", dir, err)
	}
	return nil
}

func cloneKeyringDocument(doc keyringDocument) keyringDocument {
	keys := make(map[string]string, len(doc.Keys)+1)
	for keyID, key := range doc.Keys {
		keys[keyID] = key
	}
	return keyringDocument{
		Version:     doc.Version,
		ActiveKeyID: doc.ActiveKeyID,
		Keys:        keys,
	}
}
