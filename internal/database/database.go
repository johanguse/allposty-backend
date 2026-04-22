package database

import (
	"github.com/allposty/allposty-backend/internal/models"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string, log *zap.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	log.Info("database connected")
	return db, nil
}

// Migrate runs auto-migrations for all models.
// For production, prefer versioned SQL migrations instead.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.Workspace{},
		&models.WorkspaceMember{},
		&models.SocialAccount{},
		&models.Post{},
		&models.PostPlatform{},
		&models.MediaFile{},
		&models.Subscription{},
		&models.RefreshToken{},
	)
}
