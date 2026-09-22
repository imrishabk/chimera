package ingest

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// PPTXExtractor converts .pptx slides to Markdown using stdlib only.
// Each slide becomes "## Slide N" with paragraphs; bulleted paragraphs
// (<a:buChar>, <a:buAutoNum>, <a:buBlip>) become "- " items.
// Slide order follows numeric suffix (slide1.xml, slide2.xml, ...).
// Speaker notes, images, and charts are dropped in v1.
type PPTXExtractor struct {
	MaxEntries      int
	MaxUncompressed int64
	MaxSlides       int // default 200
}

func (PPTXExtractor) Supports(mime, ext string) bool {
	return mime == MIMEPPTX || ext == ".pptx"
}

func (e PPTXExtractor) limits() (int, int64, int) {
	maxE, maxU, maxS := e.MaxEntries, e.MaxUncompressed, e.MaxSlides
	if maxE <= 0 {
		maxE = 1000
	}
	if maxU <= 0 {
		maxU = 200 * 1024 * 1024
	}
	if maxS <= 0 {
		maxS = 200
	}
	return maxE, maxU, maxS
}

func (e PPTXExtractor) Extract(_ context.Context, f OpenedFile) (MarkdownDoc, error) {
	fh, err := os.Open(f.Path)
	if err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "cannot open staged file", Err: err}
	}
	defer fh.Close()
	fi, err := fh.Stat()
	if err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "stat failed", Err: err}
	}
	zr, err := zip.NewReader(fh, fi.Size())
	if err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "not a valid pptx (zip parse failed)", Err: err}
	}
	maxE, maxU, maxS := e.limits()
	if len(zr.File) > maxE {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "pptx has too many entries (possible zip bomb)"}
	}
	var totalUncompressed uint64
	slides := map[int]*zip.File{}
	for _, zf := range zr.File {
		totalUncompressed += zf.UncompressedSize64
		name := zf.Name
		if strings.HasPrefix(name, "/") || strings.Contains(name, "..") {
			return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "pptx contains unsafe entry path"}
		}
		if zf.FileInfo().Mode()&0o120000 != 0 {
			return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "pptx contains symlink entry"}
		}
		if strings.HasPrefix(name, "ppt/slides/slide") && strings.HasSuffix(name, ".xml") {
			mid := strings.TrimSuffix(strings.TrimPrefix(name, "ppt/slides/slide"), ".xml")
			if n, err := strconv.Atoi(mid); err == nil && n >= 1 {
				slides[n] = zf
			}
		}
	}
	if totalUncompressed > uint64(maxU) {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "pptx uncompressed size exceeds limit"}
	}
	if len(slides) == 0 {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "pptx contains no slides"}
	}
	if len(slides) > maxS {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: fmt.Sprintf("pptx exceeds slide limit %d", maxS)}
	}
	ordered := make([]int, 0, len(slides))
	for n := range slides {
		ordered = append(ordered, n)
	}
	sort.Ints(ordered)
	var b strings.Builder
	for i, n := range ordered {
		rc, err := slides[n].Open()
		if err != nil {
			continue
		}
		paras, err := parseSlideXML(io.LimitReader(rc, 10*1024*1024))
		_ = rc.Close()
		if err != nil || len(paras) == 0 {
			continue
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(fmt.Sprintf("## Slide %d\n\n", n))
		for _, p := range paras {
			t := strings.TrimSpace(p.text)
			if t == "" {
				continue
			}
			if p.isList {
				b.WriteString("- " + t + "\n\n")
			} else {
				b.WriteString(t + "\n\n")
			}
		}
	}
	md := CleanMarkdown(b.String(), f.Filename)
	if TooShortAfterClean(md) {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "empty after cleaning"}
	}
	return MarkdownDoc{
		Markdown:       md,
		Source:         f.Filename,
		OrigMIME:       f.MIME,
		OrigExt:        f.Ext,
		PageCount:      len(ordered),
		ChecksumSHA256: f.Checksum,
		SizeBytes:      f.Size,
	}, nil
}

type slidePara struct {
	text   string
	isList bool
}

// parseSlideXML streams ppt/slides/slideN.xml collecting <a:p> paragraphs.
// <a:t> runs are concatenated; <a:br> adds newline; any bullet marker
// (<a:buChar>, <a:buAutoNum>, <a:buBlip>) flags the paragraph as a list item.
func parseSlideXML(r io.Reader) ([]slidePara, error) {
	dec := xml.NewDecoder(r)
	var paras []slidePara
	var (
		inP, inT  bool
		isList    bool
		curText   strings.Builder
	)
	flush := func() {
		t := strings.TrimSpace(curText.String())
		if t != "" {
			paras = append(paras, slidePara{text: t, isList: isList})
		}
		curText.Reset()
		isList = false
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				inP = true
				curText.Reset()
				isList = false
			case "t":
				inT = true
			case "buChar", "buAutoNum", "buBlip":
				if inP {
					isList = true
				}
			case "br":
				if inP {
					curText.WriteString("\n")
				}
			case "tab":
				if inP {
					curText.WriteString(" ")
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				if inP {
					inP = false
					flush()
				}
			case "t":
				inT = false
			}
		case xml.CharData:
			if inT && inP {
				s := string(t)
				if curText.Len() > 0 && !strings.HasSuffix(curText.String(), " ") && !strings.HasSuffix(curText.String(), "\n") && !strings.HasPrefix(s, " ") {
					curText.WriteString(" ")
				}
				curText.WriteString(s)
			}
		}
	}
	return paras, nil
}
