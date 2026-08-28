package handler

import (
	"encoding/json"
	"net/http"

	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/service"
	"github.com/imrishabk/chimera/services/worker/internal/validator"
)

type IngestHandler struct {
	svc service.RAGService
}

func NewIngestHandler(svc service.RAGService) *IngestHandler {
	return &IngestHandler{svc: svc}
}

func (h *IngestHandler) Push(w http.ResponseWriter, r *http.Request) {
	var req model.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"invalid request body"}`))
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	resp, err := h.svc.IngestDocuments(r.Context(), &req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": resp})
}
