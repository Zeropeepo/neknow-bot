package database

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/Zeropeepo/neknow-bot/pkg/database/models"
)

func Migrate(db *gorm.DB) error {
    if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
        return fmt.Errorf("failed to create vector extension: %w", err)
    }

    if err := db.AutoMigrate(
        &models.User{},
        &models.Bot{},
        &models.File{},
        &models.Conversation{},
        &models.Message{},
        &models.FileChunk{},
    ); err != nil {
        return fmt.Errorf("failed to migrate: %w", err)
    }

    if err := db.Exec(`
        ALTER TABLE file_chunks
        DROP CONSTRAINT IF EXISTS fk_file_chunks_file,
        ADD CONSTRAINT fk_file_chunks_file
        FOREIGN KEY (file_id) REFERENCES files(id)
        ON DELETE CASCADE
    `).Error; err != nil {
        return fmt.Errorf("failed to add cascade constraint: %w", err)
    }

    return nil
}
