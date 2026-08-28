package handler

import (
	"encoding/json"
	"net/http"

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

func (h *QueryHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var req model.QueryRequest
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
	resp, err := h.svc.QueryRAG(r.Context(), &req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"success":false,"error":"` + err.Error() + `"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": resp})
}
