package rules

import (
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// benchMarkdownFile builds a MarkdownFile with everything rules need:
// Body, AST, Content, BodyStartLine. Uses synthMarkdown from the
// prose-hygiene benchmark.
func benchMarkdownFile(lines int) *MarkdownFile {
	body := synthMarkdown(lines)
	parser := goldmark.New(goldmark.WithExtensions(extension.Strikethrough)).Parser()
	astRoot := parser.Parse(text.NewReader(body))
	return &MarkdownFile{
		Path:          "bench.md",
		Content:       body,
		Body:          body,
		AST:           astRoot,
		BodyStartLine: 1,
		ProseBlocks:   FlattenProse(body, astRoot),
	}
}

// BenchmarkGoldmarkParse measures the AST parse cost per file. Compare
// against per-rule benchmarks below to see how parse stacks up against the
// rules walking it.
func BenchmarkGoldmarkParse(b *testing.B) {
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
		b.Run(sz.name, func(b *testing.B) {
			b.SetBytes(int64(len(body)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = parser.Parse(text.NewReader(body))
			}
		})
	}
}

// BenchmarkMarkdownRules runs each registered Markdown / MarkdownAST /
// MarkdownText rule individually on the same medium-sized input, so they
// can be ranked relative to one another.
//
// Spelling is excluded because it forks aspell (benchmarked separately).
func BenchmarkMarkdownRules(b *testing.B) {
	mf := benchMarkdownFile(500)
	ctx := &MarkdownContext{}
	b.SetBytes(int64(len(mf.Body)))

	type ruleEntry struct {
		id    string
		check func() []Diagnostic
	}
	var entries []ruleEntry
	for _, r := range Markdown() {
		if r.ID() == "spelling" {
			continue
		}
		r := r
		entries = append(entries, ruleEntry{r.ID(), func() []Diagnostic { return r.Check(mf, ctx) }})
	}
	for _, r := range MarkdownAST() {
		r := r
		entries = append(entries, ruleEntry{r.ID(), func() []Diagnostic { return r.Check(mf, ctx) }})
	}
	for _, r := range MarkdownText() {
		r := r
		entries = append(entries, ruleEntry{r.ID(), func() []Diagnostic { return r.Check(mf, ctx) }})
	}

	for _, e := range entries {
		b.Run(e.id, func(b *testing.B) {
			b.SetBytes(int64(len(mf.Body)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = e.check()
			}
		})
	}
}
