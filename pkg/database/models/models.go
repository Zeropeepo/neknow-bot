package models

import "time"

type User struct {
	ID           string `gorm:"primaryKey"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
func (User) TableName() string { return "users" }


type Bot struct {
	ID           string `gorm:"primaryKey"`
	UserID       string `gorm:"not null;index"`
	Name         string `gorm:"not null"`
	Description  string
	SystemPrompt string `gorm:"not null"`
	Model        string `gorm:"not null"`
	IsPublic     bool `gorm:"default:false"`
	Status       string `gorm:"not null;default:'active'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Bot) TableName() string { return "bots" }

type File struct {
	ID        string `gorm:"primaryKey"`
	BotID     string `gorm:"not null;index"`
	UserID    string `gorm:"not null;index"`
	Name      string `gorm:"not null"`
	Size      int64
	MimeType  string
	Bucket    string `gorm:"not null"`
	ObjectKey string `gorm:"not null"`
	Status    string `gorm:"not null;default:'pending'"`
	ErrorMsg  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (File) TableName() string { return "files" }

type Conversation struct {
	ID        string `gorm:"primaryKey"`
	BotID     string `gorm:"not null;index"`
	UserID    string `gorm:"not null;index"`
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Conversation) TableName() string { return "conversations" }

type Message struct {
	ID             string `gorm:"primaryKey"`
	ConversationID string `gorm:"not null;index"`
	Role           string `gorm:"not null"`
	Content        string `gorm:"not null;type:text"`
	CreatedAt      time.Time
}

func (Message) TableName() string { return "messages" }
