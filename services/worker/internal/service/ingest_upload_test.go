package service

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	aicorepb "github.com/imrishabk/chimera/services/worker/internal/grpc"
	"github.com/imrishabk/chimera/services/worker/internal/config"
	appErrs "github.com/imrishabk/chimera/services/worker/internal/errors"
	"github.com/imrishabk/chimera/services/worker/internal/ingest"
	"github.com/imrishabk/chimera/services/worker/internal/model"
)

type fakeJobRepo struct {
	mu   sync.Mutex
	jobs map[uuid.UUID]*model.IngestJob
}

func newFakeJobRepo() *fakeJobRepo { return &fakeJobRepo{jobs: map[uuid.UUID]*model.IngestJob{}} }

func (f *fakeJobRepo) Create(_ context.Context, job *model.IngestJob) (*model.IngestJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job.ID = uuid.New()
	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now
	cp := *job
	f.jobs[job.ID] = &cp
	return &cp, nil
}
func (f *fakeJobRepo) Get(_ context.Context, id uuid.UUID) (*model.IngestJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	j, ok := f.jobs[id]
	if !ok {
		return nil, appErrs.ErrIngestionJobNotFound
	}
	cp := *j
	return &cp, nil
}
func (f *fakeJobRepo) ListBySession(_ context.Context, sid uuid.UUID, limit, offset int) ([]model.IngestJob, error) {
	return nil, nil
}
func (f *fakeJobRepo) FindBySessionChecksum(_ context.Context, sid uuid.UUID, sum string) (*model.IngestJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, j := range f.jobs {
		if j.SessionID == sid && j.Checksum != nil && *j.Checksum == sum && j.Status != "failed" {
			cp := *j
			return &cp, nil
		}
	}
	return nil, appErrs.ErrIngestionJobNotFound
}
func (f *fakeJobRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string, n int, em string) (*model.IngestJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	j, ok := f.jobs[id]
	if !ok {
		return nil, appErrs.ErrIngestionJobNotFound
	}
	j.Status = status
	j.DocCount = n
	j.Error = em
	j.UpdatedAt = time.Now()
	cp := *j
	return &cp, nil
}

type fakeRAG struct {
	mu   sync.Mutex
	last *model.IngestRequest
}

func (f *fakeRAG) IngestDocuments(_ context.Context, req *model.IngestRequest) (*aicorepb.IngestResponse, error) {
	f.mu.Lock()
	f.last = req
	f.mu.Unlock()
	return &aicorepb.IngestResponse{Success: true, Count: int32(len(req.Documents))}, nil
}
func (f *fakeRAG) QueryRAG(_ context.Context, req *model.QueryRequest) (*aicorepb.QueryResponse, error) {
	return &aicorepb.QueryResponse{Answer: "ok"}, nil
}

func stageFile(t *testing.T, name string, data []byte) ingest.OpenedFile {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	ext := ".txt"
	mime := ingest.MIMEText
	if len(name) > 5 && name[len(name)-5:] == ".docx" {
		ext = ".docx"
		mime = ingest.MIMEDOCX
	}
	return ingest.OpenedFile{Filename: name, MIME: mime, Ext: ext, Size: int64(len(data)), Checksum: "test", Path: path}
}

func TestEnqueueUploadCompletes(t *testing.T) {
	fr := newFakeJobRepo()
	rag := &fakeRAG{}
	svc := NewIngestJobServiceWithConfig(fr, rag, config.DefaultIngestConfig())
	f := stageFile(t, "notes.txt", []byte("hello world upload background test content here"))
	sid := uuid.New()
	job, err := svc.EnqueueUpload(context.Background(), sid, uuid.New(), []ingest.OpenedFile{f}, "")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "pending" {
		t.Fatalf("want pending, got %s", job.Status)
	}
	// Wait for background (timeout 5s).
	deadline := time.Now().Add(5 * time.Second)
	for {
		got, _ := fr.Get(context.Background(), job.ID)
		if got.Status == "completed" || got.Status == "failed" {
			if got.Status != "completed" {
				t.Fatalf("want completed, got %s err %s", got.Status, got.Error)
			}
			if got.DocCount == 0 {
				t.Errorf("want doc_count > 0")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("background did not complete, last status %s", got.Status)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if rag.last == nil || len(rag.last.Documents) == 0 {
		t.Fatalf("rag not called")
	}
	d := rag.last.Documents[0]
	if d.SourceType != "text" && d.SourceType != "markdown" {
		t.Errorf("unexpected source_type %q", d.SourceType)
	}
	if d.DocID == "" || d.Content == "" {
		t.Errorf("DocID/Content required: %+v", d)
	}
	if _, err := os.Stat(f.Path); !os.IsNotExist(err) {
		t.Errorf("staged temp not cleaned: %s", f.Path)
	}
}

func TestEnqueueUploadAllFail(t *testing.T) {
	fr := newFakeJobRepo()
	rag := &fakeRAG{}
	svc := NewIngestJobServiceWithConfig(fr, rag, config.DefaultIngestConfig())
	f := stageFile(t, "empty.txt", []byte{})
	sid := uuid.New()
	job, err := svc.EnqueueUpload(context.Background(), sid, uuid.New(), []ingest.OpenedFile{f}, "")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		got, _ := fr.Get(context.Background(), job.ID)
		if got.Status == "failed" {
			if got.Error == "" {
				t.Errorf("want error message on failed job")
			}
			return
		}
		if got.Status == "completed" {
			t.Fatalf("want failed for empty file")
		}
		if time.Now().After(deadline) {
			t.Fatalf("background did not finish")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
