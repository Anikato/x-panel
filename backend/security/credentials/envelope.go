package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	encryptedMarker = "xpanel:enc:"
	envelopePrefix  = "xpanel:enc:v1:"
	keySize         = 32
)

type parsedEnvelope struct {
	KeyID      string
	WrappedDEK []byte
	Ciphertext []byte
}

func sealEnvelope(keyID string, kek []byte, scope, plaintext string) (string, error) {
	if keyID == "" {
		return "", errors.New("credential envelope key id is empty")
	}
	if scope == "" {
		return "", errors.New("credential envelope scope is empty")
	}
	if len(kek) != keySize {
		return "", fmt.Errorf("credential KEK must be %d bytes", keySize)
	}

	dek := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return "", fmt.Errorf("generate credential DEK: %w", err)
	}

	wrappedDEK, err := sealAEAD(kek, dek, []byte("xpanel:dek:v1:"+keyID))
	if err != nil {
		return "", fmt.Errorf("wrap credential DEK: %w", err)
	}
	ciphertext, err := sealAEAD(dek, []byte(plaintext), []byte("xpanel:data:v1:"+scope))
	if err != nil {
		return "", fmt.Errorf("encrypt credential: %w", err)
	}

	encoding := base64.RawURLEncoding
	return envelopePrefix + keyID + ":" +
		encoding.EncodeToString(wrappedDEK) + ":" +
		encoding.EncodeToString(ciphertext), nil
}

func openEnvelope(kek []byte, scope, encoded string) (string, error) {
	if scope == "" {
		return "", errors.New("credential envelope scope is empty")
	}
	if len(kek) != keySize {
		return "", fmt.Errorf("credential KEK must be %d bytes", keySize)
	}

	envelope, err := parseEnvelope(encoded)
	if err != nil {
		return "", err
	}
	dek, err := openAEAD(kek, envelope.WrappedDEK, []byte("xpanel:dek:v1:"+envelope.KeyID))
	if err != nil {
		return "", fmt.Errorf("unwrap credential DEK: %w", err)
	}
	if len(dek) != keySize {
		return "", fmt.Errorf("credential DEK must be %d bytes", keySize)
	}
	plaintext, err := openAEAD(dek, envelope.Ciphertext, []byte("xpanel:data:v1:"+scope))
	if err != nil {
		return "", fmt.Errorf("decrypt credential: %w", err)
	}
	return string(plaintext), nil
}

func parseEnvelope(encoded string) (parsedEnvelope, error) {
	if !strings.HasPrefix(encoded, envelopePrefix) {
		return parsedEnvelope{}, errors.New("unsupported or malformed credential envelope")
	}
	parts := strings.Split(encoded, ":")
	if len(parts) != 6 || parts[0] != "xpanel" || parts[1] != "enc" || parts[2] != "v1" {
		return parsedEnvelope{}, errors.New("malformed credential envelope")
	}
	if parts[3] == "" {
		return parsedEnvelope{}, errors.New("credential envelope key id is empty")
	}

	encoding := base64.RawURLEncoding
	wrappedDEK, err := encoding.DecodeString(parts[4])
	if err != nil {
		return parsedEnvelope{}, fmt.Errorf("decode wrapped credential DEK: %w", err)
	}
	ciphertext, err := encoding.DecodeString(parts[5])
	if err != nil {
		return parsedEnvelope{}, fmt.Errorf("decode credential ciphertext: %w", err)
	}
	if err := validateAEADPayload(wrappedDEK); err != nil {
		return parsedEnvelope{}, fmt.Errorf("wrapped credential DEK: %w", err)
	}
	if err := validateAEADPayload(ciphertext); err != nil {
		return parsedEnvelope{}, fmt.Errorf("credential ciphertext: %w", err)
	}

	return parsedEnvelope{
		KeyID:      parts[3],
		WrappedDEK: wrappedDEK,
		Ciphertext: ciphertext,
	}, nil
}

func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encryptedMarker)
}

func EnvelopeKeyID(value string) (string, error) {
	envelope, err := parseEnvelope(value)
	if err != nil {
		return "", err
	}
	return envelope.KeyID, nil
}

func sealAEAD(key, plaintext, additionalData []byte) ([]byte, error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return aead.Seal(nonce, nonce, plaintext, additionalData), nil
}

func openAEAD(key, payload, additionalData []byte) ([]byte, error) {
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(payload) < aead.NonceSize()+aead.Overhead() {
		return nil, errors.New("AEAD payload is too short")
	}
	nonce := payload[:aead.NonceSize()]
	ciphertext := payload[aead.NonceSize():]
	return aead.Open(nil, nonce, ciphertext, additionalData)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != keySize {
		return nil, fmt.Errorf("AES-256 key must be %d bytes", keySize)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func validateAEADPayload(payload []byte) error {
	aead, err := newGCM(make([]byte, keySize))
	if err != nil {
		return err
	}
	if len(payload) < aead.NonceSize()+aead.Overhead() {
		return errors.New("AEAD payload is too short")
	}
	return nil
}
