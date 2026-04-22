package repository

import (
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// Refresh token methods

func (r *UserRepository) CreateRefreshToken(token *models.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *UserRepository) FindRefreshToken(tokenStr string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	if err := r.db.Where("token = ? AND revoked = false", tokenStr).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *UserRepository) RevokeRefreshToken(tokenStr string) error {
	return r.db.Model(&models.RefreshToken{}).
		Where("token = ?", tokenStr).
		Update("revoked", true).Error
}

func (r *UserRepository) RevokeAllUserTokens(userID uuid.UUID) error {
	return r.db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked = false", userID).
		Update("revoked", true).Error
}
