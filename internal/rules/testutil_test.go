package rules

import (
	"sort"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

var testParser = goldmark.New(goldmark.WithExtensions(extension.Strikethrough)).Parser()

// mdFile builds a MarkdownFile with the given content for testing,
// populated the same way the runner populates real files.
func mdFile(content string) *MarkdownFile {
	b := []byte(content)
	_, body, fmLines, _ := SplitFrontmatter(b)
	astRoot := testParser.Parse(text.NewReader(body))
	return &MarkdownFile{
		Path:          "test.md",
		Content:       b,
		Body:          body,
		AST:           astRoot,
		BodyStartLine: fmLines + 1,
		ProseBlocks:   FlattenProse(body, astRoot),
	}
}

// htmlFile builds an HTMLFile with text content for testing.
func htmlFile(text string) *HTMLFile {
	return &HTMLFile{Path: "test.html", URLPath: "/test.html", Text: text}
}

// messages extracts the Message strings from a slice of diagnostics.
func messages(diags []Diagnostic) []string {
	out := make([]string, 0, len(diags))
	for _, d := range diags {
		out = append(out, d.Message)
	}
	return out
}

// containsMsg reports whether any diagnostic message contains substr.
func containsMsg(diags []Diagnostic, substr string) bool {
	for _, d := range diags {
		if strings.Contains(d.Message, substr) {
			return true
		}
	}
	return false
}

// linesOf returns the sorted, unique line numbers from diagnostics.
func linesOf(diags []Diagnostic) []int {
	seen := map[int]bool{}
	for _, d := range diags {
		seen[d.Line] = true
	}
	out := make([]int, 0, len(seen))
	for l := range seen {
		out = append(out, l)
	}
	sort.Ints(out)
	return out
}

// assertNoDiags fails the test if diags is non-empty.
func assertNoDiags(t *testing.T, diags []Diagnostic) {
	t.Helper()
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %d: %v", len(diags), messages(diags))
	}
}
