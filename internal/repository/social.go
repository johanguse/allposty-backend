package repository

import (
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SocialRepository struct {
	db *gorm.DB
}

func NewSocialRepository(db *gorm.DB) *SocialRepository {
	return &SocialRepository{db: db}
}

func (r *SocialRepository) Create(account *models.SocialAccount) error {
	return r.db.Create(account).Error
}

func (r *SocialRepository) FindByID(id uuid.UUID) (*models.SocialAccount, error) {
	var a models.SocialAccount
	if err := r.db.First(&a, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *SocialRepository) FindByWorkspace(workspaceID uuid.UUID) ([]models.SocialAccount, error) {
	var accounts []models.SocialAccount
	err := r.db.Where("workspace_id = ?", workspaceID).Find(&accounts).Error
	return accounts, err
}

func (r *SocialRepository) FindByWorkspaceAndPlatformUser(workspaceID uuid.UUID, platform models.Platform, platformUserID string) (*models.SocialAccount, error) {
	var a models.SocialAccount
	err := r.db.Where("workspace_id = ? AND platform = ? AND platform_user_id = ?",
		workspaceID, platform, platformUserID).First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *SocialRepository) Update(account *models.SocialAccount) error {
	return r.db.Save(account).Error
}

func (r *SocialRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.SocialAccount{}, "id = ?", id).Error
}

func (r *SocialRepository) CountByWorkspace(workspaceID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&models.SocialAccount{}).Where("workspace_id = ?", workspaceID).Count(&count).Error
	return count, err
}
