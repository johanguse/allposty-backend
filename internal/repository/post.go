package repository

import (
	"time"

	"github.com/allposty/allposty-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) Create(post *models.Post) error {
	return r.db.Create(post).Error
}

func (r *PostRepository) FindByID(id uuid.UUID) (*models.Post, error) {
	var post models.Post
	if err := r.db.Preload("Platforms.SocialAccount").First(&post, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *PostRepository) FindByWorkspace(workspaceID uuid.UUID, status *models.PostStatus) ([]models.Post, error) {
	q := r.db.Preload("Platforms").Where("workspace_id = ?", workspaceID)
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	var posts []models.Post
	return posts, q.Order("created_at desc").Find(&posts).Error
}

// FindScheduledBefore returns scheduled posts due for publishing.
func (r *PostRepository) FindScheduledBefore(t time.Time) ([]models.Post, error) {
	var posts []models.Post
	err := r.db.Preload("Platforms.SocialAccount").
		Where("status = ? AND scheduled_at <= ?", models.PostStatusScheduled, t).
		Find(&posts).Error
	return posts, err
}

func (r *PostRepository) Update(post *models.Post) error {
	return r.db.Save(post).Error
}

func (r *PostRepository) UpdateStatus(id uuid.UUID, status models.PostStatus, errMsg *string) error {
	updates := map[string]any{"status": status}
	if errMsg != nil {
		updates["error_message"] = *errMsg
	}
	if status == models.PostStatusPublished {
		now := time.Now()
		updates["published_at"] = now
	}
	return r.db.Model(&models.Post{}).Where("id = ?", id).Updates(updates).Error
}

func (r *PostRepository) UpdatePlatformStatus(id uuid.UUID, status models.PostStatus, platformPostID *string, errMsg *string) error {
	updates := map[string]any{"status": status}
	if platformPostID != nil {
		updates["platform_post_id"] = *platformPostID
	}
	if errMsg != nil {
		updates["error_message"] = *errMsg
	}
	if status == models.PostStatusPublished {
		now := time.Now()
		updates["published_at"] = now
	}
	return r.db.Model(&models.PostPlatform{}).Where("id = ?", id).Updates(updates).Error
}

func (r *PostRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Post{}, "id = ?", id).Error
}

// FindByDateRange returns posts in a workspace within a date range (for the calendar view).
func (r *PostRepository) FindByDateRange(workspaceID uuid.UUID, from, to time.Time) ([]models.Post, error) {
	var posts []models.Post
	err := r.db.Preload("Platforms").
		Where("workspace_id = ? AND scheduled_at BETWEEN ? AND ?", workspaceID, from, to).
		Order("scheduled_at asc").
		Find(&posts).Error
	return posts, err
}
