package service

import (
	"fmt"
	"sync"

	"xpanel/app/dto"
	"xpanel/global"
	"xpanel/security/credentials"

	"gorm.io/gorm"
)

type ICredentialSecurityService interface {
	RotateCredentialKey() (*dto.CredentialRotationInfo, error)
}

type credentialReencryptFunc func(*gorm.DB, global.CredentialProtector) error

type CredentialSecurityService struct {
	db        *gorm.DB
	protector global.CredentialProtector
	reencrypt credentialReencryptFunc
	mu        *sync.Mutex
}

var credentialRotationMu sync.Mutex

func NewICredentialSecurityService() ICredentialSecurityService {
	return newCredentialSecurityService(
		global.DB,
		global.CREDENTIALS,
		credentials.ReencryptDatabase,
	)
}

func newCredentialSecurityService(
	db *gorm.DB,
	protector global.CredentialProtector,
	reencrypt credentialReencryptFunc,
) *CredentialSecurityService {
	return &CredentialSecurityService{
		db:        db,
		protector: protector,
		reencrypt: reencrypt,
		mu:        &credentialRotationMu,
	}
}

func (s *CredentialSecurityService) RotateCredentialKey() (*dto.CredentialRotationInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil, fmt.Errorf("credential rotation database is unavailable")
	}
	if s.protector == nil {
		return nil, fmt.Errorf("credential protector is unavailable")
	}
	if s.reencrypt == nil {
		return nil, fmt.Errorf("credential re-encryption function is unavailable")
	}

	keyID, err := s.protector.AddActiveKey()
	if err != nil {
		return nil, fmt.Errorf("activate new credential key: %w", err)
	}
	if err := s.reencrypt(s.db, s.protector); err != nil {
		return nil, fmt.Errorf("re-encrypt database with credential key %s: %w", keyID, err)
	}

	state, err := credentials.ScanDatabase(s.db)
	if err != nil {
		return nil, fmt.Errorf("verify credential key rotation %s: %w", keyID, err)
	}
	if state.HasPlaintext {
		return nil, fmt.Errorf("verify credential key rotation %s: plaintext remains", keyID)
	}
	for usedKeyID := range state.KeyIDs {
		if usedKeyID != keyID {
			return nil, fmt.Errorf(
				"verify credential key rotation %s: database still references key %s",
				keyID,
				usedKeyID,
			)
		}
	}
	return &dto.CredentialRotationInfo{KeyID: keyID}, nil
}
