package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/Zeropeepo/neknow-bot/internal/chat/domain"
	"github.com/Zeropeepo/neknow-bot/internal/chat/dto"
	chatService "github.com/Zeropeepo/neknow-bot/internal/chat/service"
	"github.com/Zeropeepo/neknow-bot/pkg/middleware"
	"github.com/Zeropeepo/neknow-bot/pkg/response"
)

type ChatHandler struct {
	service  domain.Service
	validate *validator.Validate
}

func NewChatHandler(service domain.Service) *ChatHandler {
	return &ChatHandler{
		service:  service,
		validate: validator.New(),
	}
}

func (h *ChatHandler) CreateConversation(c *gin.Context) {
	botID := c.Param("id")
	userID := c.GetString(middleware.UserIDKey)

	conv, err := h.service.CreateConversation(c.Request.Context(), userID, botID)
	if err != nil {
		if errors.Is(err, chatService.ErrUnauthorized) {
			response.Unauthorized(c)
			return
		}
		response.InternalError(c)
		return
	}

	response.Created(c, dto.CreateConversationResponse{
		ID:        conv.ID,
		BotID:     conv.BotID,
		Title:     conv.Title,
		CreatedAt: conv.CreatedAt,
	})
}

func (h *ChatHandler) GetConversations(c *gin.Context) {
	botID := c.Param("id")
	userID := c.GetString(middleware.UserIDKey)

	convs, err := h.service.GetConversations(c.Request.Context(), userID, botID)
	if err != nil {
		if errors.Is(err, chatService.ErrUnauthorized) {
			response.Unauthorized(c)
			return
		}
		response.InternalError(c)
		return
	}

	result := make([]*dto.CreateConversationResponse, len(convs))
	for i, conv := range convs {
		result[i] = &dto.CreateConversationResponse{
			ID:        conv.ID,
			BotID:     conv.BotID,
			Title:     conv.Title,
			CreatedAt: conv.CreatedAt,
		}
	}

	response.Success(c, dto.ConversationListResponse{
		Conversations: result,
		Total:         len(result),
	})
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
	convID := c.Param("conv_id")
	userID := c.GetString(middleware.UserIDKey)

	conv, messages, err := h.service.GetConversation(c.Request.Context(), userID, convID)
	if err != nil {
		switch {
		case errors.Is(err, chatService.ErrConversationNotFound):
			response.NotFound(c, "conversation not found")
		case errors.Is(err, chatService.ErrUnauthorized):
			response.Unauthorized(c)
		default:
			response.InternalError(c)
		}
		return
	}

	msgResponses := make([]*dto.MessageResponse, len(messages))
	for i, msg := range messages {
		msgResponses[i] = &dto.MessageResponse{
			ID:             msg.ID,
			ConversationID: msg.ConversationID,
			Role:           string(msg.Role),
			Content:        msg.Content,
			CreatedAt:      msg.CreatedAt,
		}
	}

	response.Success(c, dto.ConversationDetailResponse{
		ID:        conv.ID,
		BotID:     conv.BotID,
		Title:     conv.Title,
		Messages:  msgResponses,
		CreatedAt: conv.CreatedAt,
	})
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	convID := c.Param("conv_id")
	userID := c.GetString(middleware.UserIDKey)

	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokenCh, err := h.service.SendMessage(c.Request.Context(), userID, convID, req.Content)
	if err != nil {
		switch {
		case errors.Is(err, chatService.ErrConversationNotFound):
			response.NotFound(c, "conversation not found")
		case errors.Is(err, chatService.ErrUnauthorized):
			response.Unauthorized(c)
		default:
			log.Printf("send_message_failed user_id=%s conv_id=%s err=%v", userID, convID, err)
			response.InternalError(c)
		}
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		token, ok := <-tokenCh
		if !ok {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			return false
		}

		event := dto.SSEEvent{Content: token}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		return true
	})
}

func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	convID := c.Param("conv_id")
	userID := c.GetString(middleware.UserIDKey)

	err := h.service.DeleteConversation(c.Request.Context(), userID, convID)
	if err != nil {
		switch {
		case errors.Is(err, chatService.ErrConversationNotFound):
			response.NotFound(c, "conversation not found")
		case errors.Is(err, chatService.ErrUnauthorized):
			response.Unauthorized(c)
		default:
			response.InternalError(c)
		}
		return
	}

	response.Success(c, nil)
}

func (h *ChatHandler) UpdateConversation(c *gin.Context) {
	convID := c.Param("conv_id")
	userID := c.GetString(middleware.UserIDKey)

	var req dto.UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	conv, err := h.service.UpdateConversation(c.Request.Context(), userID, convID, req.Title)
	if err != nil {
		switch {
		case errors.Is(err, chatService.ErrConversationNotFound):
			response.NotFound(c, "conversation not found")
		case errors.Is(err, chatService.ErrUnauthorized):
			response.Unauthorized(c)
		default:
			response.InternalError(c)
		}
		return
	}

	response.Success(c, dto.CreateConversationResponse{
		ID:        conv.ID,
		BotID:     conv.BotID,
		Title:     conv.Title,
		CreatedAt: conv.CreatedAt,
	})
}
