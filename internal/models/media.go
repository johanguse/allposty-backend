package models

import "github.com/google/uuid"

type MediaFile struct {
	Base
	WorkspaceID uuid.UUID `gorm:"type:uuid;not null;index" json:"workspace_id"`
	UploadedBy  uuid.UUID `gorm:"type:uuid;not null" json:"uploaded_by"`
	Filename    string    `gorm:"not null" json:"filename"`
	MimeType    string    `gorm:"not null" json:"mime_type"`
	SizeBytes   int64     `gorm:"not null" json:"size_bytes"`
	R2Key       string    `gorm:"not null" json:"-"` // internal storage key
	PublicURL   string    `gorm:"not null" json:"url"`
	Folder      *string   `json:"folder,omitempty"`

	Workspace Workspace `gorm:"foreignKey:WorkspaceID" json:"-"`
}

type Subscription struct {
	Base
	OrganizationID     uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"organization_id"`
	StripeCustomerID   string    `gorm:"uniqueIndex" json:"-"`
	StripeSubID        string    `gorm:"uniqueIndex" json:"-"`
	Tier               string    `gorm:"not null;default:'free'" json:"tier"` // free | pro | agency
	Status             string    `gorm:"not null" json:"status"`              // active | canceled | past_due
	CurrentPeriodEnd   int64     `json:"current_period_end"`

	Organization Organization `gorm:"foreignKey:OrganizationID" json:"-"`
}
