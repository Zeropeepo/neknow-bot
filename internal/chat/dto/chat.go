package dto

import "time"

type CreateConversationResponse struct {
	ID        string    `json:"id"`
	BotID     string    `json:"bot_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type ConversationListResponse struct {
	Conversations []*CreateConversationResponse `json:"conversations"`
	Total         int                           `json:"total"`
}

type MessageResponse struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

type ConversationDetailResponse struct {
	ID        string             `json:"id"`
	BotID     string             `json:"bot_id"`
	Title     string             `json:"title"`
	Messages  []*MessageResponse `json:"messages"`
	CreatedAt time.Time          `json:"created_at"`
}

type SendMessageRequest struct {
	Content string `json:"content" validate:"required,min=1,max=2000"`
}

type UpdateConversationRequest struct {
	Title string `json:"title" validate:"required,min=1,max=100"`
}

// SSE event format
type SSEEvent struct {
	Content string `json:"content"`
}
