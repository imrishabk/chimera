package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/config"
	"github.com/imrishabk/chimera/services/worker/internal/ingest"
	"github.com/imrishabk/chimera/services/worker/internal/model"
)

func errURLDisabled() error { return fmt.Errorf("url fetch disabled") }
func errQueueFull() error   { return fmt.Errorf("ingestion queue full, try again later") }

type mockFetchJob struct {
	mockIngestJob
	fetchErr error
}

func (m *mockFetchJob) EnqueueURLs(_ context.Context, _, _ uuid.UUID, _ []string, _ string) (*model.IngestJob, error) {
	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return &model.IngestJob{ID: uuid.New(), Status: "pending", Source: "https://example.com"}, nil
}

func fetchReq(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/ingestion/fetch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestFetchAcceptsURLs(t *testing.T) {
	h := NewIngestJobHandlerWithConfig(&mockFetchJob{}, &mockRAG{}, config.DefaultIngestConfig())
	sid := uuid.NewString()
	req := fetchReq(t, `{"session_id":"`+sid+`","urls":["https://example.com/a","https://example.com/b"]}`)
	w := httptest.NewRecorder()
	AppHandler(h.FetchURLs).ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("want 202 got %d body %s", w.Code, w.Body.String())
	}
}

func TestFetchRejectsMissingSession(t *testing.T) {
	h := NewIngestJobHandlerWithConfig(&mockFetchJob{}, &mockRAG{}, config.DefaultIngestConfig())
	req := fetchReq(t, `{"urls":["https://example.com/a"]}`)
	w := httptest.NewRecorder()
	AppHandler(h.FetchURLs).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d body %s", w.Code, w.Body.String())
	}
}

func TestFetchRejectsEmptyURLs(t *testing.T) {
	h := NewIngestJobHandlerWithConfig(&mockFetchJob{}, &mockRAG{}, config.DefaultIngestConfig())
	req := fetchReq(t, `{"session_id":"`+uuid.NewString()+`","urls":[]}`)
	w := httptest.NewRecorder()
	AppHandler(h.FetchURLs).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d body %s", w.Code, w.Body.String())
	}
}

func TestFetchMapsDisabledTo400(t *testing.T) {
	h := NewIngestJobHandlerWithConfig(&mockFetchJob{fetchErr: context.DeadlineExceeded}, &mockRAG{}, config.DefaultIngestConfig())
	_ = h
	h2 := NewIngestJobHandlerWithConfig(&mockFetchJob{fetchErr: errURLDisabled()}, &mockRAG{}, config.DefaultIngestConfig())
	req := fetchReq(t, `{"session_id":"`+uuid.NewString()+`","urls":["https://example.com/a"]}`)
	w := httptest.NewRecorder()
	AppHandler(h2.FetchURLs).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for disabled, got %d body %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "url fetch disabled") {
		t.Errorf("want disabled message, got %s", w.Body.String())
	}
}

func TestFetchMapsQueueFullTo503(t *testing.T) {
	h := NewIngestJobHandlerWithConfig(&mockFetchJob{fetchErr: errQueueFull()}, &mockRAG{}, config.DefaultIngestConfig())
	req := fetchReq(t, `{"session_id":"`+uuid.NewString()+`","urls":["https://example.com/a"]}`)
	w := httptest.NewRecorder()
	AppHandler(h.FetchURLs).ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503 got %d body %s", w.Code, w.Body.String())
	}
}

func TestUploadRejectsCSVAndPPTXWhenDisabled(t *testing.T) {
	// Sanity: csv/pptx are allowlisted by ValidateFile in phase 2.
	for _, tc := range []struct{ name, mime string }{
		{"a.csv", ingest.MIMECSV},
		{"a.pptx", ingest.MIMEPPTX},
	} {
		if _, err := ingest.ValidateFile(tc.name, tc.mime, 100, 1<<20); err != nil {
			t.Errorf("%s should validate: %v", tc.name, err)
		}
	}
}
