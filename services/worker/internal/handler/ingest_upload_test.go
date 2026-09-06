package handler

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/config"
	"github.com/imrishabk/chimera/services/worker/internal/ingest"
	"github.com/imrishabk/chimera/services/worker/internal/model"
)

type mockIngestJob struct {
	enqueue func(ctx context.Context, sid, uid uuid.UUID, files []ingest.OpenedFile, title string) (*model.IngestJob, error)
}

func (m *mockIngestJob) CreateJob(ctx context.Context, req *model.IngestRequest, uid uuid.UUID) (*model.IngestJob, error) {
	return &model.IngestJob{ID: uuid.New(), Status: "pending"}, nil
}
func (m *mockIngestJob) EnqueueUpload(ctx context.Context, sid, uid uuid.UUID, files []ingest.OpenedFile, title string) (*model.IngestJob, error) {
	if m.enqueue != nil {
		return m.enqueue(ctx, sid, uid, files, title)
	}
	return &model.IngestJob{ID: uuid.New(), SessionID: sid, Status: "pending", Source: files[0].Filename}, nil
}
func (m *mockIngestJob) EnqueueURLs(ctx context.Context, sid, uid uuid.UUID, urls []string, title string) (*model.IngestJob, error) {
	return &model.IngestJob{ID: uuid.New(), SessionID: sid, Status: "pending", Source: urls[0]}, nil
}
func (m *mockIngestJob) GetJob(ctx context.Context, id uuid.UUID) (*model.IngestJob, error) {
	return &model.IngestJob{ID: id}, nil
}
func (m *mockIngestJob) ListJobs(ctx context.Context, sid uuid.UUID, limit, offset int) ([]model.IngestJob, error) {
	return nil, nil
}

func multipartUpload(t *testing.T, sessionID string, files map[string][]byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if sessionID != "" {
		_ = mw.WriteField("session_id", sessionID)
	}
	for name, data := range files {
		fw, err := mw.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/ingestion/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestUploadAcceptsTxt(t *testing.T) {
	jobSvc := &mockIngestJob{}
	h := NewIngestJobHandlerWithConfig(jobSvc, &mockRAG{}, config.DefaultIngestConfig())
	sid := uuid.NewString()
	req := multipartUpload(t, sid, map[string][]byte{"notes.txt": []byte("hello world from upload test content")})
	w := httptest.NewRecorder()
	AppHandler(h.Upload).ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("want 202 got %d body %s", w.Code, w.Body.String())
	}
	// Staged temp files transfer ownership to service; mock does not clean.
	// Ensure no leak in handler path on success is service responsibility.
}

func TestUploadAcceptsCsvAndHtml(t *testing.T) {
	jobSvc := &mockIngestJob{}
	h := NewIngestJobHandlerWithConfig(jobSvc, &mockRAG{}, config.DefaultIngestConfig())
	sid := uuid.NewString()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("session_id", sid)
	for name, data := range map[string]string{
		"data.csv":  "name,value\nalpha,1\n",
		"page.html": "<html><body><p>Hello html upload content here.</p></body></html>",
	} {
		fw, err := mw.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/ingestion/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	AppHandler(h.Upload).ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("want 202 got %d body %s", w.Code, w.Body.String())
	}
}

func TestUploadRejectsMissingSession(t *testing.T) {
	h := NewIngestJobHandlerWithConfig(&mockIngestJob{}, &mockRAG{}, config.DefaultIngestConfig())
	req := multipartUpload(t, "", map[string][]byte{"a.txt": []byte("hello world content here")})
	w := httptest.NewRecorder()
	AppHandler(h.Upload).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d body %s", w.Code, w.Body.String())
	}
}

func TestUploadRejectsNoFiles(t *testing.T) {
	h := NewIngestJobHandlerWithConfig(&mockIngestJob{}, &mockRAG{}, config.DefaultIngestConfig())
	req := multipartUpload(t, uuid.NewString(), nil)
	w := httptest.NewRecorder()
	AppHandler(h.Upload).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d body %s", w.Code, w.Body.String())
	}
}

func TestUploadRejectsUnsupportedExt(t *testing.T) {
	h := NewIngestJobHandlerWithConfig(&mockIngestJob{}, &mockRAG{}, config.DefaultIngestConfig())
	req := multipartUpload(t, uuid.NewString(), map[string][]byte{"evil.exe": []byte("MZ fake binary content here........")})
	w := httptest.NewRecorder()
	AppHandler(h.Upload).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d body %s", w.Code, w.Body.String())
	}
}

func TestUploadCleansTempOnValidationError(t *testing.T) {
	before, _ := os.ReadDir(os.TempDir())
	h := NewIngestJobHandlerWithConfig(&mockIngestJob{}, &mockRAG{}, config.DefaultIngestConfig())
	req := multipartUpload(t, "", map[string][]byte{"a.txt": []byte("hello world content here")})
	w := httptest.NewRecorder()
	AppHandler(h.Upload).ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 got %d", w.Code)
	}
	time.Sleep(50 * time.Millisecond)
	after, _ := os.ReadDir(os.TempDir())
	if len(after) > len(before)+2 {
		t.Errorf("possible temp leak: before %d after %d", len(before), len(after))
	}
}
