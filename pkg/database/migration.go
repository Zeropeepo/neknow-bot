package database

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/Zeropeepo/neknow-bot/pkg/database/models"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.User{},
		&models.Bot{},
	); err != nil {
		return fmt.Errorf("failed to migrate: %w", err)
	}
	return nil
}
