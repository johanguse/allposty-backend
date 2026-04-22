package repository

import (
	"github.com/allposty/allposty-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(sub *models.Subscription) error {
	return r.db.Create(sub).Error
}

func (r *SubscriptionRepository) FindByOrg(orgID uuid.UUID) (*models.Subscription, error) {
	var sub models.Subscription
	if err := r.db.Where("organization_id = ?", orgID).First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *SubscriptionRepository) FindByStripeSubID(stripeSubID string) (*models.Subscription, error) {
	var sub models.Subscription
	if err := r.db.Where("stripe_sub_id = ?", stripeSubID).First(&sub).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *SubscriptionRepository) Update(sub *models.Subscription) error {
	return r.db.Save(sub).Error
}
