package ingest

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/dslipak/pdf"
)

// PDFExtractor converts PDF pages to Markdown with "## Page N" separators.
type PDFExtractor struct {
	MaxPages int // 0 = default 500
}

func (e PDFExtractor) Supports(mime, ext string) bool {
	return mime == MIMEPDF || ext == ".pdf"
}

func (e PDFExtractor) maxPages() int {
	if e.MaxPages > 0 {
		return e.MaxPages
	}
	return 500
}

func (e PDFExtractor) Extract(ctx context.Context, f OpenedFile) (MarkdownDoc, error) {
	fh, err := os.Open(f.Path)
	if err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "cannot open staged file", Err: err}
	}
	defer fh.Close()
	fi, err := fh.Stat()
	if err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "stat failed", Err: err}
	}
	r, err := pdf.NewReader(fh, fi.Size())
	if err != nil {
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "encrypt") || strings.Contains(strings.ToLower(msg), "password") {
			return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "pdf encrypted or extract forbidden"}
		}
		if strings.Contains(msg, "not a PDF") {
			return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "not a valid PDF"}
		}
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "pdf parse failed", Err: err}
	}
	numPages := r.NumPage()
	if numPages == 0 {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "no extractable text (scanned image? OCR out of scope)"}
	}
	if numPages > e.maxPages() {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: fmt.Sprintf("pdf exceeds page limit %d", e.maxPages())}
	}
	var buf bytes.Buffer
	totalChars := 0
	for i := 1; i <= numPages; i++ {
		if err := ctx.Err(); err != nil {
			return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "cancelled", Err: err}
		}
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		rows, err := p.GetTextByRow()
		if err != nil {
			continue // skip unreadable page, keep others
		}
		var pageBuf bytes.Buffer
		for _, row := range rows {
			line := joinRow(row.Content)
			line = strings.TrimRight(line, " \t")
			if line == "" {
				continue
			}
			pageBuf.WriteString(line)
			pageBuf.WriteString("\n")
		}
		pageText := strings.TrimSpace(pageBuf.String())
		if pageText == "" {
			continue
		}
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(fmt.Sprintf("## Page %d\n\n%s", i, pageText))
		totalChars += len(pageText)
	}
	if totalChars == 0 {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "no extractable text (scanned image? OCR out of scope)"}
	}
	md := CleanMarkdown(buf.String(), f.Filename)
	if TooShortAfterClean(md) {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "empty after cleaning"}
	}
	return MarkdownDoc{
		Markdown:       md,
		Source:         f.Filename,
		OrigMIME:       f.MIME,
		OrigExt:        f.Ext,
		PageCount:      numPages,
		ChecksumSHA256: f.Checksum,
		SizeBytes:      f.Size,
	}, nil
}

// joinRow concatenates text fragments on one row, inserting a space where
// the PDF split words/lines into separate show operations without spacing.
func joinRow(frags pdf.TextHorizontal) string {
	var b strings.Builder
	for _, w := range frags {
		s := w.S
		if s == "" {
			continue
		}
		if b.Len() > 0 {
			cur := b.String()
			last := rune(cur[len(cur)-1])
			first, _ := firstRune(s)
			if needsSpace(last, first) {
				b.WriteByte(' ')
			}
		}
		b.WriteString(s)
	}
	return b.String()
}

func firstRune(s string) (rune, bool) {
	for _, r := range s {
		return r, true
	}
	return 0, false
}

func needsSpace(last, first rune) bool {
	if last == ' ' || last == '\t' || first == ' ' || first == '\t' {
		return false
	}
	// Split words like "Hello"+"World" need a space; split word parts
	// ("Sec"+"ond") are rare at Tj granularity — prefer readability.
	if isWordChar(last) && isWordChar(first) {
		return true
	}
	// End of sentence followed by capital, e.g. "one."+"Second".
	if (last == '.' || last == ':' || last == ';') && unicode.IsUpper(first) {
		return true
	}
	return false
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
