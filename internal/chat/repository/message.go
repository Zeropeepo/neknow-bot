package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Zeropeepo/neknow-bot/internal/chat/domain"
	"github.com/Zeropeepo/neknow-bot/pkg/database/models"
)

type messageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) domain.MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, msg *domain.Message) error {
	msg.ID = uuid.New().String()
	msg.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(toMsgModel(msg)).Error
}

func (r *messageRepository) FindByConversationID(ctx context.Context, conversationID string, limit int) ([]*domain.Message, error) {
	var ms []models.Message

	result := r.db.WithContext(ctx).
		Raw(`SELECT * FROM messages
			 WHERE conversation_id = ?
			 ORDER BY created_at DESC
			 LIMIT ?`, conversationID, limit).
		Scan(&ms)

	if result.Error != nil {
		return nil, result.Error
	}

	for i, j := 0, len(ms)-1; i < j; i, j = i+1, j-1 {
		ms[i], ms[j] = ms[j], ms[i]
	}

	msgs := make([]*domain.Message, len(ms))
	for i, m := range ms {
		msgs[i] = toMsgDomain(&m)
	}
	return msgs, nil
}

// Mapper
func toMsgModel(m *domain.Message) *models.Message {
	return &models.Message{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Role:           string(m.Role),
		Content:        m.Content,
		CreatedAt:      m.CreatedAt,
	}
}

func toMsgDomain(m *models.Message) *domain.Message {
	return &domain.Message{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Role:           domain.Role(m.Role),
		Content:        m.Content,
		CreatedAt:      m.CreatedAt,
	}
}
