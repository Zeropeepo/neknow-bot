package dto

import "time"


// Request
type CreateBotRequest struct {
	Name         string `json:"name"          validate:"required,min=1,max=100"`
	Description  string `json:"description"   validate:"max=500"`
	SystemPrompt string `json:"system_prompt" validate:"required,min=10,max=2000"`
	Model        string `json:"model"         validate:"required"`
	IsPublic     bool   `json:"is_public"`
}

type UpdateBotRequest struct {
	Name         *string `json:"name"          validate:"omitempty,min=1,max=100"`
	Description  *string `json:"description"   validate:"omitempty,max=500"`
	SystemPrompt *string `json:"system_prompt" validate:"omitempty,min=10,max=2000"`
	Model        *string `json:"model"         validate:"omitempty"`
	IsPublic     *bool   `json:"is_public"`
	Status       *string `json:"status"        validate:"omitempty,oneof=active inactive"`
}


// Response
type BotResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SystemPrompt string    `json:"system_prompt"`
	Model        string    `json:"model"`
	IsPublic     bool      `json:"is_public"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type BotListResponse struct {
	Bots  []*BotResponse `json:"bots"`
	Total int            `json:"total"`
}
