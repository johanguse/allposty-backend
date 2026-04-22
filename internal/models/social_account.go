package models

import "github.com/google/uuid"

type Platform string

const (
	PlatformInstagram Platform = "instagram"
	PlatformFacebook  Platform = "facebook"
	PlatformLinkedIn  Platform = "linkedin"
	PlatformTwitter   Platform = "twitter"
	PlatformTikTok    Platform = "tiktok"
	PlatformYouTube   Platform = "youtube"
)

// SocialAccount stores a connected social media account for a workspace.
// Credentials are stored encrypted in CredentialsEnc.
type SocialAccount struct {
	Base
	WorkspaceID    uuid.UUID `gorm:"type:uuid;not null;index" json:"workspace_id"`
	Platform       Platform  `gorm:"not null" json:"platform"`
	PlatformUserID string    `gorm:"not null" json:"platform_user_id"`
	Name           string    `gorm:"not null" json:"name"`
	Username       string    `json:"username"`
	AvatarURL      *string   `json:"avatar_url,omitempty"`
	CredentialsEnc string    `gorm:"not null" json:"-"` // AES-256-GCM encrypted JSON

	Workspace Workspace `gorm:"foreignKey:WorkspaceID" json:"-"`
}
