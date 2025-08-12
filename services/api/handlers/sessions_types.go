package handlers

type startSessionRequest struct {
	DeviceID  string  `json:"device_id"`
	BookTitle string  `json:"book_title"`
	Author    *string `json:"author,omitempty"`
	Source    *string `json:"source,omitempty"`
	StartPage int     `json:"start_page"`
	StartedAt *string `json:"started_at,omitempty"`
}

type stopSessionRequest struct {
	DeviceID string  `json:"device_id"`
	EndPage  *int    `json:"end_page,omitempty"`
	EndedAt  *string `json:"ended_at,omitempty"`
}

type continueSessionRequest struct {
	DeviceID  string  `json:"device_id"`
	StartedAt *string `json:"started_at,omitempty"`
}

type sessionResponse struct {
	ID              int64   `json:"id"`
	BookID          int64   `json:"book_id"`
	DeviceID        string  `json:"device_id"`
	StartPage       int     `json:"start_page"`
	EndPage         *int    `json:"end_page,omitempty"`
	StartedAt       string  `json:"started_at"`
	EndedAt         *string `json:"ended_at,omitempty"`
	DurationSeconds *int64  `json:"duration_seconds,omitempty"`
	CreatedAt       string  `json:"created_at"`
	BookTitle       string  `json:"book_title"`
	Author          *string `json:"author,omitempty"`
	Source          *string `json:"source,omitempty"`
}
