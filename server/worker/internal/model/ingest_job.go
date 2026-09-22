package model

import (
	"time"

	"github.com/google/uuid"
)

type IngestJob struct {
	ID         uuid.UUID `json:"id" db:"id"`
	SessionID  uuid.UUID `json:"session_id" db:"session_id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	Status     string    `json:"status" db:"status"`
	Source     string    `json:"source" db:"source"`
	SourceType string    `json:"source_type" db:"source_type"`
	DocCount   int       `json:"doc_count" db:"doc_count"`
	Error      string    `json:"error" db:"error"`
	FileName   *string   `json:"file_name,omitempty" db:"file_name"`
	MimeType   *string   `json:"mime_type,omitempty" db:"mime_type"`
	FileSize   *int64    `json:"file_size,omitempty" db:"file_size"`
	Checksum   *string   `json:"checksum,omitempty" db:"checksum"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
