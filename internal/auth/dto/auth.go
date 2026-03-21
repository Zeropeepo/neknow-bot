package dto

import(
	"time"
)

type RegisterRequest struct {
	Email		string	`json:"email" validate:"required,email"`
	Password	string	`json:"password" validate:"required,min=8,max=60"`
}

type LoginRequest struct{
	Email		string	`json:"email" validate:"required,email"`
	Password	string	`json:"password" validate:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}


// Response

type UserResponse struct {
	ID				string		`json:"id"`
	Email			string		`json:"email"`
	CreatedAt		time.Time	`json:"created_at"`
}

type TokenResponse struct {
	AccessToken		string		`json:"access_token"`
	RefreshToken	string 		`json:"refresh_token"`
	TokenType		string		`json:"token_type"`
}

type RefreshResponse struct {
	AccessToken		string		`json:"access_token"`
	TokenType		string		`json:"token_type"`
}