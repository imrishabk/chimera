package ingest

import (
	"os"
	"strings"
)

// openFile is a seam for tests.
func openFile(path string) (*os.File, error) {
	return os.Open(path)
}

// normalizeExt lowercases and ensures leading dot.
func normalizeExt(filename string) string {
	ext := ""
	if i := strings.LastIndex(filename, "."); i >= 0 {
		ext = strings.ToLower(filename[i:])
	}
	return ext
}

// protoSourceType maps internal MIME to the proto-enum source_type.
// Proto allows only: url | pdf | text | markdown | api.
// docx/pptx/csv normalize to markdown, html to url without
// requiring a proto migration.
func protoSourceType(mime, ext string) string {
	switch mime {
	case MIMEPDF:
		return "pdf"
	case MIMEText:
		if ext == ".md" || ext == ".markdown" {
			return "markdown"
		}
		return "text"
	case MIMEMarkdown:
		return "markdown"
	case MIMEDOCX, MIMEPPTX, MIMECSV:
		return "markdown"
	case MIMEHTML:
		return "url"
	default:
		switch ext {
		case ".md", ".markdown", ".docx", ".pptx", ".csv", ".html", ".htm":
			if ext == ".html" || ext == ".htm" {
				return "url"
			}
			return "markdown"
		}
		return "text"
	}
}
