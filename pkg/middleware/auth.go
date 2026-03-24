package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Zeropeepo/neknow-bot/internal/auth/domain"
	"github.com/Zeropeepo/neknow-bot/pkg/response"
)

const (
	UserIDKey    = "user_id"
	UserEmailKey = "user_email"
)

func AuthMiddleware(authService domain.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		claims, err := authService.ValidateToken(c.Request.Context(), parts[1])
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(UserEmailKey, claims.Email)

		c.Next()
	}
}
