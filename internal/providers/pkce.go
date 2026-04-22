package providers

// pkce.go — PKCE (Proof Key for Code Exchange) helpers.
// Used by Twitter/X OAuth 2.0 and any other provider that requires it.
// RFC 7636: https://www.rfc-editor.org/rfc/rfc7636

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// PKCEPair holds a code_verifier and its derived code_challenge (S256).
type PKCEPair struct {
	Verifier  string
	Challenge string
}

// NewPKCE generates a cryptographically random verifier and S256 challenge.
func NewPKCE() (*PKCEPair, error) {
	// Verifier: 32 random bytes → base64url (no padding) → 43 chars
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(b)

	// Challenge: SHA-256(verifier) → base64url
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	return &PKCEPair{Verifier: verifier, Challenge: challenge}, nil
}
