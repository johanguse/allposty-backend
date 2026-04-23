package repository

import (
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type APIKeyRepository struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(key *models.APIKey) error {
	return r.db.Create(key).Error
}

func (r *APIKeyRepository) FindByHash(hash string) (*models.APIKey, error) {
	var key models.APIKey
	if err := r.db.Where("key_hash = ? AND revoked = false", hash).First(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *APIKeyRepository) FindByUser(userID uuid.UUID) ([]models.APIKey, error) {
	var keys []models.APIKey
	err := r.db.Where("user_id = ? AND revoked = false", userID).
		Order("created_at DESC").
		Find(&keys).Error
	return keys, err
}

func (r *APIKeyRepository) FindByID(id uuid.UUID) (*models.APIKey, error) {
	var key models.APIKey
	if err := r.db.First(&key, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *APIKeyRepository) Revoke(id uuid.UUID) error {
	return r.db.Model(&models.APIKey{}).Where("id = ?", id).Update("revoked", true).Error
}

func (r *APIKeyRepository) TouchLastUsed(id uuid.UUID) error {
	return r.db.Model(&models.APIKey{}).Where("id = ?", id).
		Update("last_used_at", gorm.Expr("NOW()")).Error
}
