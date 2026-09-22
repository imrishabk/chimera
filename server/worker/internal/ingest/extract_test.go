package ingest

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func stageTemp(t *testing.T, filename string, data []byte) OpenedFile {
	t.Helper()
	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return OpenedFile{Filename: filename, MIME: MIMEText, Ext: ".txt", Size: int64(len(data)), Checksum: "test", Path: path}
}

func TestTXTExtract(t *testing.T) {
	f := stageTemp(t, "notes.txt", []byte("hello world from txt\nsecond line"))
	doc, err := (TXTExtractor{}).Extract(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Markdown, "hello world") {
		t.Errorf("unexpected markdown: %q", doc.Markdown)
	}
}

func TestTXTExtractEmpty(t *testing.T) {
	f := stageTemp(t, "empty.txt", []byte{})
	if _, err := (TXTExtractor{}).Extract(context.Background(), f); err == nil {
		t.Errorf("want error for empty file")
	}
}

func TestDOCXExtract(t *testing.T) {
	// Build minimal docx in-memory.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("word/document.xml")
	_, _ = w.Write([]byte(`<?xml version="1.0"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Report Title</w:t></w:r></w:p>
<w:p><w:r><w:rPr><w:b/></w:rPr><w:t>bold</w:t></w:r><w:r><w:t> and normal</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr/></w:pPr><w:r><w:t>item one</w:t></w:r></w:p>
<w:tbl><w:tr><w:tc><w:p><w:r><w:t>A</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>B</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:r><w:t>1</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>2</w:t></w:r></w:p></w:tc></w:tr></w:tbl>
</w:body></w:document>`))
	_ = zw.Close()
	path := filepath.Join(t.TempDir(), "sample.docx")
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	f := OpenedFile{Filename: "sample.docx", MIME: MIMEDOCX, Ext: ".docx", Size: int64(buf.Len()), Checksum: "x", Path: path}
	doc, err := (DOCXExtractor{}).Extract(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Report Title", "**bold**", "- item one", "| A | B |"} {
		if !strings.Contains(doc.Markdown, want) {
			t.Errorf("want %q in:\n%s", want, doc.Markdown)
		}
	}
}

func TestDOCXRejectsZipSlip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	_, _ = zw.Create("../evil.xml")
	_ = zw.Close()
	path := filepath.Join(t.TempDir(), "evil.docx")
	_ = os.WriteFile(path, buf.Bytes(), 0600)
	f := OpenedFile{Filename: "evil.docx", MIME: MIMEDOCX, Ext: ".docx", Size: int64(buf.Len()), Checksum: "x", Path: path}
	if _, err := (DOCXExtractor{}).Extract(context.Background(), f); err == nil {
		t.Errorf("want error for unsafe entry")
	}
}
