package ingest

import (
	"strings"
	"testing"
)

func TestCleanMarkdownStripsNULAndCollapses(t *testing.T) {
	in := "Hello\x00 World\x01\x02\r\n\r\n\r\nline with trailing   \n\n\n\nend"
	got := CleanMarkdown(in, "f.txt")
	if strings.Contains(got, "\x00") {
		t.Errorf("NUL not stripped: %q", got)
	}
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("blank lines not collapsed: %q", got)
	}
	if !strings.HasPrefix(got, "# f.txt") {
		t.Errorf("expected title block, got %q", got)
	}
}

func TestCleanMarkdownKeepsExistingHeading(t *testing.T) {
	in := "# Title\n\nbody"
	got := CleanMarkdown(in, "f.txt")
	if strings.HasPrefix(got, "# f.txt") {
		t.Errorf("should not prepend title when heading exists: %q", got)
	}
}

func TestCleanMarkdownEmpty(t *testing.T) {
	if got := CleanMarkdown("   \n\n  ", "f.txt"); got != "" {
		t.Errorf("want empty, got %q", got)
	}
	if !TooShortAfterClean("  hi  ") {
		t.Errorf("want TooShort for tiny string")
	}
	if TooShortAfterClean("this is long enough content") {
		t.Errorf("want not-short for normal string")
	}
}

func TestSplitMarkdown(t *testing.T) {
	big := strings.Repeat("para one text here.\n\n", 5000) // ~100k
	parts := SplitMarkdown(big, 60000)
	if len(parts) < 2 {
		t.Fatalf("want split, got %d parts", len(parts))
	}
	for _, p := range parts {
		if len(p) > 60000 {
			t.Errorf("part exceeds cap: %d", len(p))
		}
	}
}

func TestToDocumentsMapping(t *testing.T) {
	doc := MarkdownDoc{
		Markdown:       "# report\n\nhello",
		Source:         "report.pdf",
		OrigMIME:       "application/pdf",
		OrigExt:        ".pdf",
		PageCount:      3,
		ChecksumSHA256: "abc",
	}
	docs := ToDocuments(doc, 60000)
	if len(docs) != 1 {
		t.Fatalf("want 1 doc, got %d", len(docs))
	}
	d := docs[0]
	if d.SourceType != "pdf" {
		t.Errorf("want pdf source_type, got %q", d.SourceType)
	}
	if d.DocID == "" {
		t.Errorf("DocID required")
	}
	if d.Metadata["original_filename"] != "report.pdf" {
		t.Errorf("metadata missing: %+v", d.Metadata)
	}
	if d.Metadata["checksum_sha256"] != "abc" {
		t.Errorf("checksum missing: %+v", d.Metadata)
	}
	// docx normalizes to markdown (proto enum safe)
	doc2 := MarkdownDoc{Markdown: "# x\n\ny", Source: "n.docx", OrigMIME: MIMEDOCX, OrigExt: ".docx", PageCount: 1}
	if got := ToDocuments(doc2, 60000)[0].SourceType; got != "markdown" {
		t.Errorf("want docx->markdown, got %q", got)
	}
}

func TestValidateFile(t *testing.T) {
	if _, err := ValidateFile("a.pdf", "application/pdf", 100, 1024); err != nil {
		t.Errorf("valid pdf rejected: %v", err)
	}
	if _, err := ValidateFile("a.pdf.exe", "application/pdf", 100, 1024); err == nil {
		t.Errorf("double extension should be rejected")
	}
	if _, err := ValidateFile("a.pdf", "application/pdf", 0, 1024); err == nil {
		t.Errorf("empty should be rejected")
	}
	if _, err := ValidateFile("a.pdf", "application/pdf", 2048, 1024); err == nil {
		t.Errorf("oversize should be rejected")
	}
	if _, err := ValidateFile("a.xyz", "application/octet-stream", 100, 1024); err == nil {
		t.Errorf("unknown ext should be rejected")
	}
}
