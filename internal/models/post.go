package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type PostStatus string

const (
	PostStatusDraft     PostStatus = "draft"
	PostStatusScheduled PostStatus = "scheduled"
	PostStatusPublished PostStatus = "published"
	PostStatusFailed    PostStatus = "failed"
)

// Post is the canonical post record. Per-platform overrides live in PostPlatform.
type Post struct {
	Base
	WorkspaceID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"workspace_id"`
	Caption      string     `gorm:"type:text" json:"caption"`
	MediaURLs    pq.StringArray `gorm:"type:text[]" json:"media_urls"`
	Status       PostStatus `gorm:"not null;default:'draft'" json:"status"`
	ScheduledAt  *time.Time `json:"scheduled_at,omitempty"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	AsynqTaskID  *string    `json:"asynq_task_id,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`

	Workspace Workspace      `gorm:"foreignKey:WorkspaceID" json:"-"`
	Platforms []PostPlatform `gorm:"foreignKey:PostID" json:"platforms,omitempty"`
}

// PostPlatform holds platform-specific overrides and publish results.
type PostPlatform struct {
	Base
	PostID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"post_id"`
	SocialAccountID uuid.UUID  `gorm:"type:uuid;not null;index" json:"social_account_id"`
	Platform        Platform   `gorm:"not null" json:"platform"`
	CaptionOverride *string    `gorm:"type:text" json:"caption_override,omitempty"`
	Status          PostStatus `gorm:"not null;default:'draft'" json:"status"`
	PlatformPostID  *string    `json:"platform_post_id,omitempty"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	ErrorMessage    *string    `json:"error_message,omitempty"`

	Post          Post          `gorm:"foreignKey:PostID" json:"-"`
	SocialAccount SocialAccount `gorm:"foreignKey:SocialAccountID" json:"social_account,omitempty"`
}
