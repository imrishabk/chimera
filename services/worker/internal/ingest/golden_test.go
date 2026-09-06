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

// TestGoldenFiles extracts each testdata sample and compares exact Markdown
// against the committed .expected.md file.
func TestGoldenFiles(t *testing.T) {
	cases := []struct {
		file string
		mime string
		ext  string
	}{
		{"sample.txt", MIMEText, ".txt"},
		{"sample.md", MIMEMarkdown, ".md"},
		{"sample.pdf", MIMEPDF, ".pdf"},
		{"sample.docx", MIMEDOCX, ".docx"},
	}
	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			path := filepath.Join("testdata", tc.file)
			fi, err := os.Stat(path)
			if err != nil {
				t.Fatalf("missing golden input %s: %v", path, err)
			}
			f := OpenedFile{Filename: tc.file, MIME: tc.mime, Ext: tc.ext, Size: fi.Size(), Checksum: "golden", Path: path}
			doc, err := ExtractFile(ctx, f, DefaultExtractors(500))
			if err != nil {
				t.Fatalf("extract %s: %v", tc.file, err)
			}
			raw, err := os.ReadFile(path + ".expected.md")
			if err != nil {
				t.Fatalf("missing expected file %s.expected.md: %v", path, err)
			}
			want := strings.TrimRight(string(raw), "\n")
			if doc.Markdown != want {
				t.Errorf("golden mismatch for %s:\n--- want ---\n%s\n--- got ---\n%s", tc.file, want, doc.Markdown)
			}
		})
	}
}

func TestGoldenPDFStructure(t *testing.T) {
	path := filepath.Join("testdata", "sample.pdf")
	fi, _ := os.Stat(path)
	f := OpenedFile{Filename: "sample.pdf", MIME: MIMEPDF, Ext: ".pdf", Size: fi.Size(), Checksum: "x", Path: path}
	doc, err := (PDFExtractor{}).Extract(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if doc.PageCount != 2 {
		t.Errorf("want 2 pages, got %d", doc.PageCount)
	}
	for _, want := range []string{"## Page 1", "## Page 2", "Chimera Golden PDF", "Second page here."} {
		if !strings.Contains(doc.Markdown, want) {
			t.Errorf("want %q in pdf markdown:\n%s", want, doc.Markdown)
		}
	}
	docs := ToDocuments(doc, 60000)
	if len(docs) != 1 || docs[0].SourceType != "pdf" {
		t.Errorf("want 1 pdf doc, got %+v", docs)
	}
}

func TestGoldenDOCXStructure(t *testing.T) {
	path := filepath.Join("testdata", "sample.docx")
	fi, _ := os.Stat(path)
	f := OpenedFile{Filename: "sample.docx", MIME: MIMEDOCX, Ext: ".docx", Size: fi.Size(), Checksum: "x", Path: path}
	doc, err := (DOCXExtractor{}).Extract(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Chimera Golden Doc", "**Bold intro**", "- First item", "| Name | Value |"} {
		if !strings.Contains(doc.Markdown, want) {
			t.Errorf("want %q in docx markdown:\n%s", want, doc.Markdown)
		}
	}
}

// Error paths for the PDF extractor.

func TestPDFRejectsNonPDF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fake.pdf")
	if err := os.WriteFile(path, []byte("this is not a pdf at all, just text........"), 0600); err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(path)
	f := OpenedFile{Filename: "fake.pdf", MIME: MIMEPDF, Ext: ".pdf", Size: fi.Size(), Checksum: "x", Path: path}
	if _, err := (PDFExtractor{}).Extract(context.Background(), f); err == nil {
		t.Errorf("want error for non-PDF bytes")
	}
}

func TestPDFEnforcesPageLimit(t *testing.T) {
	src, _ := os.ReadFile(filepath.Join("testdata", "sample.pdf"))
	path := filepath.Join(t.TempDir(), "two.pdf")
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(path)
	f := OpenedFile{Filename: "two.pdf", MIME: MIMEPDF, Ext: ".pdf", Size: fi.Size(), Checksum: "x", Path: path}
	if _, err := (PDFExtractor{MaxPages: 1}).Extract(context.Background(), f); err == nil {
		t.Errorf("want page-limit error for 2-page PDF with MaxPages=1")
	}
}

func TestDOCXRejectsMissingDocumentXML(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("word/styles.xml")
	_, _ = w.Write([]byte("<styles/>"))
	_ = zw.Close()
	path := filepath.Join(t.TempDir(), "nodoc.docx")
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	f := OpenedFile{Filename: "nodoc.docx", MIME: MIMEDOCX, Ext: ".docx", Size: int64(buf.Len()), Checksum: "x", Path: path}
	if _, err := (DOCXExtractor{}).Extract(context.Background(), f); err == nil {
		t.Errorf("want error for docx without word/document.xml")
	}
}
