package model

import "github.com/google/uuid"

type RegisterRequest struct {
	Username string `json:"username" validate:"required,username"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,password"`
}

type UpdateUserRequest struct {
	Username    string `json:"username" validate:"omitempty,username"`
	Email       string `json:"email" validate:"omitempty,email"`
	OldPassword string `json:"old_password" validate:"omitempty"`
	NewPassword string `json:"new_password" validate:"omitempty,password"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type ChatRequest struct {
	SessionID uuid.UUID `json:"session_id" validate:"required"`
	Text      string    `json:"input" validate:"required"`
}

type IngestRequest struct {
	SessionID uuid.UUID  `json:"session_id" validate:"required"`
	Documents []Document `json:"documents" validate:"omitempty,min=1,dive"`
	Content   string     `json:"content" validate:"required_without=Documents"`
	Source    string     `json:"source"`
}

type Document struct {
	Content    string            `json:"content" validate:"required"`
	Source     string            `json:"source"`
	SourceType string            `json:"source_type" validate:"omitempty,oneof=url pdf text markdown api"`
	DocID      string            `json:"doc_id"`
	ChunkIndex int32             `json:"chunk_index"`
	PageTitle  string            `json:"page_title"`
	Chapter    string            `json:"chapter"`
	Metadata   map[string]string `json:"metadata"`
}

type QueryRequest struct {
	SessionID uuid.UUID         `json:"session_id" validate:"required"`
	Query     string            `json:"query" validate:"required"`
	K         int32             `json:"k" validate:"omitempty,min=1,max=20"`
	Filter    map[string]string `json:"filter"`
}
