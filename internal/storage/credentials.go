package storage

// credentials.go — AES-256-GCM encryption for social media OAuth tokens.
// The app secret from config is used as the encryption key (hashed to 32 bytes).

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"

	"github.com/allposty/allposty-backend/internal/providers"
)

type CredentialStore struct {
	key []byte // 32-byte AES key derived from app secret
}

func NewCredentialStore(appSecret string) *CredentialStore {
	hash := sha256.Sum256([]byte(appSecret))
	return &CredentialStore{key: hash[:]}
}

// Encrypt serializes creds to JSON and encrypts with AES-256-GCM.
// Returns a base64-encoded ciphertext safe to store in DB.
func (s *CredentialStore) Encrypt(creds *providers.OAuthCredentials) (string, error) {
	plain, err := json.Marshal(creds)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plain, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decodes and decrypts the stored ciphertext back to OAuthCredentials.
func (s *CredentialStore) Decrypt(encoded string) (*providers.OAuthCredentials, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	var creds providers.OAuthCredentials
	if err := json.Unmarshal(plain, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}
