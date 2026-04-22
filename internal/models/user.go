package models

import "github.com/google/uuid"

type User struct {
	Base
	Email         string  `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash  string  `gorm:"not null" json:"-"`
	Name          string  `gorm:"not null" json:"name"`
	AvatarURL     *string `json:"avatar_url,omitempty"`
	EmailVerified bool    `gorm:"default:false" json:"email_verified"`
	GoogleID      *string `gorm:"uniqueIndex" json:"-"`

	// Plan limits (denormalized for fast enforcement)
	PlanTier string `gorm:"default:'free'" json:"plan_tier"` // free | pro | agency

	// Relations
	Memberships   []WorkspaceMember `gorm:"foreignKey:UserID" json:"-"`
	RefreshTokens []RefreshToken    `gorm:"foreignKey:UserID" json:"-"`
}

type RefreshToken struct {
	Base
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"-"`
	Token     string    `gorm:"uniqueIndex;not null" json:"-"`
	ExpiresAt int64     `gorm:"not null" json:"-"`
	Revoked   bool      `gorm:"default:false" json:"-"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
}
