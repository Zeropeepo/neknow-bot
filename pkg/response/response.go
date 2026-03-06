package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
	})
}

func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Error:   message,
	})
}

func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, APIResponse{
		Success: false,
		Error:   "unauthorized",
	})
}

func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Error:   message,
	})
}

func Forbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, APIResponse{
		Success: false,
		Error:   "forbidden",
	})
}

func InternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error:   "internal server error",
	})
}
