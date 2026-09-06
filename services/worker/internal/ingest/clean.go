package ingest

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// CleanMarkdown normalizes extracted text to Markdown per spec §6.4.
// Pure function: no I/O, no logging.
func CleanMarkdown(s, filename string) string {
	if s == "" {
		return ""
	}
	// Unicode NFC.
	s = norm.NFC.String(s)
	// Normalize line endings.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// Strip NUL, replace other Cc (except \n \t) with space.
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == 0 {
			continue
		}
		if r == '\n' || r == '\t' {
			b.WriteRune(r)
			continue
		}
		if unicode.Is(unicode.Cc, r) {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	s = b.String()
	// Trim trailing spaces per line.
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimRight(ln, " \t")
	}
	s = strings.Join(lines, "\n")
	// Collapse 3+ newlines to 2.
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Prepend title block if no heading present.
	hasHeading := false
	for _, ln := range strings.SplitN(s, "\n", 5) {
		if strings.HasPrefix(strings.TrimSpace(ln), "#") {
			hasHeading = true
			break
		}
	}
	if !hasHeading && filename != "" {
		s = "# " + filename + "\n\n" + s
	}
	return s
}

// TooShortAfterClean reports empty-after-cleaning per spec (<10 non-space chars).
func TooShortAfterClean(s string) bool {
	n := 0
	for _, r := range s {
		if !unicode.IsSpace(r) {
			n++
		}
	}
	return n < 10
}
