package service

import (
	"context"
	"errors"

	"github.com/Zeropeepo/neknow-bot/internal/bot/domain"
	"github.com/Zeropeepo/neknow-bot/pkg/config"
)

var (
	ErrBotNotFound    = errors.New("bot not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrInvalidModel   = errors.New("invalid model")
)

type botService struct {
	repo domain.Repository
	cfg  *config.Config
}

func NewBotService(repo domain.Repository, cfg *config.Config) domain.Service {
	return &botService{repo: repo, cfg: cfg}
}

func (s *botService) Create(ctx context.Context, userID string, req domain.CreateBotRequest) (*domain.Bot, error) {
	if !s.isValidModel(string(req.Model)) {
		return nil, ErrInvalidModel
	}

	bot := &domain.Bot{
		UserID:       userID,
		Name:         req.Name,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		Model:        req.Model,
		IsPublic:     req.IsPublic,
		Status:       domain.BotStatusActive, // default active saat dibuat
	}

	if err := s.repo.Create(ctx, bot); err != nil {
		return nil, err
	}

	return bot, nil
}

func (s *botService) GetByID(ctx context.Context, userID, botID string) (*domain.Bot, error) {
	bot, err := s.repo.FindByID(ctx, botID)
	if err != nil {
		return nil, err
	}
	if bot == nil {
		return nil, ErrBotNotFound
	}

	// Bot private hanya bisa diakses pemiliknya
	if !bot.IsPublic && bot.UserID != userID {
		return nil, ErrUnauthorized
	}

	return bot, nil
}

func (s *botService) GetAllByUser(ctx context.Context, userID string) ([]*domain.Bot, error) {
	return s.repo.FindByUserID(ctx, userID)
}

func (s *botService) Update(ctx context.Context, userID, botID string, req domain.UpdateBotRequest) (*domain.Bot, error) {
	bot, err := s.repo.FindByID(ctx, botID)
	if err != nil {
		return nil, err
	}
	if bot == nil {
		return nil, ErrBotNotFound
	}
	if bot.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Hanya update field yang dikirim (tidak nil)
	if req.Name != nil {
		bot.Name = *req.Name
	}
	if req.Description != nil {
		bot.Description = *req.Description
	}
	if req.SystemPrompt != nil {
		bot.SystemPrompt = *req.SystemPrompt
	}
	if req.Model != nil {
		if !s.isValidModel(string(*req.Model)) {
			return nil, ErrInvalidModel
		}
		bot.Model = *req.Model
	}
	if req.IsPublic != nil {
		bot.IsPublic = *req.IsPublic
	}
	if req.Status != nil {
		bot.Status = *req.Status
	}

	if err := s.repo.Update(ctx, bot); err != nil {
		return nil, err
	}

	return bot, nil
}

func (s *botService) Delete(ctx context.Context, userID, botID string) error {
	bot, err := s.repo.FindByID(ctx, botID)
	if err != nil {
		return err
	}
	if bot == nil {
		return ErrBotNotFound
	}
	if bot.UserID != userID {
		return ErrUnauthorized
	}

	return s.repo.Delete(ctx, botID)
}

// ─── Helper ─────────────────────────────────────────────

func (s *botService) isValidModel(model string) bool {
	for _, m := range s.cfg.Bot.AllowedModels {
		if m == model {
			return true
		}
	}
	return false
}
