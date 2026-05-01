package rules

import (
	"fmt"
	"strings"
	"testing"

	"github.com/joodaloop/hugolint/internal/config"
)

func synthFrontmatter(fields int) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("title: Benchmark Title\n")
	b.WriteString("description: Benchmark description for frontmatter validation.\n")
	b.WriteString("date: 2026-05-01\n")
	b.WriteString("weight: 42\n")
	b.WriteString("draft: false\n")
	b.WriteString("tags:\n")
	b.WriteString("  - perf\n")
	b.WriteString("  - benchmark\n")
	for i := 0; i < fields; i++ {
		fmt.Fprintf(&b, "extra_%03d: value %03d\n", i, i)
	}
	b.WriteString("---\n\n")
	b.WriteString("Body text.\n")
	return []byte(b.String())
}

func benchFrontmatterSchema(fields int) map[string]config.FieldSpec {
	schema := map[string]config.FieldSpec{
		"title":       {Type: "string", Required: true, Min: 1, Max: 80},
		"description": {Type: "text", Required: true, Min: 1, Max: 160},
		"date":        {Type: "date", Required: true},
		"weight":      {Type: "number", Min: 0, Max: 100},
		"draft":       {Type: "bool"},
		"tags":        {Type: "list", Items: "string"},
	}
	for i := 0; i < fields; i++ {
		schema[fmt.Sprintf("extra_%03d", i)] = config.FieldSpec{Type: "string"}
	}
	return schema
}

func benchFrontmatterContext(schema map[string]config.FieldSpec) *FrontmatterContext {
	return &FrontmatterContext{
		Config: &config.Config{
			Paths: config.Paths{MarkdownRoot: "content"},
			Sections: map[string]map[string]config.FieldSpec{
				"root": schema,
			},
		},
	}
}

func BenchmarkSplitFrontmatter(b *testing.B) {
	content := synthFrontmatter(40)
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, _ = SplitFrontmatter(content)
	}
}

func BenchmarkParseFrontmatterYAML(b *testing.B) {
	content := synthFrontmatter(40)
	raw, _, _, _ := SplitFrontmatter(content)
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ParseFrontmatterYAML(raw); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFrontmatterRuleCheck(b *testing.B) {
	content := synthFrontmatter(40)
	raw, _, _, line0 := SplitFrontmatter(content)
	parsed, err := ParseFrontmatterYAML(raw)
	if err != nil {
		b.Fatal(err)
	}
	ctx := benchFrontmatterContext(benchFrontmatterSchema(40))
	f := &FrontmatterFile{
		Path:   "content/bench.md",
		Parsed: parsed,
		Line0:  line0,
	}
	rule := markdownFrontmatter{}
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rule.Check(f, ctx)
	}
}
