package domain

import (
	"context"
	"time"
)

type BotStatus string
type BotModel string

const (
	BotStatusActive   BotStatus = "active"
	BotStatusInactive BotStatus = "inactive"
)

// Bot adalah representasi bot di level bisnis.
type Bot struct {
	ID           string
	UserID       string
	Name         string
	Description  string
	SystemPrompt string
	Model        BotModel
	IsPublic     bool
	Status       BotStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateBotRequest adalah data yang dibutuhkan untuk membuat bot baru.
type CreateBotRequest struct {
	Name         string
	Description  string
	SystemPrompt string
	Model        BotModel
	IsPublic     bool
}

// UpdateBotRequest menggunakan pointer supaya bisa bedakan
// field yang tidak dikirim (nil) vs dikirim kosong ("").
type UpdateBotRequest struct {
	Name         *string
	Description  *string
	SystemPrompt *string
	Model        *BotModel
	IsPublic     *bool
	Status       *BotStatus
}

type Repository interface {
	Create(ctx context.Context, bot *Bot) error
	FindByID(ctx context.Context, id string) (*Bot, error)
	FindByUserID(ctx context.Context, userID string) ([]*Bot, error)
	Update(ctx context.Context, bot *Bot) error
	Delete(ctx context.Context, id string) error
}

type Service interface {
	Create(ctx context.Context, userID string, req CreateBotRequest) (*Bot, error)
	GetByID(ctx context.Context, userID, botID string) (*Bot, error)
	GetAllByUser(ctx context.Context, userID string) ([]*Bot, error)
	Update(ctx context.Context, userID, botID string, req UpdateBotRequest) (*Bot, error)
	Delete(ctx context.Context, userID, botID string) error
}
