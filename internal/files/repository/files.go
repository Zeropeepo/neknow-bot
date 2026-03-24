package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Zeropeepo/neknow-bot/internal/files/domain"
	"github.com/Zeropeepo/neknow-bot/pkg/database/models"
)

type fileRepository struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) domain.Repository {
	return &fileRepository{db: db}
}

func (r *fileRepository) Create(ctx context.Context, file *domain.File) error {
	file.ID = uuid.New().String()
	file.CreatedAt = time.Now()
	file.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(toModel(file)).Error
}

func (r *fileRepository) FindByID(ctx context.Context, id string) (*domain.File, error) {
	if !isValidUUID(id) {
		return nil, nil
	}

	var m models.File
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toDomain(&m), nil
}

func (r *fileRepository) FindByBotID(ctx context.Context, botID string) ([]*domain.File, error) {
	var ms []models.File
	result := r.db.WithContext(ctx).
		Where("bot_id = ?", botID).
		Order("created_at DESC").
		Find(&ms)
	if result.Error != nil {
		return nil, result.Error
	}

	files := make([]*domain.File, len(ms))
	for i, m := range ms {
		files[i] = toDomain(&m)
	}
	return files, nil
}

func (r *fileRepository) UpdateStatus(ctx context.Context, id string, status domain.FileStatus, errMsg string) error {
	return r.db.WithContext(ctx).
		Model(&models.File{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":     string(status),
			"error_msg":  errMsg,
			"updated_at": time.Now(),
		}).Error
}

func (r *fileRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.File{}).Error
}

// Helper
func isValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

func toModel(f *domain.File) *models.File {
	return &models.File{
		ID:        f.ID,
		BotID:     f.BotID,
		UserID:    f.UserID,
		Name:      f.Name,
		Size:      f.Size,
		MimeType:  f.MimeType,
		Bucket:    f.Bucket,
		ObjectKey: f.ObjectKey,
		Status:    string(f.Status),
		ErrorMsg:  f.ErrorMsg,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}

func toDomain(m *models.File) *domain.File {
	return &domain.File{
		ID:        m.ID,
		BotID:     m.BotID,
		UserID:    m.UserID,
		Name:      m.Name,
		Size:      m.Size,
		MimeType:  m.MimeType,
		Bucket:    m.Bucket,
		ObjectKey: m.ObjectKey,
		Status:    domain.FileStatus(m.Status),
		ErrorMsg:  m.ErrorMsg,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
