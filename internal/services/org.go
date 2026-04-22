package services

import (
	"errors"
	"strings"

	"github.com/allposty/allposty-backend/internal/models"
	"github.com/allposty/allposty-backend/internal/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrOrgNotFound       = errors.New("organization not found")
	ErrWorkspaceNotFound = errors.New("workspace not found")
	ErrSlugTaken         = errors.New("slug already taken")
	ErrForbidden         = errors.New("forbidden")
)

type OrgService struct {
	orgs *repository.OrgRepository
}

func NewOrgService(orgs *repository.OrgRepository) *OrgService {
	return &OrgService{orgs: orgs}
}

func (s *OrgService) CreateOrg(ownerID uuid.UUID, name string) (*models.Organization, error) {
	slug := slugify(name)

	// Ensure slug uniqueness
	if existing, _ := s.orgs.FindOrgBySlug(slug); existing != nil {
		slug = slug + "-" + uuid.New().String()[:8]
	}

	org := &models.Organization{
		Name:    name,
		Slug:    slug,
		OwnerID: ownerID,
	}
	if err := s.orgs.CreateOrg(org); err != nil {
		return nil, err
	}

	// Auto-create a default workspace
	ws := &models.Workspace{
		OrganizationID: org.ID,
		Name:           "Default",
		Slug:           "default",
	}
	if err := s.orgs.CreateWorkspace(ws); err != nil {
		return nil, err
	}

	// Add owner as workspace member
	_ = s.orgs.AddMember(&models.WorkspaceMember{
		WorkspaceID: ws.ID,
		UserID:      ownerID,
		Role:        models.RoleOwner,
	})

	return org, nil
}

func (s *OrgService) GetOrg(orgID, userID uuid.UUID) (*models.Organization, error) {
	org, err := s.orgs.FindOrgByID(orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}
	// Only owner or workspace members can see the org
	orgs, _ := s.orgs.FindOrgsByUser(userID)
	for _, o := range orgs {
		if o.ID == orgID {
			return org, nil
		}
	}
	if org.OwnerID == userID {
		return org, nil
	}
	return nil, ErrForbidden
}

func (s *OrgService) ListOrgs(userID uuid.UUID) ([]models.Organization, error) {
	return s.orgs.FindOrgsByUser(userID)
}

func (s *OrgService) CreateWorkspace(orgID, userID uuid.UUID, name string) (*models.Workspace, error) {
	org, err := s.orgs.FindOrgByID(orgID)
	if err != nil {
		return nil, ErrOrgNotFound
	}
	if org.OwnerID != userID {
		return nil, ErrForbidden
	}

	slug := slugify(name)
	ws := &models.Workspace{
		OrganizationID: orgID,
		Name:           name,
		Slug:           slug,
	}
	if err := s.orgs.CreateWorkspace(ws); err != nil {
		return nil, err
	}

	_ = s.orgs.AddMember(&models.WorkspaceMember{
		WorkspaceID: ws.ID,
		UserID:      userID,
		Role:        models.RoleOwner,
	})

	return ws, nil
}

func (s *OrgService) ListWorkspaces(orgID, userID uuid.UUID) ([]models.Workspace, error) {
	if _, err := s.GetOrg(orgID, userID); err != nil {
		return nil, err
	}
	return s.orgs.FindWorkspacesByOrg(orgID)
}

func (s *OrgService) RequireWorkspaceAccess(workspaceID, userID uuid.UUID) error {
	if !s.orgs.UserHasWorkspaceAccess(workspaceID, userID) {
		return ErrForbidden
	}
	return nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}
