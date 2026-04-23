package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// APIKey represents a long-lived credential for programmatic API access.
//
// Key format:  allposty_<43-char base64url random>
// Stored:      SHA-256 hex of the full key — fast lookup, no bcrypt needed
//              because the random part is already 256 bits of entropy.
// Display:     Prefix field holds "allposty_XXXXXXXX" (first 8 random chars)
//              so users can identify keys without storing the plaintext.
type APIKey struct {
	Base
	UserID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Name       string         `gorm:"not null" json:"name"`
	KeyHash    string         `gorm:"uniqueIndex;not null" json:"-"`
	Prefix     string         `gorm:"not null" json:"prefix"` // allposty_XXXXXXXX
	Scopes     pq.StringArray `gorm:"type:text[]" json:"scopes"`
	LastUsedAt *time.Time     `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	Revoked    bool           `gorm:"default:false" json:"revoked"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}
