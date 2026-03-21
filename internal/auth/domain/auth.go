package domain

import (
	"context"
	"time"
)

type User struct {
	ID				string
	Email			string
	PasswordHash	string
	CreatedAt		time.Time
	UpdatedAt		time.Time
}

type TokenClaims struct {
	UserID 			string
	Email			string
}

type Repository interface {
	Create(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string)
	FindByID(ctx context.Context, id string) (*User, error)
}	

type Service interface{
	Register(ctx context.Context, email, password string) (*User, error)
	Login(ctx context.Context, email, password string) (accessToken string, 
		refreshToken string, err error)
	RefreshToken(ctx context.Context, refreshToken string) (newAccessToken string, err error)
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
}