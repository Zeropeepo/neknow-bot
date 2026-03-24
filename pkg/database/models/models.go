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

