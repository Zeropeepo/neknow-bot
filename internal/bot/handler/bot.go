package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/Zeropeepo/neknow-bot/internal/bot/domain"
	"github.com/Zeropeepo/neknow-bot/internal/bot/dto"
	botService "github.com/Zeropeepo/neknow-bot/internal/bot/service"
	"github.com/Zeropeepo/neknow-bot/pkg/middleware"
	"github.com/Zeropeepo/neknow-bot/pkg/response"
)

type BotHandler struct {
	service  domain.Service
	validate *validator.Validate
}

func NewBotHandler(service domain.Service) *BotHandler {
	return &BotHandler{
		service:  service,
		validate: validator.New(),
	}
}

func (h *BotHandler) Create(c *gin.Context) {
	var req dto.CreateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := c.GetString(middleware.UserIDKey)

	bot, err := h.service.Create(c.Request.Context(), userID, domain.CreateBotRequest{
		Name:         req.Name,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		Model:        domain.BotModel(req.Model),
		IsPublic:     req.IsPublic,
	})
	if err != nil {
		if errors.Is(err, botService.ErrInvalidModel) {
			response.BadRequest(c, "invalid model")
			return
		}
		response.InternalError(c)
		return
	}

	response.Created(c, toBotResponse(bot))
}

func (h *BotHandler) GetByID(c *gin.Context) {
	botID := c.Param("id")
	userID := c.GetString(middleware.UserIDKey)

	bot, err := h.service.GetByID(c.Request.Context(), userID, botID)
	if err != nil {
		if errors.Is(err, botService.ErrBotNotFound) {
			response.NotFound(c, "bot not found")
			return
		}
		if errors.Is(err, botService.ErrUnauthorized) {
			response.Unauthorized(c)
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, toBotResponse(bot))
}

func (h *BotHandler) GetAll(c *gin.Context) {
	userID := c.GetString(middleware.UserIDKey)

	bots, err := h.service.GetAllByUser(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c)
		return
	}

	botResponses := make([]*dto.BotResponse, len(bots))
	for i, bot := range bots {
		botResponses[i] = toBotResponse(bot)
	}

	response.Success(c, dto.BotListResponse{
		Bots:  botResponses,
		Total: len(botResponses),
	})
}

func (h *BotHandler) Update(c *gin.Context) {
	botID := c.Param("id")
	userID := c.GetString(middleware.UserIDKey)

	var req dto.UpdateBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var modelPtr *domain.BotModel
	if req.Model != nil {
		m := domain.BotModel(*req.Model)
		modelPtr = &m
	}

	var statusPtr *domain.BotStatus
	if req.Status != nil {
		s := domain.BotStatus(*req.Status)
		statusPtr = &s
	}

	bot, err := h.service.Update(c.Request.Context(), userID, botID, domain.UpdateBotRequest{
		Name:         req.Name,
		Description:  req.Description,
		SystemPrompt: req.SystemPrompt,
		Model:        modelPtr,
		IsPublic:     req.IsPublic,
		Status:       statusPtr,
	})
	if err != nil {
		if errors.Is(err, botService.ErrBotNotFound) {
			response.NotFound(c, "bot not found")
			return
		}
		if errors.Is(err, botService.ErrUnauthorized) {
			response.Unauthorized(c)
			return
		}
		if errors.Is(err, botService.ErrInvalidModel) {
			response.BadRequest(c, "invalid model")
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, toBotResponse(bot))
}

func (h *BotHandler) Delete(c *gin.Context) {
	botID := c.Param("id")
	userID := c.GetString(middleware.UserIDKey)

	err := h.service.Delete(c.Request.Context(), userID, botID)
	if err != nil {
		if errors.Is(err, botService.ErrBotNotFound) {
			response.NotFound(c, "bot not found")
			return
		}
		if errors.Is(err, botService.ErrUnauthorized) {
			response.Unauthorized(c)
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, nil)
}


// Mapper
func toBotResponse(b *domain.Bot) *dto.BotResponse {
	return &dto.BotResponse{
		ID:           b.ID,
		Name:         b.Name,
		Description:  b.Description,
		SystemPrompt: b.SystemPrompt,
		Model:        string(b.Model),
		IsPublic:     b.IsPublic,
		Status:       string(b.Status),
		CreatedAt:    b.CreatedAt,
		UpdatedAt:    b.UpdatedAt,
	}
}
