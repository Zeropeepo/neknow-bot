package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/Zeropeepo/neknow-bot/internal/auth/domain"
	"github.com/Zeropeepo/neknow-bot/internal/auth/dto"

	"github.com/Zeropeepo/neknow-bot/pkg/response"
)

type AuthHandler struct {
	service		domain.Service
	validate	*validator.Validate
}

func NewAuthHandler(service domain.Service) *AuthHandler {
	return &AuthHandler{
		service: service,
		validate: validator.New(),
	}
}

func (h *AuthHandler) Register(c *gin.Context){
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.BadRequest(c, "validation failed: "+err.Error())
		return
	}

	user, err := h.service.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err.Error() == "email already exists" {
			response.BadRequest(c, "email already exists")
		}
		response.InternalError(c)
		return
	}

	response.Created(c, dto.UserResponse{
		ID: user.ID,
		Email: user.Email,
		CreatedAt: user.CreatedAt,
	})
}

func (h *AuthHandler) Login(c *gin.Context){
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.BadRequest(c, "validation failed: "+err.Error())
		return
	}

	accessToken, refreshToken, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err.Error() == "invalid credentials" {
			response.Unauthorized(c)
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, dto.TokenResponse{
		AccessToken: accessToken,
		RefreshToken: refreshToken,
		TokenType: "Bearer",
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context){
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request payload")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.BadRequest(c, "validation failed: "+err.Error())
		return
	}

	newAccessToken, err := h.service.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if err.Error() == "invalid refresh token" {
			response.Unauthorized(c)
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, dto.RefreshResponse{
		AccessToken: newAccessToken,
		TokenType: "Bearer",
	})

}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString("user_id")
	email := c.GetString("user_email")

	if userID == "" {
		response.Unauthorized(c)
		return
	}

	response.Success(c, dto.UserResponse{
		ID:    userID,
		Email: email,
	})
}