package domain

import (
	"context"
	"time"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Conversation struct {
	ID        string
	BotID     string
	UserID    string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Message struct {
	ID             string
	ConversationID string
	Role           Role
	Content        string
	CreatedAt      time.Time
}

type RAGRequest struct {
	BotID       string
	SystemPrompt string
	Query       string
	History     []HistoryMessage
}

type HistoryMessage struct {
	Role    Role
	Content string
}

type ConversationRepository interface {
	Create(ctx context.Context, conv *Conversation) error
	FindByID(ctx context.Context, id string) (*Conversation, error)
	FindByBotID(ctx context.Context, botID string) ([]*Conversation, error)
	Delete(ctx context.Context, id string) error
}

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) error
	FindByConversationID(ctx context.Context, conversationID string, limit int) ([]*Message, error)
}

type Service interface {
	CreateConversation(ctx context.Context, userID, botID string) (*Conversation, error)
	GetConversations(ctx context.Context, userID, botID string) ([]*Conversation, error)
	GetConversation(ctx context.Context, userID, convID string) (*Conversation, []*Message, error)
	SendMessage(ctx context.Context, userID, convID string, content string) (<-chan string, error)
	DeleteConversation(ctx context.Context, userID, convID string) error
}
