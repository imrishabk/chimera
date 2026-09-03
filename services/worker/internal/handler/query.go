package handler

import (
	"encoding/json"
	"net/http"

	"github.com/imrishabk/chimera/services/worker/internal/errors"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/service"
	"github.com/imrishabk/chimera/services/worker/internal/validator"
)

type QueryHandler struct {
	svc service.RAGService
}

func NewQueryHandler(svc service.RAGService) *QueryHandler {
	return &QueryHandler{svc: svc}
}

func (h *QueryHandler) Submit(w http.ResponseWriter, r *http.Request) error {
	var req model.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return &errors.HandlerError{Status: http.StatusBadRequest, Message: "invalid body"}
	}
	if err := validator.Validate.Struct(req); err != nil {
		return err
	}
	resp, err := h.svc.QueryRAG(r.Context(), &req)
	if err != nil {
		return err
	}
	return writeJSONData(w, http.StatusOK, resp)
}
