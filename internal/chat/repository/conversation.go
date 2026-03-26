package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Zeropeepo/neknow-bot/internal/chat/domain"
	"github.com/Zeropeepo/neknow-bot/pkg/database/models"
)

type conversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) domain.ConversationRepository {
	return &conversationRepository{db: db}
}

func (r *conversationRepository) Create(ctx context.Context, conv *domain.Conversation) error {
	conv.ID = uuid.New().String()
	conv.CreatedAt = time.Now()
	conv.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(toConvModel(conv)).Error
}

func (r *conversationRepository) FindByID(ctx context.Context, id string) (*domain.Conversation, error) {
	if !isValidUUID(id) {
		return nil, nil
	}

	var m models.Conversation
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return toConvDomain(&m), nil
}

func (r *conversationRepository) FindByBotID(ctx context.Context, botID string) ([]*domain.Conversation, error) {
	var ms []models.Conversation
	result := r.db.WithContext(ctx).
		Where("bot_id = ?", botID).
		Order("updated_at DESC").  
		Find(&ms)
	if result.Error != nil {
		return nil, result.Error
	}

	convs := make([]*domain.Conversation, len(ms))
	for i, m := range ms {
		convs[i] = toConvDomain(&m)
	}
	return convs, nil
}

func (r *conversationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.Conversation{}).Error
}


// Mapper
func toConvModel(c *domain.Conversation) *models.Conversation {
	return &models.Conversation{
		ID:        c.ID,
		BotID:     c.BotID,
		UserID:    c.UserID,
		Title:     c.Title,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func toConvDomain(m *models.Conversation) *domain.Conversation {
	return &domain.Conversation{
		ID:        m.ID,
		BotID:     m.BotID,
		UserID:    m.UserID,
		Title:     m.Title,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func isValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
