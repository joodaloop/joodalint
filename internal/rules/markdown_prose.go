package rules

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark/ast"
	"golang.org/x/net/html"
)

// htmlTextContent extracts visible text content from an HTML fragment,
// using the standard html tokenizer. Returns nil, false if there is no
// meaningful text content (only tags/whitespace).
func htmlTextContent(src []byte) ([]byte, bool) {
	z := html.NewTokenizer(bytes.NewReader(src))
	var sb strings.Builder
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.TextToken {
			sb.Write(z.Text())
		}
	}
	if sb.Len() == 0 {
		return nil, false
	}
	return []byte(sb.String()), true
}

// isStyleScriptHTMLBlock reports whether an HTMLBlock is a <style> or
// <script> raw block whose contents are not prose. CSS/JS shouldn't be
// fed to prose checks or the speller.
func isStyleScriptHTMLBlock(body []byte, h *ast.HTMLBlock) bool {
	lines := h.Lines()
	if lines.Len() == 0 {
		return false
	}
	first := lines.At(0)
	lower := bytes.ToLower(body[first.Start:first.Stop])
	return bytes.Contains(lower, []byte("<style")) || bytes.Contains(lower, []byte("<script"))
}

// FlattenProse walks the markdown AST and returns one ProseBlock per
// top-level prose container (paragraph, heading, list-item text block,
// blockquote paragraph, HTML block). Inline code spans, fenced/indented
// code blocks, link URLs, and autolink targets are skipped; link text,
// image alt text, and raw inline HTML are included. Each span records
// the byte offset into body so callers can resolve line numbers via
// MarkdownFile.LineAt.
func FlattenProse(body []byte, root ast.Node) []ProseBlock {
	var blocks []ProseBlock
	var current *ProseBlock

	flush := func() {
		if current != nil && len(current.Spans) > 0 {
			blocks = append(blocks, *current)
		}
		current = nil
	}

	addSeg := func(start, stop int) {
		if current == nil || start >= stop {
			return
		}
		current.Spans = append(current.Spans, ProseSpan{
			Text:   body[start:stop],
			Offset: start,
		})
	}

	addSpan := func(text []byte, offset int) {
		if current == nil || len(text) == 0 {
			return
		}
		current.Spans = append(current.Spans, ProseSpan{
			Text:   text,
			Offset: offset,
		})
	}

	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := n.(type) {
		case *ast.Paragraph, *ast.Heading, *ast.TextBlock:
			if entering {
				flush()
				current = &ProseBlock{}
			} else {
				flush()
			}
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			return ast.WalkSkipChildren, nil
		case *ast.HTMLBlock:
			if entering {
				flush()
				if isStyleScriptHTMLBlock(body, node) {
					return ast.WalkSkipChildren, nil
				}
				blk := ProseBlock{}
				lines := node.Lines()
				for i := 0; i < lines.Len(); i++ {
					seg := lines.At(i)
					if seg.Stop > seg.Start {
						text, ok := htmlTextContent(body[seg.Start:seg.Stop])
						if ok {
							blk.Spans = append(blk.Spans, ProseSpan{
								Text:   text,
								Offset: seg.Start,
							})
						}
					}
				}
				if len(blk.Spans) > 0 {
					blocks = append(blocks, blk)
				}
			}
			return ast.WalkSkipChildren, nil
		case *ast.CodeSpan, *ast.AutoLink:
			return ast.WalkSkipChildren, nil
		case *ast.Text:
			if entering {
				addSeg(node.Segment.Start, node.Segment.Stop)
			}
		case *ast.RawHTML:
			if entering {
				segs := node.Segments
				for i := 0; i < segs.Len(); i++ {
					seg := segs.At(i)
					text, ok := htmlTextContent(body[seg.Start:seg.Stop])
					if ok {
						addSpan(text, seg.Start)
					}
				}
			}
		}
		return ast.WalkContinue, nil
	})
	flush()
	return blocks
}
