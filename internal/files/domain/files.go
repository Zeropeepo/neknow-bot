package domain

import (
	"context"
	"time"
)

type FileStatus string

const (
	FileStatusPending  FileStatus = "pending"
	FileStatusIndexing FileStatus = "indexing"
	FileStatusIndexed  FileStatus = "indexed"
	FileStatusFailed   FileStatus = "failed"
)

type File struct {
	ID        string
	BotID     string
	UserID    string
	Name      string
	Size      int64
	MimeType  string
	Bucket    string
	ObjectKey string
	Status    FileStatus
	ErrorMsg  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Repository interface {
	Create(ctx context.Context, file *File) error
	FindByID(ctx context.Context, id string) (*File, error)
	FindByBotID(ctx context.Context, botID string) ([]*File, error)
	UpdateStatus(ctx context.Context, id string, status FileStatus, errMsg string) error
	Delete(ctx context.Context, id string) error
}

type Service interface {
	Upload(ctx context.Context, userID, botID string, req UploadFileRequest) (*File, error)
	GetByID(ctx context.Context, userID, botID, fileID string) (*File, error)
	GetAllByBot(ctx context.Context, userID, botID string) ([]*File, error)
	Delete(ctx context.Context, userID, botID, fileID string) error
}

type UploadFileRequest struct {
	Name      string
	Size      int64
	MimeType  string
	Content   []byte
}
