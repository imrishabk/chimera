package ingest

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// CSVExtractor converts .csv files to a Markdown table.
// Delimiter is sniffed from comma/semicolon/tab on the first line.
type CSVExtractor struct {
	MaxRows int // default 5000
	MaxCols int // default 50
}

func (CSVExtractor) Supports(mime, ext string) bool {
	return mime == MIMECSV || ext == ".csv"
}

func (e CSVExtractor) limits() (int, int) {
	maxR, maxC := e.MaxRows, e.MaxCols
	if maxR <= 0 {
		maxR = 5000
	}
	if maxC <= 0 {
		maxC = 50
	}
	return maxR, maxC
}

func (e CSVExtractor) Extract(_ context.Context, f OpenedFile) (MarkdownDoc, error) {
	maxR, maxC := e.limits()
	fh, err := os.Open(f.Path)
	if err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "cannot open staged file", Err: err}
	}
	defer fh.Close()
	head := make([]byte, 4096)
	n, _ := fh.ReadAt(head, 0)
	delim := sniffDelimiter(head[:n])
	if _, err := fh.Seek(0, io.SeekStart); err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "seek failed", Err: err}
	}
	r := csv.NewReader(io.LimitReader(fh, 5*1024*1024+1))
	r.Comma = delim
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	var rows [][]string
	for len(rows) <= maxR {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "csv parse failed", Err: err}
		}
		if len(rec) > maxC {
			rec = rec[:maxC]
		}
		empty := true
		for _, c := range rec {
			if strings.TrimSpace(c) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		rows = append(rows, rec)
	}
	if len(rows) == 0 {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "empty after cleaning"}
	}
	truncated := false
	if len(rows) > maxR {
		rows = rows[:maxR]
		truncated = true
	}
	md := CleanMarkdown(renderMDTable(rows), f.Filename)
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
		Truncated:      truncated,
	}, nil
}

func sniffDelimiter(sample []byte) rune {
	line := string(sample)
	if i := strings.IndexAny(line, "\r\n"); i >= 0 {
		line = line[:i]
	}
	counts := map[rune]int{',': strings.Count(line, ","), ';': strings.Count(line, ";"), '\t': strings.Count(line, "\t")}
	best, bestN := ',', -1
	for _, d := range []rune{',', ';', '\t'} {
		if counts[d] > bestN {
			best, bestN = d, counts[d]
		}
	}
	if bestN <= 0 {
		return ','
	}
	return best
}

// ensure fmt import is used
var _ = fmt.Sprintf
