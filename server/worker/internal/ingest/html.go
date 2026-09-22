package ingest

import (
	"context"
	"os"
	"strings"

	"golang.org/x/net/html"
)

// HTMLExtractor converts uploaded .html files to Markdown using a
// readability-lite strategy: prefer <main>/<article> when present,
// otherwise body minus nav/footer/aside, dropping script/style.
type HTMLExtractor struct{}

func (HTMLExtractor) Supports(mime, ext string) bool {
	return mime == MIMEHTML || ext == ".html" || ext == ".htm"
}

func (HTMLExtractor) Extract(_ context.Context, f OpenedFile) (MarkdownDoc, error) {
	fh, err := os.Open(f.Path)
	if err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "cannot open staged file", Err: err}
	}
	defer fh.Close()
	doc, err := html.Parse(fh)
	if err != nil {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "html parse failed", Err: err}
	}
	title := findTitle(doc)
	root := contentRoot(doc)
	md := htmlToMarkdown(root)
	md = CleanMarkdown(md, f.Filename)
	if TooShortAfterClean(md) {
		return MarkdownDoc{}, &ExtractionError{Filename: f.Filename, Reason: "empty after cleaning"}
	}
	return MarkdownDoc{
		Markdown:       md,
		Source:         f.Filename,
		OrigMIME:       f.MIME,
		OrigExt:        f.Ext,
		PageCount:      1,
		PageTitle:      title,
		ChecksumSHA256: f.Checksum,
		SizeBytes:      f.Size,
	}, nil
}

func findTitle(n *html.Node) string {
	var title string
	var walk func(*html.Node)
	walk = func(nd *html.Node) {
		if title != "" {
			return
		}
		if nd.Type == html.ElementNode && nd.Data == "title" && nd.FirstChild != nil {
			title = strings.TrimSpace(nd.FirstChild.Data)
			return
		}
		for c := nd.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return title
}

// contentRoot prefers <main>, then <article>, else <body>, else whole doc.
func contentRoot(doc *html.Node) *html.Node {
	for _, tag := range []string{"main", "article"} {
		if n := findElement(doc, tag); n != nil {
			return n
		}
	}
	if n := findElement(doc, "body"); n != nil {
		return n
	}
	return doc
}

func findElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findElement(c, tag); found != nil {
			return found
		}
	}
	return nil
}

func skippedContainer(tag string) bool {
	switch tag {
	case "script", "style", "noscript", "template", "nav", "footer", "aside":
		return true
	}
	return false
}

// htmlToMarkdown walks the content subtree and emits Markdown.
func htmlToMarkdown(root *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node, *strings.Builder)
	flushPara := func(text string) {
		t := strings.TrimSpace(text)
		if t == "" {
			return
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(t)
	}
	// block renders a node's inline content as one paragraph-ish block.
	var renderInline func(*html.Node) string
	renderInline = func(n *html.Node) string {
		var sb strings.Builder
		walk(n, &sb)
		return sb.String()
	}
	walk = func(n *html.Node, inline *strings.Builder) {
		emit := func(s string) {
			if inline != nil {
				inlineWrite(inline, s)
			} else {
				// top-level stray text becomes its own paragraph
				flushPara(s)
			}
		}
		switch n.Type {
		case html.TextNode:
			// Collapse internal whitespace but keep word separation.
			t := strings.Join(strings.Fields(n.Data), " ")
			if t == "" {
				return
			}
			if inline != nil {
				inlineWrite(inline, t)
			} else {
				flushPara(t)
			}
			return
		case html.ElementNode:
			tag := n.Data
			if skippedContainer(tag) {
				return
			}
			switch tag {
			case "h1":
				flushPara("# " + renderInlineChildren(n, walk))
				return
			case "h2":
				flushPara("## " + renderInlineChildren(n, walk))
				return
			case "h3", "h4", "h5", "h6":
				flushPara("### " + renderInlineChildren(n, walk))
				return
			case "p", "div", "section", "header", "blockquote":
				flushPara(renderInlineChildren(n, walk))
				return
			case "br":
				emit("\n")
				return
			case "hr":
				if b.Len() > 0 {
					b.WriteString("\n\n")
				}
				b.WriteString("---")
				return
			case "li":
				flushPara("- " + renderInlineChildren(n, walk))
				return
			case "ul", "ol":
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c, nil)
				}
				return
			case "pre":
				code := strings.Trim(extractText(n), "\n")
				if code != "" {
					if b.Len() > 0 {
						b.WriteString("\n\n")
					}
					b.WriteString("```\n" + code + "\n```")
				}
				return
			case "a":
				text := renderInlineChildren(n, walk)
				href := attr(n, "href")
				if text == "" {
					return
				}
				if href == "" || (!strings.HasPrefix(href, "http://") && !strings.HasPrefix(href, "https://")) {
					emit(text)
					return
				}
				emit("[" + text + "](" + href + ")")
				return
			case "strong", "b":
				emit("**" + renderInlineChildren(n, walk) + "**")
				return
			case "em", "i":
				emit("*" + renderInlineChildren(n, walk) + "*")
				return
			case "code":
				emit("`" + strings.TrimSpace(extractText(n)) + "`")
				return
			case "img":
				alt := attr(n, "alt")
				if alt != "" {
					emit(alt)
				}
				return
			case "table":
				tbl := extractTable(n)
				if len(tbl) > 0 {
					if b.Len() > 0 {
						b.WriteString("\n\n")
					}
					b.WriteString(renderMDTable(tbl))
				}
				return
			case "tr", "td", "th", "thead", "tbody":
				// handled via extractTable; avoid double-render
				return
			}
		}
		// default: recurse
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inline)
		}
		_ = renderInline
	}
	walk(root, nil)
	return b.String()
}

func renderInlineChildren(n *html.Node, walk func(*html.Node, *strings.Builder)) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, &sb)
	}
	return strings.TrimSpace(sb.String())
}

// inlineWrite appends s to an inline run, inserting a space only where
// word separation requires it (avoids "the**first**" and "page .").
func inlineWrite(sb *strings.Builder, s string) {
	if s == "" {
		return
	}
	if sb.Len() > 0 {
		cur := sb.String()
		last := rune(cur[len(cur)-1])
		first := rune(s[0])
		if inlineNeedsSpace(last, first) {
			sb.WriteByte(' ')
		}
	}
	sb.WriteString(s)
}

func inlineNeedsSpace(last, first rune) bool {
	if last == ' ' || last == '\t' || last == '\n' {
		return false
	}
	switch first {
	case ' ', '\t', '\n', ',', '.', ';', ':', '!', '?', ')', ']', '}':
		return false
	}
	switch last {
	case '(', '[', '{':
		return false
	}
	return true
}

func extractText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(nd *html.Node) {
		if nd.Type == html.TextNode {
			sb.WriteString(nd.Data)
		}
		for c := nd.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

func extractTable(tbl *html.Node) [][]string {
	var rows [][]string
	var walkTr func(*html.Node)
	walkTr = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			var row []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
					row = append(row, strings.Join(strings.Fields(extractText(c)), " "))
				}
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkTr(c)
		}
	}
	walkTr(tbl)
	return rows
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}
