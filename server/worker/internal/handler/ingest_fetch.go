package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	appErrs "github.com/imrishabk/chimera/services/worker/internal/errors"
)

// FetchRequest enqueues server-side URL fetching + ingestion.
// Requires ALLOW_URL_FETCH=true, else 400.
type FetchRequest struct {
	SessionID uuid.UUID `json:"session_id"`
	URLs      []string  `json:"urls"`
	PageTitle string    `json:"page_title"`
}

// FetchURLs handles POST /ingestion/fetch.
func (h *IngestHandler) FetchURLs(w http.ResponseWriter, r *http.Request) error {
	if h.job == nil {
		return appErrs.ErrIngestionSeviceUnavailable
	}
	var req FetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return appErrs.ErrInvalidBody
	}
	if req.SessionID == uuid.Nil {
		return appErrs.ErrInvalidSession
	}
	if len(req.URLs) == 0 {
		return appErrs.ErrInvalidUpload
	}
	maxURLs := h.cfg.MaxURLsPerRequest
	if maxURLs <= 0 {
		maxURLs = 10
	}
	if len(req.URLs) > maxURLs {
		return appErrs.ErrTooManyFiles
	}
	for _, u := range req.URLs {
		if len(strings.TrimSpace(u)) == 0 || len(u) > 2048 {
			return appErrs.ErrInvalidUpload
		}
	}
	userID := extractUserID(r)
	job, err := h.job.EnqueueURLs(r.Context(), req.SessionID, userID, req.URLs, strings.TrimSpace(req.PageTitle))
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "queue full"):
			return appErrs.ErrUploadQueueFull
		case strings.Contains(msg, "disabled"):
			return appErrs.ErrURLFetchDisabled
		default:
			return err
		}
	}
	return writeJSONData(w, http.StatusAccepted, job)
}
