package model

import "github.com/google/uuid"

type RegisterRequest struct {
	Username string `json:"username" validate:"required,username"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ChatRequest struct {
	SessionID uuid.UUID `json:"session_id"`
	Text      string    `json:"input"`
}
