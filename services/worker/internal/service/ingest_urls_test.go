package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/config"
	"github.com/imrishabk/chimera/services/worker/internal/ingest"
)

func TestEnqueueUploadDedupsIdenticalContent(t *testing.T) {
	fr := newFakeJobRepo()
	rag := &fakeRAG{}
	svc := NewIngestJobServiceWithConfig(fr, rag, config.DefaultIngestConfig())
	sid := uuid.New()
	mkfile := func() ingest.OpenedFile {
		path := filepath.Join(t.TempDir(), "same.txt")
		if err := os.WriteFile(path, []byte("identical dedup content here for testing"), 0600); err != nil {
			t.Fatal(err)
		}
		return ingest.OpenedFile{Filename: "same.txt", MIME: ingest.MIMEText, Ext: ".txt", Size: 39, Checksum: "dedup-sum-1", Path: path}
	}
	first, err := svc.EnqueueUpload(context.Background(), sid, uuid.New(), []ingest.OpenedFile{mkfile()}, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.EnqueueUpload(context.Background(), sid, uuid.New(), []ingest.OpenedFile{mkfile()}, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("want dedup same job, got %s vs %s", first.ID, second.ID)
	}
	// Different session must not dedup.
	third, err := svc.EnqueueUpload(context.Background(), uuid.New(), uuid.New(), []ingest.OpenedFile{mkfile()}, "")
	if err != nil {
		t.Fatal(err)
	}
	if third.ID == first.ID {
		t.Errorf("different session should create new job")
	}
}

func TestEnqueueURLsDisabledByDefault(t *testing.T) {
	fr := newFakeJobRepo()
	svc := NewIngestJobServiceWithConfig(fr, &fakeRAG{}, config.DefaultIngestConfig())
	if _, err := svc.EnqueueURLs(context.Background(), uuid.New(), uuid.New(), []string{"https://example.com/"}, ""); err == nil {
		t.Fatalf("want disabled error")
	} else if got := err.Error(); got != "url fetch disabled" {
		t.Fatalf("want disabled message, got %q", got)
	}
}

func TestEnqueueURLsHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><head><title>Fetched</title></head><body><p>Fetched url body content for rag ingestion test.</p></body></html>"))
	}))
	defer srv.Close()
	fr := newFakeJobRepo()
	rag := &fakeRAG{}
	cfg := config.DefaultIngestConfig()
	cfg.AllowURLFetch = true
	cfg.URLFetchAllowPrivate = true
	svc := NewIngestJobServiceWithConfig(fr, rag, cfg)
	sid := uuid.New()
	job, err := svc.EnqueueURLs(context.Background(), sid, uuid.New(), []string{srv.URL + "/article"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "pending" {
		t.Fatalf("want pending, got %s", job.Status)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		got, _ := fr.Get(context.Background(), job.ID)
		if got.Status == "completed" {
			break
		}
		if got.Status == "failed" {
			t.Fatalf("want completed, err %s", got.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("background timeout")
		}
		time.Sleep(50 * time.Millisecond)
	}
	if rag.last == nil || len(rag.last.Documents) == 0 {
		t.Fatalf("rag not called")
	}
	if got := rag.last.Documents[0].SourceType; got != "url" {
		t.Errorf("want url source_type, got %q", got)
	}
}

func TestEnqueueURLsAllFailNoJob(t *testing.T) {
	fr := newFakeJobRepo()
	cfg := config.DefaultIngestConfig()
	cfg.AllowURLFetch = true
	svc := NewIngestJobServiceWithConfig(fr, &fakeRAG{}, cfg)
	// 127.0.0.1 is SSRF-blocked even when fetch is enabled.
	if _, err := svc.EnqueueURLs(context.Background(), uuid.New(), uuid.New(), []string{"http://127.0.0.1/"}, ""); err == nil {
		t.Fatalf("want all-failed error")
	}
}
