package ingest

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// DOCXExtractor converts .docx (OOXML) to Markdown using stdlib only.
// Maps: Heading1-3 -> #/##/###, bold -> **x**, italic -> *x*,
// numbered/bullet paragraphs -> "- ", tables -> GFM tables.
// Images, footnotes, headers/footers are dropped in v1.
type DOCXExtractor struct {
	MaxEntries       int   // default 1000
	MaxUncompressed  int64 // default 200MB
}

func (DOCXExtractor) Supports(mime, ext string) bool {
	return mime == MIMEDOCX || ext == ".docx"
}

func (e DOCXExtractor) limits() (int, int64) {
	maxE := e.MaxEntries
	if maxE <= 0 {
		maxE = 1000
	}
	maxU := e.MaxUncompressed
	if maxU <= 0 {
		maxU = 200 * 1024 * 1024
	}
	return maxE, maxU
}

func (e DOCXExtractor) Extract(_ context.Context, f OpenedFile) (MarkdownDoc, error) {
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
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "not a valid docx (zip parse failed)", Err: err}
	}
	maxE, maxU := e.limits()
	if len(zr.File) > maxE {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "docx has too many entries (possible zip bomb)"}
	}
	var totalUncompressed uint64
	for _, zf := range zr.File {
		totalUncompressed += zf.UncompressedSize64
		// Zip-slip guard: reject absolute paths, .. and symlinks even though
		// we never extract to disk.
		name := zf.Name
		if strings.HasPrefix(name, "/") || strings.Contains(name, "..") {
			return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "docx contains unsafe entry path"}
		}
		if zf.FileInfo().Mode()&0o120000 != 0 {
			return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "docx contains symlink entry"}
		}
	}
	if totalUncompressed > uint64(maxU) {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "docx uncompressed size exceeds limit"}
	}
	var docXML io.ReadCloser
	for _, zf := range zr.File {
		if zf.Name == "word/document.xml" {
			rc, err := zf.Open()
			if err != nil {
				return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "cannot read document.xml", Err: err}
			}
			docXML = rc
			break
		}
	}
	if docXML == nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "docx missing word/document.xml"}
	}
	defer docXML.Close()
	paras, tables, err := parseDocxXML(io.LimitReader(docXML, 50*1024*1024))
	if err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "docx xml parse failed", Err: err}
	}
	var b strings.Builder
	for _, blk := range paras {
		if blk.table != nil {
			b.WriteString(renderMDTable(blk.table))
			b.WriteString("\n\n")
			continue
		}
		text := strings.TrimSpace(blk.text)
		if text == "" {
			continue
		}
		switch blk.style {
		case "Heading1":
			b.WriteString("# " + text + "\n\n")
		case "Heading2":
			b.WriteString("## " + text + "\n\n")
		case "Heading3":
			b.WriteString("### " + text + "\n\n")
		default:
			if blk.isList {
				b.WriteString("- " + text + "\n\n")
			} else {
				b.WriteString(text + "\n\n")
			}
		}
	}
	_ = tables
	md := CleanMarkdown(b.String(), f.Filename)
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

type mdBlock struct {
	text   string
	style  string
	isList bool
	table  [][]string
}

func renderMDTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	cols := 0
	for _, r := range rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	var b strings.Builder
	esc := func(s string) string {
		s = strings.ReplaceAll(s, "|", "\\|")
		s = strings.ReplaceAll(s, "\n", " ")
		return strings.TrimSpace(s)
	}
	writeRow := func(r []string) {
		b.WriteString("|")
		for i := 0; i < cols; i++ {
			v := ""
			if i < len(r) {
				v = esc(r[i])
			}
			b.WriteString(" " + v + " |")
		}
		b.WriteString("\n")
	}
	writeRow(rows[0])
	b.WriteString("|")
	for i := 0; i < cols; i++ {
		b.WriteString(" --- |")
	}
	b.WriteString("\n")
	for _, r := range rows[1:] {
		writeRow(r)
	}
	return strings.TrimRight(b.String(), "\n")
}

// parseDocxXML streams word/document.xml and returns ordered blocks.
// Tables are emitted as mdBlock{table} in document order; paragraphs otherwise.
func parseDocxXML(r io.Reader) ([]mdBlock, int, error) {
	dec := xml.NewDecoder(r)
	var blocks []mdBlock
	tableCount := 0
	var (
		inP, inR, inT, inTbl, inTr, inTc     bool
		inB, inI, inPStyle, inNumPr          bool
		curBold, curItalic                   bool
		curText                              strings.Builder
		curStyle                             string
		curIsList                            bool
		curTable                             [][]string
		curRow                               []string
		curCell                              strings.Builder
	)
	flushParagraph := func() {
		t := curText.String()
		if strings.TrimSpace(t) != "" || curStyle != "" {
			blocks = append(blocks, mdBlock{text: t, style: curStyle, isList: curIsList})
		}
		curText.Reset()
		curStyle = ""
		curIsList = false
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			name := t.Name.Local
			switch name {
			case "p":
				if !inTbl {
					inP = true
					curText.Reset()
					curStyle = ""
					curIsList = false
				} else {
					// paragraph inside table cell: separate with space
					if curCell.Len() > 0 {
						curCell.WriteString(" ")
					}
				}
			case "tbl":
				inTbl = true
				curTable = nil
			case "tr":
				if inTbl {
					inTr = true
					curRow = nil
				}
			case "tc":
				if inTbl {
					inTc = true
					curCell.Reset()
				}
			case "r":
				inR = true
				inB, inI = false, false
				curBold, curItalic = false, false
			case "t":
				inT = true
			case "b":
				if inR {
					inB = true
					curBold = true
				}
			case "i":
				if inR {
					inI = true
					curItalic = true
				}
			case "pStyle":
				inPStyle = true
				for _, a := range t.Attr {
					if a.Name.Local == "val" {
						v := a.Value
						if strings.HasPrefix(v, "Heading") {
							curStyle = v
						} else if v == "Title" {
							curStyle = "Heading1"
						}
					}
				}
			case "numPr":
				if inP && !inTbl {
					curIsList = true
				}
				inNumPr = true
			case "br":
				if inP && !inTbl {
					curText.WriteString("\n")
				} else if inTc {
					curCell.WriteString(" ")
				}
			case "tab":
				if inP && !inTbl {
					curText.WriteString(" ")
				} else if inTc {
					curCell.WriteString(" ")
				}
			}
		case xml.EndElement:
			name := t.Name.Local
			switch name {
			case "p":
				if !inTbl {
					inP = false
					flushParagraph()
				}
			case "tbl":
				inTbl = false
				blocks = append(blocks, mdBlock{table: curTable})
				tableCount++
				curTable = nil
			case "tr":
				if inTbl && inTr {
					inTr = false
					curTable = append(curTable, curRow)
					curRow = nil
				}
			case "tc":
				if inTbl && inTc {
					inTc = false
					curRow = append(curRow, strings.TrimSpace(curCell.String()))
				}
			case "r":
				inR = false
				inB, inI = false, false
				curBold, curItalic = false, false
			case "t":
				inT = false
			case "b":
				inB = false
			case "i":
				inI = false
			case "pStyle":
				inPStyle = false
			case "numPr":
				inNumPr = false
			}
		case xml.CharData:
			if inT {
				s := string(t)
				if inTbl && inTc {
					curCell.WriteString(s)
				} else if inP {
					bold := inB || curBold
					italic := inI || curItalic
					if bold && italic {
						curText.WriteString("***" + s + "***")
					} else if bold {
						curText.WriteString("**" + s + "**")
					} else if italic {
						curText.WriteString("*" + s + "*")
					} else {
						curText.WriteString(s)
					}
				}
			} else if inPStyle {
				// some producers put style name as chardata
				_ = inNumPr
			}
		}
	}
	return blocks, tableCount, nil
}

// ensure fmt import is used in non-test builds
var _ = fmt.Sprintf
