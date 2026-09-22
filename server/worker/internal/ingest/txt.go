package ingest

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// TXTExtractor handles text/plain and text/markdown by pass-through + clean.
type TXTExtractor struct{}

func (TXTExtractor) Supports(mime, ext string) bool {
	if mime == MIMEText || mime == MIMEMarkdown {
		return true
	}
	// Fallback: sniffers often report text/plain for .md.
	if ext == ".md" || ext == ".markdown" || ext == ".txt" {
		return mime == "" || strings.HasPrefix(mime, "text/")
	}
	return false
}

func (TXTExtractor) Extract(_ context.Context, f OpenedFile) (MarkdownDoc, error) {
	fh, err := os.Open(f.Path)
	if err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "cannot open staged file", Err: err}
	}
	defer fh.Close()
	raw, err := io.ReadAll(io.LimitReader(fh, 2*1024*1024+1))
	if err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "read failed", Err: err}
	}
	if len(raw) == 0 {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "empty file"}
	}
	// Replace invalid UTF-8.
	text := strings.ToValidUTF8(string(raw), "\uFFFD")
	md := CleanMarkdown(text, f.Filename)
	if TooShortAfterClean(md) {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "empty after cleaning"}
	}
	return MarkdownDoc{
		Markdown:       md,
		Source:         f.Filename,
		OrigMIME:       f.MIME,
		OrigExt:        f.Ext,
		PageCount:      1,
		ChecksumSHA256: f.Checksum,
		SizeBytes:      f.Size,
	}, nil
}

// SniffText validates that content looks like text (rejects binaries).
func SniffText(sample []byte) error {
	if len(sample) == 0 {
		return fmt.Errorf("empty file")
	}
	// NUL byte in first 8KB strongly indicates binary.
	n := len(sample)
	if n > 8192 {
		n = 8192
	}
	for _, b := range sample[:n] {
		if b == 0 {
			return fmt.Errorf("binary content detected")
		}
	}
	return nil
}
