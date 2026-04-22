package models

import "github.com/google/uuid"

// Organization is the top-level tenant. Users can own or belong to many orgs.
type Organization struct {
	Base
	Name      string  `gorm:"not null" json:"name"`
	Slug      string  `gorm:"uniqueIndex;not null" json:"slug"`
	LogoURL   *string `json:"logo_url,omitempty"`
	OwnerID   uuid.UUID `gorm:"type:uuid;not null;index" json:"owner_id"`

	Owner      User        `gorm:"foreignKey:OwnerID" json:"-"`
	Workspaces []Workspace `gorm:"foreignKey:OrganizationID" json:"workspaces,omitempty"`
}

// Workspace lives inside an Organization. Social accounts and posts belong here.
type Workspace struct {
	Base
	OrganizationID uuid.UUID `gorm:"type:uuid;not null;index" json:"organization_id"`
	Name           string    `gorm:"not null" json:"name"`
	Slug           string    `gorm:"not null" json:"slug"`
	Description    *string   `json:"description,omitempty"`

	Organization   Organization      `gorm:"foreignKey:OrganizationID" json:"-"`
	Members        []WorkspaceMember `gorm:"foreignKey:WorkspaceID" json:"-"`
	SocialAccounts []SocialAccount   `gorm:"foreignKey:WorkspaceID" json:"-"`
	Posts          []Post            `gorm:"foreignKey:WorkspaceID" json:"-"`
	MediaFiles     []MediaFile       `gorm:"foreignKey:WorkspaceID" json:"-"`
}

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// WorkspaceMember maps a User to a Workspace with a role.
type WorkspaceMember struct {
	Base
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index" json:"workspace_id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Role        Role      `gorm:"not null;default:'member'" json:"role"`

	Workspace Workspace `gorm:"foreignKey:WorkspaceID" json:"-"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
