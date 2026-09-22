package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGoldenPhase2Files(t *testing.T) {
	cases := []struct {
		file string
		mime string
		ext  string
	}{
		{"sample.html", MIMEHTML, ".html"},
		{"sample.csv", MIMECSV, ".csv"},
		{"sample.pptx", MIMEPPTX, ".pptx"},
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
				t.Fatalf("missing expected file: %v", err)
			}
			if want := strings.TrimRight(string(raw), "\n"); doc.Markdown != want {
				t.Errorf("golden mismatch for %s:\n--- want ---\n%s\n--- got ---\n%s", tc.file, want, doc.Markdown)
			}
		})
	}
}

func TestHTMLDropsNavFooterScript(t *testing.T) {
	path := filepath.Join("testdata", "sample.html")
	fi, _ := os.Stat(path)
	f := OpenedFile{Filename: "sample.html", MIME: MIMEHTML, Ext: ".html", Size: fi.Size(), Checksum: "x", Path: path}
	doc, err := (HTMLExtractor{}).Extract(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if doc.PageTitle != "Chimera HTML Sample" {
		t.Errorf("want title, got %q", doc.PageTitle)
	}
	for _, bad := range []string{"Nav content", "Footer content", "var x = 1"} {
		if strings.Contains(doc.Markdown, bad) {
			t.Errorf("want %q dropped, got:\n%s", bad, doc.Markdown)
		}
	}
	for _, want := range []string{"# Main Article Title", "**first**", "[link](https://example.com/page)", "- Alpha item", "| K | V |"} {
		if !strings.Contains(doc.Markdown, want) {
			t.Errorf("want %q in:\n%s", want, doc.Markdown)
		}
	}
	if docs := ToDocuments(doc, 60000); len(docs) != 1 || docs[0].SourceType != "url" {
		t.Errorf("html should map to url source_type: %+v", docs)
	}
}

func TestCSVSemicolonDelimiter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "semi.csv")
	if err := os.WriteFile(path, []byte("a;b\n1;2\n3;4\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fi, _ := os.Stat(path)
	f := OpenedFile{Filename: "semi.csv", MIME: MIMECSV, Ext: ".csv", Size: fi.Size(), Checksum: "x", Path: path}
	doc, err := (CSVExtractor{}).Extract(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc.Markdown, "| a | b |") {
		t.Errorf("semicolon delimiter not detected:\n%s", doc.Markdown)
	}
}

func TestCSVRejectsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.csv")
	if err := os.WriteFile(path, []byte("\n\n"), 0600); err != nil {
		t.Fatal(err)
	}
	f := OpenedFile{Filename: "empty.csv", MIME: MIMECSV, Ext: ".csv", Size: 2, Checksum: "x", Path: path}
	if _, err := (CSVExtractor{}).Extract(context.Background(), f); err == nil {
		t.Errorf("want error for empty csv")
	}
}

func TestPPTXSlideStructure(t *testing.T) {
	path := filepath.Join("testdata", "sample.pptx")
	fi, _ := os.Stat(path)
	f := OpenedFile{Filename: "sample.pptx", MIME: MIMEPPTX, Ext: ".pptx", Size: fi.Size(), Checksum: "x", Path: path}
	doc, err := (PPTXExtractor{}).Extract(context.Background(), f)
	if err != nil {
		t.Fatal(err)
	}
	if doc.PageCount != 2 {
		t.Errorf("want 2 slides, got %d", doc.PageCount)
	}
	for _, want := range []string{"## Slide 1", "## Slide 2", "- First bullet", "Thanks for reading."} {
		if !strings.Contains(doc.Markdown, want) {
			t.Errorf("want %q in:\n%s", want, doc.Markdown)
		}
	}
}

// Fetch SSRF guards.

func TestFetchBlocksPrivateTargets(t *testing.T) {
	opts := DefaultFetchOptions()
	opts.Timeout = 3 * time.Second
	for _, raw := range []string{
		"file:///etc/passwd",
		"ftp://example.com/x",
		"http://127.0.0.1/",
		"http://10.0.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://localhost:8000/",
	} {
		if _, err := FetchURL(context.Background(), raw, opts); err == nil {
			t.Errorf("want blocked: %s", raw)
		}
	}
}

func TestFetchRejectsOversize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write(make([]byte, 2048))
	}))
	defer srv.Close()
	opts := DefaultFetchOptions()
	opts.Timeout = 5 * time.Second
	opts.MaxBytes = 100
	opts.AllowPrivate = true
	if _, err := FetchURL(context.Background(), srv.URL, opts); err == nil {
		t.Errorf("want oversize error")
	}
}

func TestFetchHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html><head><title>T</title></head><body><p>Hello fetch body content here.</p></body></html>"))
	}))
	defer srv.Close()
	opts := DefaultFetchOptions()
	opts.Timeout = 5 * time.Second
	opts.AllowPrivate = true
	got, err := FetchURL(context.Background(), srv.URL+"/page", opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got.Body), "Hello fetch") {
		t.Errorf("unexpected body: %s", got.Body)
	}
}
