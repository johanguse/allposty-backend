package repository

import (
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrgRepository struct {
	db *gorm.DB
}

func NewOrgRepository(db *gorm.DB) *OrgRepository {
	return &OrgRepository{db: db}
}

// --- Organization ---

func (r *OrgRepository) CreateOrg(org *models.Organization) error {
	return r.db.Create(org).Error
}

func (r *OrgRepository) FindOrgByID(id uuid.UUID) (*models.Organization, error) {
	var org models.Organization
	if err := r.db.First(&org, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &org, nil
}

func (r *OrgRepository) FindOrgBySlug(slug string) (*models.Organization, error) {
	var org models.Organization
	if err := r.db.Where("slug = ?", slug).First(&org).Error; err != nil {
		return nil, err
	}
	return &org, nil
}

// FindOrgsByUser returns all orgs the user owns or is a member of (via a workspace).
func (r *OrgRepository) FindOrgsByUser(userID uuid.UUID) ([]models.Organization, error) {
	var orgs []models.Organization
	err := r.db.
		Joins("JOIN workspaces ON workspaces.organization_id = organizations.id").
		Joins("JOIN workspace_members ON workspace_members.workspace_id = workspaces.id AND workspace_members.user_id = ?", userID).
		Where("workspaces.deleted_at IS NULL AND workspace_members.deleted_at IS NULL").
		Or("organizations.owner_id = ?", userID).
		Distinct("organizations.*").
		Find(&orgs).Error
	return orgs, err
}

// --- Workspace ---

func (r *OrgRepository) CreateWorkspace(ws *models.Workspace) error {
	return r.db.Create(ws).Error
}

func (r *OrgRepository) FindWorkspaceByID(id uuid.UUID) (*models.Workspace, error) {
	var ws models.Workspace
	if err := r.db.First(&ws, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &ws, nil
}

func (r *OrgRepository) FindWorkspacesByOrg(orgID uuid.UUID) ([]models.Workspace, error) {
	var workspaces []models.Workspace
	err := r.db.Where("organization_id = ?", orgID).Find(&workspaces).Error
	return workspaces, err
}

// --- Members ---

func (r *OrgRepository) AddMember(member *models.WorkspaceMember) error {
	return r.db.Create(member).Error
}

func (r *OrgRepository) FindMember(workspaceID, userID uuid.UUID) (*models.WorkspaceMember, error) {
	var m models.WorkspaceMember
	if err := r.db.Where("workspace_id = ? AND user_id = ?", workspaceID, userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *OrgRepository) ListMembers(workspaceID uuid.UUID) ([]models.WorkspaceMember, error) {
	var members []models.WorkspaceMember
	err := r.db.Preload("User").Where("workspace_id = ?", workspaceID).Find(&members).Error
	return members, err
}

// CountWorkspacesByOwner returns the total number of workspaces across all orgs owned by userID.
func (r *OrgRepository) CountWorkspacesByOwner(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.Workspace{}).
		Joins("JOIN organizations ON organizations.id = workspaces.organization_id").
		Where("organizations.owner_id = ? AND workspaces.deleted_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

func (r *OrgRepository) UserHasWorkspaceAccess(workspaceID, userID uuid.UUID) bool {
	var count int64
	r.db.Model(&models.WorkspaceMember{}).
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Count(&count)
	return count > 0
}
