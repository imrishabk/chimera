package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/imrishabk/chimera/services/worker/internal/errors"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/service"
	"github.com/imrishabk/chimera/services/worker/internal/validator"
)

type IngestHandler struct {
	rag service.RAGService
	job service.IngestJobService
}

func NewIngestHandler(rag service.RAGService) *IngestHandler {
	return &IngestHandler{rag: rag}
}

func NewIngestJobHandler(job service.IngestJobService, rag service.RAGService) *IngestHandler {
	return &IngestHandler{rag: rag, job: job}
}

func (h *IngestHandler) Push(w http.ResponseWriter, r *http.Request) error {
	var req model.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return &errors.HandlerError{Status: http.StatusBadRequest, Message: "invalid body"}
	}
	if err := validator.Validate.Struct(req); err != nil {
		return err
	}
	// if job tracking available, use it; else fallback to direct RAG
	if h.job != nil {
		userID := extractUserID(r)
		job, err := h.job.CreateJob(r.Context(), &req, userID)
		if err != nil {
			return err
		}
		return writeJSONData(w, http.StatusOK, job)
	}
	resp, err := h.rag.IngestDocuments(r.Context(), &req)
	if err != nil {
		return &errors.HandlerError{Status: http.StatusBadGateway, Message: err.Error()}
	}
	return writeJSONData(w, http.StatusOK, resp)
}

func (h *IngestHandler) Get(w http.ResponseWriter, r *http.Request) error {
	if h.job == nil {
		return &errors.HandlerError{Status: http.StatusServiceUnavailable, Message: "ingest job tracking not available"}
	}
	idStr := r.PathValue("jobId")
	if idStr == "" {
		idStr = r.URL.Query().Get("jobId")
	}
	// chi fallback
	if idStr == "" {
		// try chi URLParam via context
		if v := r.Context().Value("jobId"); v != nil {
			idStr = v.(string)
		}
	}
	uid, err := parseUUID(idStr, r)
	if err != nil {
		return &errors.HandlerError{Status: http.StatusBadRequest, Message: "invalid parameter jobId"}
	}
	job, err := h.job.GetJob(r.Context(), uid)
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, job)
}

func extractUserID(r *http.Request) uuid.UUID {
	if v := r.Header.Get("X-User-Id"); v != "" {
		if uid, err := uuid.Parse(v); err == nil {
			return uid
		}
	}
	if v := r.Context().Value("userID"); v != nil {
		if uid, ok := v.(uuid.UUID); ok {
			return uid
		}
		if s, ok := v.(string); ok {
			if uid, err := uuid.Parse(s); err == nil {
				return uid
			}
		}
	}
	return uuid.Nil
}

func (h *IngestHandler) List(w http.ResponseWriter, r *http.Request) error {
	if h.job == nil {
		return &errors.HandlerError{Status: http.StatusServiceUnavailable, Message: "ingest job tracking not available"}
	}
	sidStr := r.URL.Query().Get("session_id")
	if sidStr == "" {
		sidStr = r.PathValue("sessionId")
	}
	// chi URLParam fallback for list route
	if sidStr == "" {
		sidStr = r.URL.Query().Get("sessionId")
	}
	uid, err := parseUUID(sidStr, r)
	if err != nil {
		return &errors.HandlerError{Status: http.StatusBadRequest, Message: "invalid sessionId"}
	}
	jobs, err := h.job.ListJobs(r.Context(), uid, 20, 0)
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, jobs)
}
