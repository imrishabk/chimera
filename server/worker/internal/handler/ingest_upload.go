package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"

	appErrs "github.com/imrishabk/chimera/services/worker/internal/errors"
	"github.com/imrishabk/chimera/services/worker/internal/ingest"
)

// Upload accepts multipart file uploads, stages them to disk, and enqueues
// an async ingestion job. Returns 202 with the pending job.
// Form fields: session_id (uuid, required), files (1+, required),
// page_title (optional, single-file only).
func (h *IngestHandler) Upload(w http.ResponseWriter, r *http.Request) error {
	if h.job == nil {
		return appErrs.ErrIngestionSeviceUnavailable
	}
	mr, err := r.MultipartReader()
	if err != nil {
		return appErrs.ErrInvalidBody
	}
	var (
		sessionID uuid.UUID
		pageTitle string
		files     []ingest.OpenedFile
		total     int64
	)
	cleanup := func() {
		for _, f := range files {
			if f.Path != "" {
				_ = os.Remove(f.Path)
			}
		}
	}
	maxPerFile := h.cfg.MaxBytesPerFile()
	maxTotal := h.cfg.MaxTotalBytes()
	maxFiles := h.cfg.MaxFilesPerRequest
	if maxPerFile <= 0 {
		maxPerFile = 25 * 1024 * 1024
	}
	if maxTotal <= 0 {
		maxTotal = 100 * 1024 * 1024
	}
	if maxFiles <= 0 {
		maxFiles = 10
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			cleanup()
			return appErrs.ErrInvalidBody
		}
		name := part.FormName()
		switch name {
		case "session_id":
			raw, _ := io.ReadAll(io.LimitReader(part, 128))
			sid, err := uuid.Parse(strings.TrimSpace(string(raw)))
			if err != nil {
				cleanup()
				return appErrs.ErrInvalidSession
			}
			sessionID = sid
		case "page_title":
			raw, _ := io.ReadAll(io.LimitReader(part, 512))
			pageTitle = strings.TrimSpace(string(raw))
		case "files", "file":
			filename := ingest.BaseFilename(part.FileName())
			if filename == "" || filename == "." || filename == "/" {
				cleanup()
				return appErrs.ErrInvalidUpload
			}
			if len(files) >= maxFiles {
				cleanup()
				return appErrs.ErrTooManyFiles
			}
			tmp, err := os.CreateTemp("", "chimera-upload-*")
			if err != nil {
				cleanup()
				return err
			}
			tmpName := tmp.Name()
			// Stream with per-file cap (+1 to detect overflow) while hashing.
			hasher := sha256.New()
			limited := io.LimitReader(part, maxPerFile+1)
			n, err := io.Copy(io.MultiWriter(tmp, hasher), limited)
			_ = tmp.Close()
			if err != nil {
				_ = os.Remove(tmpName)
				cleanup()
				return err
			}
			if n > maxPerFile {
				_ = os.Remove(tmpName)
				cleanup()
				return appErrs.ErrFileTooLarge
			}
			if n == 0 {
				_ = os.Remove(tmpName)
				cleanup()
				return appErrs.ErrInvalidUpload
			}
			total += n
			if total > maxTotal {
				_ = os.Remove(tmpName)
				cleanup()
				return appErrs.ErrRequestTooLarge
			}
			mime := sniffStaged(tmpName)
			mime = normalizeMIME(mime, filename)
			ext, err := ingest.ValidateFile(filename, mime, n, maxPerFile)
			if err != nil {
				_ = os.Remove(tmpName)
				cleanup()
				if isTooLarge(err) {
					return appErrs.ErrFileTooLarge
				}
				return appErrs.ErrUnsupportedFile
			}
			files = append(files, ingest.OpenedFile{
				Filename: filename,
				MIME:     mime,
				Ext:      ext,
				Size:     n,
				Checksum: hex.EncodeToString(hasher.Sum(nil)),
				Path:     tmpName,
			})
		default:
			// Drain unknown fields to keep the stream moving.
			_, _ = io.Copy(io.Discard, io.LimitReader(part, 1<<20))
		}
	}
	if sessionID == uuid.Nil || len(files) == 0 {
		cleanup()
		return appErrs.ErrInvalidUpload
	}
	userID := extractUserID(r)
	job, err := h.job.EnqueueUpload(r.Context(), sessionID, userID, files, pageTitle)
	if err != nil {
		cleanup()
		if strings.Contains(err.Error(), "queue full") {
			return appErrs.ErrUploadQueueFull
		}
		return err
	}
	// Ownership of temp files transfers to the background worker.
	return writeJSONData(w, http.StatusAccepted, job)
}

func sniffStaged(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := f.Read(head)
	if n == 0 {
		return "application/octet-stream"
	}
	return http.DetectContentType(head[:n])
}

// normalizeMIME corrects sniffing for known extensions.
// http.DetectContentType reports docx/pptx as application/zip, .md and
// .csv as text/plain; normalize so extractor dispatch and proto mapping
// stay correct.
func normalizeMIME(sniffed, filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".docx"):
		if sniffed == "application/zip" || sniffed == "application/octet-stream" ||
			sniffed == ingest.MIMEDOCX {
			return ingest.MIMEDOCX
		}
		return sniffed
	case strings.HasSuffix(lower, ".pptx"):
		if sniffed == "application/zip" || sniffed == "application/octet-stream" ||
			sniffed == ingest.MIMEPPTX {
			return ingest.MIMEPPTX
		}
		return sniffed
	case strings.HasSuffix(lower, ".csv"):
		if strings.HasPrefix(sniffed, "text/") || sniffed == "application/octet-stream" {
			return ingest.MIMECSV
		}
		return sniffed
	case strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown"):
		if strings.HasPrefix(sniffed, "text/") {
			return ingest.MIMEMarkdown
		}
		return sniffed
	case strings.HasSuffix(lower, ".pdf"):
		if sniffed == "application/octet-stream" {
			return ingest.MIMEPDF
		}
		return sniffed
	default:
		return sniffed
	}
}

func isTooLarge(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "exceeds per-file limit")
}
