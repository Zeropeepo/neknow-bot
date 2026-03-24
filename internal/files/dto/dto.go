package dto

import "time"

type FileResponse struct {
	ID        string     `json:"id"`
	BotID     string     `json:"bot_id"`
	Name      string     `json:"name"`
	Size      int64      `json:"size"`
	MimeType  string     `json:"mime_type"`
	Status    string     `json:"status"`
	ErrorMsg  string     `json:"error_msg,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type FileListResponse struct {
	Files []*FileResponse `json:"files"`
	Total int             `json:"total"`
}
