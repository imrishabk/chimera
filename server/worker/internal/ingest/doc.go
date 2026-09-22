// Package ingest implements file extraction to Markdown for the RAG pipeline.
//
// Design: see docs/ingestion-pipeline-spec.md.
// Extraction lives in the Go worker. Output is always a Markdown string
// sent to ai-core as Document.content with a proto-enum source_type.
// Python owns chunking and embedding and is never called with raw binaries.
package ingest

import (
	"context"
	"fmt"
	"io"
)

// Supported MIME allowlist (sniffed, not extension-trusted).
const (
	MIMEText     = "text/plain"
	MIMEMarkdown = "text/markdown"
	MIMEPDF      = "application/pdf"
	MIMEDOCX     = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MIMEHTML     = "text/html"
	MIMECSV      = "text/csv"
	MIMEPPTX     = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
)

// MarkdownDoc is the normalized output of one file.
type MarkdownDoc struct {
	Markdown       string
	Source         string // original filename or URL
	OrigMIME       string
	OrigExt        string
	PageCount      int
	PageTitle      string
	ChecksumSHA256 string
	SizeBytes      int64
	Truncated      bool
}

// OpenedFile is a staged upload ready for extraction.
type OpenedFile struct {
	Filename string
	MIME     string // sniffed MIME
	Ext      string // lowercase extension with dot
	Size     int64
	Checksum string // hex sha256
	Path     string // temp file path staged on disk
}

// Extractor converts one file format to MarkdownDoc.
type Extractor interface {
	Supports(mime, ext string) bool
	Extract(ctx context.Context, f OpenedFile) (MarkdownDoc, error)
}

// ExtractionError is a user-safe per-file failure.
type ExtractionError struct {
	Filename string
	Reason   string
	Err      error
}

func (e *ExtractionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Filename, e.Reason, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Filename, e.Reason)
}

func (e *ExtractionError) Unwrap() error { return e.Err }

// ReadStaged opens the staged temp file for reading. Caller must close.
func (f OpenedFile) ReadStaged() (io.ReadCloser, error) {
	return openFile(f.Path)
}
