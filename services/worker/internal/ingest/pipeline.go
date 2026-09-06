package ingest

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/imrishabk/chimera/services/worker/internal/model"
)

// MaxSectionChars caps one model.Document.Content length.
// Python re-chunks anyway; this only keeps gRPC messages bounded.
const MaxSectionChars = 60000

// DefaultExtractors returns extractors in priority order.
func DefaultExtractors(maxPagesPDF int) []Extractor {
	return []Extractor{
		PDFExtractor{MaxPages: maxPagesPDF},
		DOCXExtractor{},
		PPTXExtractor{},
		CSVExtractor{},
		HTMLExtractor{},
		TXTExtractor{},
	}
}

// ValidateFile enforces allowlist, extension/MIME agreement, and non-empty names.
// size is staged bytes. Returns sniffed-extension-normalized ext or error.
func ValidateFile(filename, mime string, size int64, maxBytesPerFile int64) (string, error) {
	if strings.TrimSpace(filename) == "" {
		return "", fmt.Errorf("filename required")
	}
	if size <= 0 {
		return "", &ExtractionError{Filename: filename, Reason: "empty file"}
	}
	if size > maxBytesPerFile {
		return "", &ExtractionError{Filename: filename, Reason: fmt.Sprintf("file exceeds per-file limit (%d bytes)", maxBytesPerFile)}
	}
	ext := normalizeExt(filename)
	// Reject double-extension executables.
	lower := strings.ToLower(filename)
	for _, bad := range []string{".exe", ".bat", ".cmd", ".sh", ".dll", ".so", ".dylib"} {
		if strings.HasSuffix(lower, bad) {
			return "", &ExtractionError{Filename: filename, Reason: "file type not allowed"}
		}
	}
	allowedMIME := map[string]bool{
		MIMEText: true, MIMEMarkdown: true, MIMEPDF: true, MIMEDOCX: true, MIMEHTML: true,
		MIMECSV: true, MIMEPPTX: true,
		"text/plain; charset=utf-8": true,
		"text/csv; charset=utf-8":   true,
		"text/html; charset=utf-8":  true,
	}
	// net/http.DetectContentType appends "; charset=utf-8" for text.
	baseMIME := strings.Split(mime, ";")[0]
	baseMIME = strings.TrimSpace(strings.ToLower(baseMIME))
	if mime != "" && !allowedMIME[mime] && !allowedMIME[baseMIME] && !strings.HasPrefix(baseMIME, "text/") {
		// Allow text/* with matching ext as fallback; else reject.
		if !((ext == ".txt" || ext == ".md" || ext == ".markdown") && strings.HasPrefix(baseMIME, "text/")) {
			return "", &ExtractionError{Filename: filename, Reason: fmt.Sprintf("mime %q not allowed", mime)}
		}
	}
	allowedExt := map[string]bool{
		".txt": true, ".md": true, ".markdown": true, ".pdf": true, ".docx": true, ".html": true, ".htm": true,
		".csv": true, ".pptx": true,
	}
	if !allowedExt[ext] {
		return "", &ExtractionError{Filename: filename, Reason: fmt.Sprintf("extension %q not allowed", ext)}
	}
	return ext, nil
}

// ExtractFile dispatches to the first supporting extractor.
func ExtractFile(ctx context.Context, f OpenedFile, extractors []Extractor) (MarkdownDoc, error) {
	for _, e := range extractors {
		if e.Supports(f.MIME, f.Ext) {
			return e.Extract(ctx, f)
		}
	}
	return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: fmt.Sprintf("no extractor for mime %q ext %q", f.MIME, f.Ext)}
}

// ToDocuments maps one cleaned MarkdownDoc to proto-compatible model.Documents.
// SourceType is normalized to the proto enum; provenance goes to Metadata.
func ToDocuments(doc MarkdownDoc, maxChars int) []model.Document {
	if maxChars <= 0 {
		maxChars = MaxSectionChars
	}
	md := doc.Markdown
	if strings.TrimSpace(md) == "" {
		return nil
	}
	docID := uuid.NewString()
	sourceType := protoSourceType(doc.OrigMIME, doc.OrigExt)
	sections := SplitMarkdown(md, maxChars)
	truncated := ""
	if doc.Truncated {
		truncated = "true"
	}
	out := make([]model.Document, 0, len(sections))
	for i, sec := range sections {
		source := doc.Source
		if len(sections) > 1 {
			source = fmt.Sprintf("%s#part=%d/%d", doc.Source, i+1, len(sections))
		}
		meta := map[string]string{
			"original_filename": doc.Source,
			"original_mime":     doc.OrigMIME,
			"original_ext":      doc.OrigExt,
			"checksum_sha256":   doc.ChecksumSHA256,
			"page_count":        strconv.Itoa(doc.PageCount),
			"char_count":        strconv.Itoa(len(sec)),
			"truncated":         truncated,
		}
		out = append(out, model.Document{
			Content:    sec,
			Source:     source,
			SourceType: sourceType,
			DocID:      docID,
			ChunkIndex: int32(i),
			PageTitle:  doc.PageTitle,
			Metadata:   meta,
		})
	}
	return out
}

// SplitMarkdown splits on paragraph boundaries to stay under maxChars.
func SplitMarkdown(s string, maxChars int) []string {
	if len(s) <= maxChars {
		return []string{s}
	}
	paras := strings.Split(s, "\n\n")
	var sections []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			sections = append(sections, strings.TrimSpace(cur.String()))
			cur.Reset()
		}
	}
	for _, p := range paras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Single oversized paragraph: hard-split.
		if len(p) > maxChars {
			flush()
			for len(p) > maxChars {
				cut := strings.LastIndex(p[:maxChars], " ")
				if cut < maxChars/2 {
					cut = maxChars
				}
				sections = append(sections, strings.TrimSpace(p[:cut]))
				p = strings.TrimSpace(p[cut:])
			}
			if p != "" {
				cur.WriteString(p + "\n\n")
			}
			continue
		}
		if cur.Len()+len(p)+2 > maxChars {
			flush()
		}
		cur.WriteString(p + "\n\n")
	}
	flush()
	if len(sections) == 0 {
		return []string{s[:maxChars]}
	}
	return sections
}

// BaseFilename strips directory components (multipart safety).
func BaseFilename(name string) string {
	return filepath.Base(name)
}
