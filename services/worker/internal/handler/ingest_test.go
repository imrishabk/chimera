package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	aicorepb "github.com/imrishabk/chimera/services/worker/internal/grpc"
	"github.com/imrishabk/chimera/services/worker/internal/model"
)

type mockRAG struct {
	ingest func(ctx context.Context, req *model.IngestRequest) (*aicorepb.IngestResponse, error)
	query  func(ctx context.Context, req *model.QueryRequest) (*aicorepb.QueryResponse, error)
}

func (m *mockRAG) IngestDocuments(ctx context.Context, req *model.IngestRequest) (*aicorepb.IngestResponse, error) {
	if m.ingest != nil {
		return m.ingest(ctx, req)
	}
	return &aicorepb.IngestResponse{Success: true, Count: 1}, nil
}
func (m *mockRAG) QueryRAG(ctx context.Context, req *model.QueryRequest) (*aicorepb.QueryResponse, error) {
	if m.query != nil {
		return m.query(ctx, req)
	}
	return &aicorepb.QueryResponse{Answer: "ok"}, nil
}

func TestIngestPushInvalidJSON(t *testing.T) {
	h := NewIngestHandler(&mockRAG{})
	req := httptest.NewRequest(http.MethodPost, "/ingestion", strings.NewReader("{bad"))
	w := httptest.NewRecorder()
	h.Push(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d", w.Code)
	}
}

func TestIngestPushValidationFail(t *testing.T) {
	h := NewIngestHandler(&mockRAG{})
	req := httptest.NewRequest(http.MethodPost, "/ingestion", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.Push(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 got %d", w.Code)
	}
}

func TestIngestPushUnavailable(t *testing.T) {
	h := &IngestHandler{rag: &mockRAG{ingest: func(ctx context.Context, req *model.IngestRequest) (*aicorepb.IngestResponse, error) {
		return nil, context.DeadlineExceeded
	}}}
	req := httptest.NewRequest(http.MethodPost, "/ingestion", strings.NewReader(`{"session_id":"00000000-0000-0000-0000-000000000001","content":"hi"}`))
	w := httptest.NewRecorder()
	h.Push(w, req)
	if w.Code != http.StatusBadGateway {
		t.Errorf("want 502 got %d", w.Code)
	}
}
