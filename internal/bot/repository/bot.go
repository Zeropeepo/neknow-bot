package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Zeropeepo/neknow-bot/internal/bot/domain"
	"github.com/Zeropeepo/neknow-bot/pkg/database/models"
)

type botRepository struct{
	db *gorm.DB
}

func NewBotRepository(db *gorm.DB) domain.Repository {
	return &botRepository{db: db}
}

// Mapper
func toModel(b *domain.Bot) *models.Bot {
	return &models.Bot{
		ID:				b.ID,
		UserID:			b.UserID,
		Name:			b.Name,
		Description:	b.Description,
		SystemPrompt:	b.SystemPrompt,
		Model:			string(b.Model),
		IsPublic:		b.IsPublic,
		Status:			string(b.Status),
		CreatedAt:		b.CreatedAt,
		UpdatedAt:		b.UpdatedAt,
	}
}

func toDomain(m *models.Bot) *domain.Bot {
	return &domain.Bot{
		ID:           m.ID,
		UserID:       m.UserID,
		Name:         m.Name,
		Description:  m.Description,
		SystemPrompt: m.SystemPrompt,
		Model:        domain.BotModel(m.Model),
		IsPublic:     m.IsPublic,
		Status:       domain.BotStatus(m.Status),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// Functions
func (r *botRepository) Create(ctx context.Context, bot *domain.Bot) error {
	bot.ID = uuid.New().String()
	bot.CreatedAt = time.Now()
	bot.UpdatedAt = time.Now()

	dbBot := toModel(bot)
	return r.db.WithContext(ctx).Create(dbBot).Error
}


func (r *botRepository) FindByID(ctx context.Context, id string) (*domain.Bot, error) {
	var dbBot models.Bot
	results := r.db.WithContext(ctx).Where("id = ?", id).First(&dbBot)

	if errors.Is(results.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if results.Error != nil {
		if strings.Contains(results.Error.Error(), "invalid input syntax"){
			return nil, nil
		}
		return nil, results.Error
	}

	return toDomain(&dbBot), nil
}

func (r *botRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Bot, error) {
	var dbBots []models.Bot
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&dbBots)
	if result.Error != nil {
		return nil, result.Error
	}
	bots := make([]*domain.Bot, len(dbBots))
	for i, dbBot := range dbBots {
		bots[i] = toDomain(&dbBot)
	}
	return bots, nil
}

func (r *botRepository) Update(ctx context.Context, bot *domain.Bot) error {
	bot.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).
		Model(&models.Bot{}).
		Where("id = ?", bot.ID).
		Updates(toModel(bot)).Error
}

func (r *botRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.Bot{}).Error
}