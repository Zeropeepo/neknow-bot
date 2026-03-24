package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Zeropeepo/neknow-bot/internal/files/domain"
	botDomain "github.com/Zeropeepo/neknow-bot/internal/bot/domain"
	"github.com/Zeropeepo/neknow-bot/pkg/queue"
	"github.com/Zeropeepo/neknow-bot/pkg/storage"
)

var (
	ErrFileNotFound    = errors.New("file not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInvalidFileType = errors.New("invalid file type, only PDF and DOCX are supported")
	ErrFileTooLarge    = errors.New("file too large, maximum 20MB")
)

var (
	allowedMimeTypes = map[string]bool{
		"application/pdf": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	}
	maxFileSize = int64(20 * 1024 * 1024) // 20MB
)

type fileService struct {
	fileRepo domain.Repository
	botRepo  botDomain.Repository
	storage  *storage.MinIOStorage
	queue    *queue.RabbitMQ
	bucket   string
}

func NewFileService(
	fileRepo domain.Repository,
	botRepo botDomain.Repository,
	storage *storage.MinIOStorage,
	queue *queue.RabbitMQ,
	bucket string,
) domain.Service {
	return &fileService{
		fileRepo: fileRepo,
		botRepo:  botRepo,
		storage:  storage,
		queue:    queue,
		bucket:   bucket,
	}
}

func (s *fileService) Upload(ctx context.Context, userID, botID string, req domain.UploadFileRequest) (*domain.File, error) {
	// Bot validation
	bot, err := s.botRepo.FindByID(ctx, botID)
	if err != nil {
		return nil, err
	}
	if bot == nil || bot.UserID != userID {
		return nil, ErrUnauthorized
	}

	if !allowedMimeTypes[req.MimeType] {
		return nil, ErrInvalidFileType
	}

	if req.Size > maxFileSize {
		return nil, ErrFileTooLarge
	}

	// Object key: bots/{botID}/files/{timestamp}_{filename}
	objectKey := fmt.Sprintf("bots/%s/files/%d_%s", botID, time.Now().UnixNano(), req.Name)

	// Upload to MinIO
	if err := s.storage.Upload(ctx, objectKey, req.Content, req.MimeType); err != nil {
		return nil, err
	}

	file := &domain.File{
		BotID:     botID,
		UserID:    userID,
		Name:      req.Name,
		Size:      req.Size,
		MimeType:  req.MimeType,
		Bucket:    s.bucket,
		ObjectKey: objectKey,
		Status:    domain.FileStatusPending,
	}

	if err := s.fileRepo.Create(ctx, file); err != nil {
		_ = s.storage.Delete(ctx, objectKey)
		return nil, err
	}

	if err := s.queue.PublishIndexFile(ctx, queue.IndexFileMessage{
		FileID:    file.ID,
		BotID:     botID,
		ObjectKey: objectKey,
		MimeType:  req.MimeType,
		FileName:  req.Name,
	}); err != nil {
		_ = s.fileRepo.UpdateStatus(ctx, file.ID, domain.FileStatusFailed, "failed to queue indexing: "+err.Error())
	}

	return file, nil
}

func (s *fileService) GetByID(ctx context.Context, userID, botID, fileID string) (*domain.File, error) {
	file, err := s.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, ErrFileNotFound
	}
	if file.BotID != botID || file.UserID != userID {
		return nil, ErrUnauthorized
	}
	return file, nil
}

func (s *fileService) GetAllByBot(ctx context.Context, userID, botID string) ([]*domain.File, error) {
	// Validasi bot milik user
	bot, err := s.botRepo.FindByID(ctx, botID)
	if err != nil {
		return nil, err
	}
	if bot == nil || bot.UserID != userID {
		return nil, ErrUnauthorized
	}

	return s.fileRepo.FindByBotID(ctx, botID)
}

func (s *fileService) Delete(ctx context.Context, userID, botID, fileID string) error {
	file, err := s.fileRepo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file == nil {
		return ErrFileNotFound
	}
	if file.BotID != botID || file.UserID != userID {
		return ErrUnauthorized
	}

	if err := s.storage.Delete(ctx, file.ObjectKey); err != nil {
		return err
	}

	return s.fileRepo.Delete(ctx, fileID)
}
