package repository

import (
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MediaRepository struct {
	db *gorm.DB
}

func NewMediaRepository(db *gorm.DB) *MediaRepository {
	return &MediaRepository{db: db}
}

func (r *MediaRepository) Create(file *models.MediaFile) error {
	return r.db.Create(file).Error
}

func (r *MediaRepository) FindByID(id uuid.UUID) (*models.MediaFile, error) {
	var f models.MediaFile
	if err := r.db.First(&f, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *MediaRepository) FindByWorkspace(workspaceID uuid.UUID, folder *string) ([]models.MediaFile, error) {
	q := r.db.Where("workspace_id = ?", workspaceID)
	if folder != nil {
		q = q.Where("folder = ?", *folder)
	}
	var files []models.MediaFile
	return files, q.Order("created_at desc").Find(&files).Error
}

func (r *MediaRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.MediaFile{}, "id = ?", id).Error
}
