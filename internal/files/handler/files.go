package handler

import (
	"errors"
	"io"
	

	"github.com/gin-gonic/gin"

	"github.com/Zeropeepo/neknow-bot/internal/files/domain"
	"github.com/Zeropeepo/neknow-bot/internal/files/dto"
	fileService "github.com/Zeropeepo/neknow-bot/internal/files/service"
	"github.com/Zeropeepo/neknow-bot/pkg/middleware"
	"github.com/Zeropeepo/neknow-bot/pkg/response"
)

const maxUploadSize = 20 * 1024 * 1024 // 20MB

type FileHandler struct {
	service domain.Service
}

func NewFileHandler(service domain.Service) *FileHandler {
	return &FileHandler{service: service}
}

func (h *FileHandler) Upload(c *gin.Context) {
	botID := c.Param("id")
	userID := c.GetString(middleware.UserIDKey)


	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	defer file.Close()

	if header.Size > maxUploadSize {
		response.BadRequest(c, "file too large (Max 20MB)")
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		response.BadRequest(c, "failed to read file")
		return
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	result, err := h.service.Upload(c.Request.Context(), userID, botID, domain.UploadFileRequest{
		Name:     header.Filename,
		Size:     header.Size,
		MimeType: mimeType,
		Content:  content,
	})
	if err != nil {
		switch {
		case errors.Is(err, fileService.ErrUnauthorized):
			response.Unauthorized(c)
		case errors.Is(err, fileService.ErrInvalidFileType):
			response.BadRequest(c, err.Error())
		case errors.Is(err, fileService.ErrFileTooLarge):
			response.BadRequest(c, err.Error())
		default:
			response.InternalError(c)
		}
		return
	}

	response.Created(c, toFileResponse(result))
}

func (h *FileHandler) GetAll(c *gin.Context) {
	botID := c.Param("id")
	userID := c.GetString(middleware.UserIDKey)

	files, err := h.service.GetAllByBot(c.Request.Context(), userID, botID)
	if err != nil {
		if errors.Is(err, fileService.ErrUnauthorized) {
			response.Unauthorized(c)
			return
		}
		response.InternalError(c)
		return
	}

	fileResponses := make([]*dto.FileResponse, len(files))
	for i, f := range files {
		fileResponses[i] = toFileResponse(f)
	}

	response.Success(c, dto.FileListResponse{
		Files: fileResponses,
		Total: len(fileResponses),
	})
}

func (h *FileHandler) GetByID(c *gin.Context) {
	botID := c.Param("id")
	fileID := c.Param("file_id")
	userID := c.GetString(middleware.UserIDKey)

	file, err := h.service.GetByID(c.Request.Context(), userID, botID, fileID)
	if err != nil {
		switch {
		case errors.Is(err, fileService.ErrFileNotFound):
			response.NotFound(c, "file not found")
		case errors.Is(err, fileService.ErrUnauthorized):
			response.Unauthorized(c)
		default:
			response.InternalError(c)
		}
		return
	}

	response.Success(c, toFileResponse(file))
}

func (h *FileHandler) Delete(c *gin.Context) {
	botID := c.Param("id")
	fileID := c.Param("file_id")
	userID := c.GetString(middleware.UserIDKey)

	err := h.service.Delete(c.Request.Context(), userID, botID, fileID)
	if err != nil {
		switch {
		case errors.Is(err, fileService.ErrFileNotFound):
			response.NotFound(c, "file not found")
		case errors.Is(err, fileService.ErrUnauthorized):
			response.Unauthorized(c)
		default:
			response.InternalError(c)
		}
		return
	}

	response.Success(c, nil)
}


// Helper
func toFileResponse(f *domain.File) *dto.FileResponse {
	return &dto.FileResponse{
		ID:        f.ID,
		BotID:     f.BotID,
		Name:      f.Name,
		Size:      f.Size,
		MimeType:  f.MimeType,
		Status:    string(f.Status),
		ErrorMsg:  f.ErrorMsg,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}
