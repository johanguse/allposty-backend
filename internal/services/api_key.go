package services

// api_key.go — API key lifecycle management.
//
// Key format:  allposty_<43-char base64url>  (~52 chars total)
// Storage:     SHA-256 hex of the full key stored in DB.
//              Plaintext is returned once on creation and never stored.
// Display:     Prefix = "allposty_" + first 8 random chars (shown in key list).
//
// Scopes (v1):
//   *            full access
//   posts:read   GET /posts, GET /posts/calendar
//   posts:write  POST/DELETE /posts, POST /posts/:id/schedule
//   social:read  GET /social/accounts
//   media:read   GET /media
//   media:write  POST/DELETE /media
//   ai:write     POST /ai/caption

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/allposty/allposty-backend/internal/models"
	"github.com/allposty/allposty-backend/internal/repository"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

var (
	ErrAPIKeyNotFound = errors.New("api key not found")
	ErrAPIKeyRevoked  = errors.New("api key revoked")
	ErrAPIKeyExpired  = errors.New("api key expired")
)

// ValidScopes is the set of recognised scope strings.
var ValidScopes = map[string]bool{
	"*":           true,
	"posts:read":  true,
	"posts:write": true,
	"social:read": true,
	"media:read":  true,
	"media:write": true,
	"ai:write":    true,
}

// RateLimitPerPlan returns the requests-per-minute ceiling for a plan tier.
func RateLimitPerPlan(tier string) int {
	switch tier {
	case "pro":
		return 300
	case "agency":
		return 1000
	default: // free
		return 60
	}
}

type APIKeyService struct {
	keys *repository.APIKeyRepository
}

func NewAPIKeyService(keys *repository.APIKeyRepository) *APIKeyService {
	return &APIKeyService{keys: keys}
}

// CreateInput carries the parameters for key creation.
type CreateAPIKeyInput struct {
	UserID    uuid.UUID
	Name      string
	Scopes    []string
	ExpiresAt *time.Time
}

// CreateResult is returned once — plaintext is not stored.
type CreateAPIKeyResult struct {
	Key    *models.APIKey
	Plain  string // full key — show to user once, then discard
}

func (s *APIKeyService) Create(in CreateAPIKeyInput) (*CreateAPIKeyResult, error) {
	// 32 random bytes → 43-char base64url (no padding)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	random := base64.RawURLEncoding.EncodeToString(b)
	plain := "allposty_" + random

	h := sha256.Sum256([]byte(plain))
	hash := hex.EncodeToString(h[:])

	scopes := in.Scopes
	if len(scopes) == 0 {
		scopes = []string{"*"}
	}

	key := &models.APIKey{
		UserID:    in.UserID,
		Name:      in.Name,
		KeyHash:   hash,
		Prefix:    "allposty_" + random[:8],
		Scopes:    pq.StringArray(scopes),
		ExpiresAt: in.ExpiresAt,
	}

	if err := s.keys.Create(key); err != nil {
		return nil, err
	}
	return &CreateAPIKeyResult{Key: key, Plain: plain}, nil
}

func (s *APIKeyService) List(userID uuid.UUID) ([]models.APIKey, error) {
	return s.keys.FindByUser(userID)
}

func (s *APIKeyService) Revoke(keyID, userID uuid.UUID) error {
	key, err := s.keys.FindByID(keyID)
	if err != nil {
		return ErrAPIKeyNotFound
	}
	if key.UserID != userID {
		return ErrForbidden
	}
	return s.keys.Revoke(keyID)
}

// Authenticate validates a raw key string and returns the key record.
// The caller is responsible for rate-limit checks.
func (s *APIKeyService) Authenticate(plain string) (*models.APIKey, error) {
	h := sha256.Sum256([]byte(plain))
	hash := hex.EncodeToString(h[:])

	key, err := s.keys.FindByHash(hash)
	if err != nil {
		return nil, ErrAPIKeyNotFound
	}
	if key.Revoked {
		return nil, ErrAPIKeyRevoked
	}
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, ErrAPIKeyExpired
	}
	// Fire-and-forget last_used update — don't block the request.
	go func() { _ = s.keys.TouchLastUsed(key.ID) }()
	return key, nil
}

// HasScope reports whether the key grants the requested scope.
func HasScope(key *models.APIKey, required string) bool {
	for _, s := range key.Scopes {
		if s == "*" || s == required {
			return true
		}
	}
	return false
}
