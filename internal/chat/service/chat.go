package service

import (
	"context"
	"errors"
	"strings"

	botDomain "github.com/Zeropeepo/neknow-bot/internal/bot/domain"
	"github.com/Zeropeepo/neknow-bot/internal/chat/domain"
	"github.com/Zeropeepo/neknow-bot/pkg/ai"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrUnauthorized         = errors.New("unauthorized")
)

const historyLimit = 10
const maxHistoryMessageChars = 1200

type chatService struct {
	convRepo domain.ConversationRepository
	msgRepo  domain.MessageRepository
	botRepo  botDomain.Repository
	rag      *ai.RAGClient
}

func NewChatService(
	convRepo domain.ConversationRepository,
	msgRepo domain.MessageRepository,
	botRepo botDomain.Repository,
	rag *ai.RAGClient,
) domain.Service {
	return &chatService{
		convRepo: convRepo,
		msgRepo:  msgRepo,
		botRepo:  botRepo,
		rag:      rag,
	}
}

func (s *chatService) CreateConversation(ctx context.Context, userID, botID string) (*domain.Conversation, error) {
	bot, err := s.botRepo.FindByID(ctx, botID)
	if err != nil {
		return nil, err
	}
	if bot == nil {
		return nil, ErrConversationNotFound
	}
	if !bot.IsPublic && bot.UserID != userID {
		return nil, ErrUnauthorized
	}

	conv := &domain.Conversation{
		BotID:  botID,
		UserID: userID,
		Title:  "New Conversation",
	}

	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *chatService) GetConversations(ctx context.Context, userID, botID string) ([]*domain.Conversation, error) {
	bot, err := s.botRepo.FindByID(ctx, botID)
	if err != nil {
		return nil, err
	}
	if bot == nil || (!bot.IsPublic && bot.UserID != userID) {
		return nil, ErrUnauthorized
	}
	return s.convRepo.FindByBotID(ctx, botID)
}

func (s *chatService) GetConversation(ctx context.Context, userID, convID string) (*domain.Conversation, []*domain.Message, error) {
	conv, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		return nil, nil, err
	}
	if conv == nil {
		return nil, nil, ErrConversationNotFound
	}
	if conv.UserID != userID {
		return nil, nil, ErrUnauthorized
	}

	messages, err := s.msgRepo.FindByConversationID(ctx, convID, 50)
	if err != nil {
		return nil, nil, err
	}
	return conv, messages, nil
}

func (s *chatService) SendMessage(ctx context.Context, userID, convID string, content string) (<-chan string, error) {
	conv, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, ErrConversationNotFound
	}
	if conv.UserID != userID {
		return nil, ErrUnauthorized
	}

	bot, err := s.botRepo.FindByID(ctx, conv.BotID)
	if err != nil {
		return nil, err
	}

	history, err := s.msgRepo.FindByConversationID(ctx, convID, historyLimit)
	if err != nil {
		return nil, err
	}

	historyMessages := make([]domain.HistoryMessage, 0, len(history))
	for _, msg := range history {
		historyMessages = append(historyMessages, domain.HistoryMessage{
			Role:    msg.Role,
			Content: truncateText(msg.Content, maxHistoryMessageChars),
		})
	}

	userMsg := &domain.Message{
		ConversationID: convID,
		Role:           domain.RoleUser,
		Content:        content,
	}
	if err := s.msgRepo.Create(ctx, userMsg); err != nil {
		return nil, err
	}

	tokenCh, err := s.rag.Stream(ctx, domain.RAGRequest{
		BotID:        conv.BotID,
		SystemPrompt: bot.SystemPrompt,
		Query:        content,
		History:      historyMessages,
	})
	if err != nil {
		return nil, err
	}

	resultCh := make(chan string, 100)
	go func() {
		defer close(resultCh)

		var fullResponse strings.Builder
		for token := range tokenCh {
			fullResponse.WriteString(token)
			resultCh <- token
		}

		assistantMsg := &domain.Message{
			ConversationID: convID,
			Role:           domain.RoleAssistant,
			Content:        fullResponse.String(),
		}
		_ = s.msgRepo.Create(context.Background(), assistantMsg)
	}()

	return resultCh, nil
}

func (s *chatService) DeleteConversation(ctx context.Context, userID, convID string) error {
	conv, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		return err
	}
	if conv == nil {
		return ErrConversationNotFound
	}
	if conv.UserID != userID {
		return ErrUnauthorized
	}
	return s.convRepo.Delete(ctx, convID)
}

func (s *chatService) UpdateConversation(ctx context.Context, userID, convID, title string) (*domain.Conversation, error) {
	conv, err := s.convRepo.FindByID(ctx, convID)
	if err != nil {
		return nil, err
	}
	if conv == nil {
		return nil, ErrConversationNotFound
	}
	if conv.UserID != userID {
		return nil, ErrUnauthorized
	}
	if err := s.convRepo.Update(ctx, convID, title); err != nil {
		return nil, err
	}
	conv.Title = title
	return conv, nil
}

func truncateText(text string, maxChars int) string {
	if maxChars <= 0 || len(text) <= maxChars {
		return text
	}
	return text[:maxChars]
}
