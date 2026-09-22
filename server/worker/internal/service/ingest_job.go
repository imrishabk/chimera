package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"charm.land/log/v2"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/config"
	"github.com/imrishabk/chimera/services/worker/internal/ingest"
	"github.com/imrishabk/chimera/services/worker/internal/model"
	"github.com/imrishabk/chimera/services/worker/internal/repo"
)

type IngestJobService interface {
	CreateJob(ctx context.Context, req *model.IngestRequest, userID uuid.UUID) (*model.IngestJob, error)
	// EnqueueUpload creates a pending job and processes files in background.
	// Ownership of staged temp files transfers to the service; they are
	// removed after processing. Returns the pending job immediately.
	// Identical content (session_id + checksum) returns the existing
	// non-failed job without re-ingesting.
	EnqueueUpload(ctx context.Context, sessionID, userID uuid.UUID, files []ingest.OpenedFile, pageTitle string) (*model.IngestJob, error)
	// EnqueueURLs fetches URLs server-side (SSRF-guarded), stages them, and
	// enqueues like EnqueueUpload. Requires ALLOW_URL_FETCH=true.
	EnqueueURLs(ctx context.Context, sessionID, userID uuid.UUID, urls []string, pageTitle string) (*model.IngestJob, error)
	GetJob(ctx context.Context, id uuid.UUID) (*model.IngestJob, error)
	ListJobs(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]model.IngestJob, error)
}

type ingestJobService struct {
	jobs repo.IngestJobRepository
	rag  RAGService
	cfg  config.IngestConfig
	sem  chan struct{}
}

func NewIngestJobService(jobs repo.IngestJobRepository, rag RAGService) IngestJobService {
	return NewIngestJobServiceWithConfig(jobs, rag, config.LoadIngestConfig())
}

func NewIngestJobServiceWithConfig(jobs repo.IngestJobRepository, rag RAGService, cfg config.IngestConfig) IngestJobService {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}
	if cfg.WorkerTimeoutMin <= 0 {
		cfg.WorkerTimeoutMin = 10
	}
	if cfg.MaxPagesPDF <= 0 {
		cfg.MaxPagesPDF = 500
	}
	if cfg.MaxCharsPerFile <= 0 {
		cfg.MaxCharsPerFile = 1000000
	}
	return &ingestJobService{jobs: jobs, rag: rag, cfg: cfg, sem: make(chan struct{}, cfg.QueueSize)}
}

func (s *ingestJobService) CreateJob(ctx context.Context, req *model.IngestRequest, userID uuid.UUID) (*model.IngestJob, error) {
	source := req.Source
	if source == "" && len(req.Documents) > 0 {
		source = req.Documents[0].Source
	}
	sourceType := ""
	if len(req.Documents) > 0 {
		sourceType = req.Documents[0].SourceType
	}
	job := &model.IngestJob{
		SessionID:  req.SessionID,
		UserID:     userID,
		Status:     "pending",
		Source:     source,
		SourceType: sourceType,
		DocCount:   len(req.Documents),
	}
	created, err := s.jobs.Create(ctx, job)
	if err != nil {
		return nil, err
	}
	// mark processing
	if _, err := s.jobs.UpdateStatus(ctx, created.ID, "processing", created.DocCount, ""); err != nil {
		return nil, err
	}
	if s.rag == nil {
		failed, _ := s.jobs.UpdateStatus(ctx, created.ID, "failed", 0, "AI Core unavailable — gRPC client not connected")
		return failed, nil
	}
	resp, err := s.rag.IngestDocuments(ctx, req)
	if err != nil {
		failed, _ := s.jobs.UpdateStatus(ctx, created.ID, "failed", 0, err.Error())
		return failed, err
	}
	return s.jobs.UpdateStatus(ctx, created.ID, "completed", int(resp.Count), resp.Error)
}

func (s *ingestJobService) EnqueueUpload(ctx context.Context, sessionID, userID uuid.UUID, files []ingest.OpenedFile, pageTitle string) (*model.IngestJob, error) {
	if sessionID == uuid.Nil || len(files) == 0 {
		cleanupStaged(files)
		return nil, fmt.Errorf("session_id and at least one file are required")
	}
	sum := batchChecksum(files)
	if existing, err := s.jobs.FindBySessionChecksum(ctx, sessionID, sum); err == nil {
		cleanupStaged(files)
		log.Info("ingest: dedup hit", "job_id", existing.ID.String(), "session_id", sessionID.String())
		return existing, nil
	}
	source := files[0].Filename
	var total int64
	for _, f := range files {
		total += f.Size
	}
	mime := files[0].MIME
	job := &model.IngestJob{
		SessionID:  sessionID,
		UserID:     userID,
		Status:     "pending",
		Source:     source,
		SourceType: "markdown",
		DocCount:   len(files),
		FileName:   &source,
		MimeType:   &mime,
		FileSize:   &total,
		Checksum:   &sum,
	}
	created, err := s.jobs.Create(ctx, job)
	if err != nil {
		if isDuplicateJob(err) {
			if existing, ferr := s.jobs.FindBySessionChecksum(ctx, sessionID, sum); ferr == nil {
				cleanupStaged(files)
				return existing, nil
			}
		}
		cleanupStaged(files)
		return nil, err
	}
	select {
	case s.sem <- struct{}{}:
	default:
		cleanupStaged(files)
		_, _ = s.jobs.UpdateStatus(context.Background(), created.ID, "failed", 0, "ingestion queue full, try again later")
		return nil, fmt.Errorf("ingestion queue full, try again later")
	}
	go s.processUpload(created.ID, sessionID, files, pageTitle)
	return created, nil
}

// EnqueueURLs fetches each URL with SSRF guards, stages successes to temp
// files, and enqueues a single background job. Per-URL fetch failures are
// collected; if every URL fails no job is created.
func (s *ingestJobService) EnqueueURLs(ctx context.Context, sessionID, userID uuid.UUID, urls []string, pageTitle string) (*model.IngestJob, error) {
	if !s.cfg.AllowURLFetch {
		return nil, fmt.Errorf("url fetch disabled")
	}
	maxURLs := s.cfg.MaxURLsPerRequest
	if maxURLs <= 0 {
		maxURLs = 10
	}
	urls = dedupeURLs(urls)
	if sessionID == uuid.Nil || len(urls) == 0 {
		return nil, fmt.Errorf("session_id and at least one url are required")
	}
	if len(urls) > maxURLs {
		return nil, fmt.Errorf("too many urls (max %d)", maxURLs)
	}
	timeout := s.cfg.URLFetchTimeoutSec
	if timeout <= 0 {
		timeout = 10
	}
	maxMB := s.cfg.URLFetchMaxMB
	if maxMB <= 0 {
		maxMB = 5
	}
	var staged []ingest.OpenedFile
	var fetchErrs []string
	for _, raw := range urls {
		if len(raw) > 2048 {
			fetchErrs = append(fetchErrs, raw+": url too long")
			continue
		}
		fetched, err := ingest.FetchURL(ctx, raw, ingest.FetchOptions{
			Timeout:      time.Duration(timeout) * time.Second,
			MaxBytes:     maxMB * 1024 * 1024,
			AllowPrivate: s.cfg.URLFetchAllowPrivate,
		})
		if err != nil {
			fetchErrs = append(fetchErrs, err.Error())
			log.Error("ingest: url fetch failed", "url", raw, "error", err)
			continue
		}
		filename, mime, ext := urlStagedName(fetched)
		tmp, err := os.CreateTemp("", "chimera-url-*")
		if err != nil {
			fetchErrs = append(fetchErrs, raw+": stage failed")
			continue
		}
		sum, n, werr := writeStaged(tmp, fetched.Body)
		_ = tmp.Close()
		if werr != nil {
			_ = os.Remove(tmp.Name())
			fetchErrs = append(fetchErrs, raw+": stage failed")
			continue
		}
		if _, err := ingest.ValidateFile(filename, mime, n, s.cfg.MaxBytesPerFile()); err != nil {
			_ = os.Remove(tmp.Name())
			fetchErrs = append(fetchErrs, raw+": unsupported content")
			continue
		}
		staged = append(staged, ingest.OpenedFile{
			Filename: filename, MIME: mime, Ext: ext, Size: n, Checksum: sum, Path: tmp.Name(),
		})
	}
	if len(staged) == 0 {
		msg := "all urls failed"
		if len(fetchErrs) > 0 {
			msg = strings.Join(fetchErrs, "; ")
		}
		return nil, fmt.Errorf("%s", msg)
	}
	job, err := s.EnqueueUpload(ctx, sessionID, userID, staged, pageTitle)
	if err != nil {
		return nil, err
	}
	if len(fetchErrs) > 0 {
		// Record partial fetch failures on the job without failing it;
		// extraction may still succeed for staged URLs.
		_, _ = s.jobs.UpdateStatus(ctx, job.ID, job.Status, job.DocCount,
			"partial url failures: "+strings.Join(fetchErrs, "; "))
		job.Error = "partial url failures: " + strings.Join(fetchErrs, "; ")
	}
	return job, nil
}

// batchChecksum identifies identical content: single file uses its sha256,
// multi-file batches hash the sorted per-file checksums.
func batchChecksum(files []ingest.OpenedFile) string {
	if len(files) == 1 {
		return files[0].Checksum
	}
	sums := make([]string, 0, len(files))
	for _, f := range files {
		sums = append(sums, f.Checksum)
	}
	sort.Strings(sums)
	h := sha256.Sum256([]byte(strings.Join(sums, ",")))
	return hex.EncodeToString(h[:])
}

func isDuplicateJob(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already ingested")
}

// dedupeURLs trims, drops empties and duplicates preserving order.
func dedupeURLs(urls []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out
}

// urlStagedName derives a safe filename + mime/ext for fetched content.
func urlStagedName(f *ingest.FetchedURL) (filename, mime, ext string) {
	base := f.FinalURL
	if base == "" {
		base = f.URL
	}
	u, err := url.Parse(base)
	name := "fetched.html"
	if err == nil {
		if bn := path.Base(strings.TrimSuffix(u.Path, "/")); bn != "" && bn != "." && bn != "/" {
			name = bn
		} else if u.Host != "" {
			name = u.Host + ".html"
		}
	}
	ct := strings.ToLower(strings.Split(f.ContentType, ";")[0])
	ct = strings.TrimSpace(ct)
	switch {
	case strings.Contains(ct, "pdf"):
		mime, ext = ingest.MIMEPDF, ".pdf"
		if !strings.HasSuffix(strings.ToLower(name), ".pdf") {
			name += ".pdf"
		}
	case strings.Contains(ct, "csv"):
		mime, ext = ingest.MIMECSV, ".csv"
		if !strings.HasSuffix(strings.ToLower(name), ".csv") {
			name += ".csv"
		}
	case strings.Contains(ct, "markdown"):
		mime, ext = ingest.MIMEMarkdown, ".md"
	default:
		mime, ext = ingest.MIMEHTML, ".html"
		if !strings.Contains(strings.ToLower(name), ".") {
			name += ".html"
		}
	}
	return ingest.BaseFilename(name), mime, ext
}

func writeStaged(tmp *os.File, body []byte) (sum string, n int64, err error) {
	h := sha256.New()
	n64, err := h.Write(body)
	if err != nil {
		return "", 0, err
	}
	if _, err := tmp.Write(body); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), int64(n64), nil
}

func (s *ingestJobService) processUpload(jobID, sessionID uuid.UUID, files []ingest.OpenedFile, pageTitle string) {
	defer func() { <-s.sem }()
	defer cleanupStaged(files)
	start := time.Now()
	bg := context.WithoutCancel(context.Background())
	ctx, cancel := context.WithTimeout(bg, time.Duration(s.cfg.WorkerTimeoutMin)*time.Minute)
	defer cancel()

	if _, err := s.jobs.UpdateStatus(ctx, jobID, "processing", len(files), ""); err != nil {
		log.Error("ingest: mark processing failed", "job_id", jobID.String(), "error", err)
		return
	}
	if s.rag == nil {
		_, _ = s.jobs.UpdateStatus(ctx, jobID, "failed", 0, "AI Core unavailable — gRPC client not connected")
		return
	}
	extractors := ingest.DefaultExtractors(s.cfg.MaxPagesPDF)
	var docs []model.Document
	var fileErrs []string
	totalChars := 0
	for _, f := range files {
		md, err := ingest.ExtractFile(ctx, f, extractors)
		if err != nil {
			fileErrs = append(fileErrs, err.Error())
			log.Error("ingest: extract failed", "job_id", jobID.String(), "file", f.Filename, "error", err)
			continue
		}
		if pageTitle != "" && len(files) == 1 {
			md.PageTitle = pageTitle
		}
		// Enforce per-file char cap: truncate at paragraph boundary.
		if s.cfg.MaxCharsPerFile > 0 && len(md.Markdown) > s.cfg.MaxCharsPerFile {
			md.Markdown = truncateAtBoundary(md.Markdown, s.cfg.MaxCharsPerFile)
			md.Truncated = true
		}
		parts := ingest.ToDocuments(md, ingest.MaxSectionChars)
		if len(parts) == 0 {
			fileErrs = append(fileErrs, f.Filename+": empty after cleaning")
			continue
		}
		for _, d := range parts {
			totalChars += len(d.Content)
		}
		docs = append(docs, parts...)
	}
	if len(docs) == 0 {
		msg := "all files failed extraction"
		if len(fileErrs) > 0 {
			msg = strings.Join(fileErrs, "; ")
		}
		_, _ = s.jobs.UpdateStatus(ctx, jobID, "failed", 0, msg)
		return
	}
	resp, err := s.rag.IngestDocuments(ctx, &model.IngestRequest{SessionID: sessionID, Documents: docs})
	if err != nil {
		msg := err.Error()
		if len(fileErrs) > 0 {
			msg = strings.Join(fileErrs, "; ") + " | " + msg
		}
		_, _ = s.jobs.UpdateStatus(ctx, jobID, "failed", 0, msg)
		log.Error("ingest: rag failed", "job_id", jobID.String(), "error", err)
		return
	}
	warn := ""
	if len(fileErrs) > 0 {
		warn = "partial failures: " + strings.Join(fileErrs, "; ")
		if resp.Error != "" {
			warn += " | " + resp.Error
		}
	} else {
		warn = resp.Error
	}
	updated, err := s.jobs.UpdateStatus(ctx, jobID, "completed", int(resp.Count), warn)
	_ = updated
	if err != nil {
		log.Error("ingest: mark completed failed", "job_id", jobID.String(), "error", err)
		return
	}
	log.Info("ingest: upload completed",
		"job_id", jobID.String(),
		"session_id", sessionID.String(),
		"files", len(files),
		"docs", len(docs),
		"chars", totalChars,
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

func truncateAtBoundary(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := strings.LastIndex(s[:max], "\n\n")
	if cut < max/2 {
		cut = max
	}
	return s[:cut]
}

func cleanupStaged(files []ingest.OpenedFile) {
	for _, f := range files {
		if f.Path != "" {
			_ = os.Remove(f.Path)
		}
	}
}

func (s *ingestJobService) GetJob(ctx context.Context, id uuid.UUID) (*model.IngestJob, error) {
	return s.jobs.Get(ctx, id)
}

func (s *ingestJobService) ListJobs(ctx context.Context, sessionID uuid.UUID, limit, offset int) ([]model.IngestJob, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.jobs.ListBySession(ctx, sessionID, limit, offset)
}
