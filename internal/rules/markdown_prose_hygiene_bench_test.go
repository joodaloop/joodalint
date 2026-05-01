package rules

import (
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// synthMarkdown builds a realistic-ish markdown body of roughly `lines` lines
// mixing prose, code fences, links, lists, and headings. Some lines contain
// patterns the rule actively flags so the benchmark exercises diagnostic
// emission, not just the no-match fast path.
func synthMarkdown(lines int) []byte {
	var b strings.Builder
	b.Grow(lines * 80)
	chunks := []string{
		"This is a paragraph of perfectly ordinary prose written for benchmark purposes.",
		"Here is [a link](https://example.com/path) inline with `some code` and **bold**.",
		"# Heading line one",
		"## Heading line two with more words after it",
		"- bullet item with a few words",
		"- another bullet, this one slightly longer than the previous bullet item",
		"1. ordered item one",
		"2. ordered item two with a `code span` inside it",
		"> quoted text that spans across what would be multiple wrapped visual lines",
		"```",
		"code := fence.Block(\"with content\")",
		"```",
		"A line with a typo: the the duplicate word should fire prose-hygiene.",
		"Mixed punctuation,like this,no space after comma, would fire missing-space-after-punct.",
		"Ranges like 10-20 and 1990-1999 may fire the hyphen-range rule.",
		"Currency $ 100 with a space between symbol and number.",
		"Normal sentence ending with a period. And another normal sentence.",
		"Line containing an em dash — like this one, plus an en dash – here too.",
		"<p>Inline HTML chunk that should be stripped before word-rep checks.</p>",
		"",
	}
	for i := 0; b.Len() < lines*80 && i < lines; i++ {
		b.WriteString(chunks[i%len(chunks)])
		b.WriteByte('\n')
	}
	return []byte(b.String())
}

func BenchmarkProseHygiene_Small(b *testing.B) {
	body := synthMarkdown(100)
	mf := &MarkdownFile{Path: "bench.md", Body: body, BodyStartLine: 1}
	b.SetBytes(int64(len(body)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = (markdownProseHygiene{}).Check(mf, nil)
	}
}

func BenchmarkProseHygiene_Medium(b *testing.B) {
	body := synthMarkdown(500)
	mf := &MarkdownFile{Path: "bench.md", Body: body, BodyStartLine: 1}
	b.SetBytes(int64(len(body)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = (markdownProseHygiene{}).Check(mf, nil)
	}
}

func BenchmarkProseHygiene_Large(b *testing.B) {
	body := synthMarkdown(2000)
	mf := &MarkdownFile{Path: "bench.md", Body: body, BodyStartLine: 1}
	b.SetBytes(int64(len(body)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = (markdownProseHygiene{}).Check(mf, nil)
	}
}

func BenchmarkFlattenProse(b *testing.B) {
	for _, sz := range []struct {
		name  string
		lines int
	}{
		{"Small", 100},
		{"Medium", 500},
		{"Large", 2000},
	} {
		body := synthMarkdown(sz.lines)
		parser := goldmark.New(goldmark.WithExtensions(extension.Strikethrough)).Parser()
		root := parser.Parse(text.NewReader(body))
		b.Run(sz.name, func(b *testing.B) {
			b.SetBytes(int64(len(body)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = FlattenProse(body, root)
			}
		})
	}
}
