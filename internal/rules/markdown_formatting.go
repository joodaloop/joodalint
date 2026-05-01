package rules

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
)

func init() {
	RegisterMarkdownAST(&markdownFormatting{})
}

// markdownFormatting flags inline formatting spans that are suspiciously
// long — usually a sign the opening delimiter was never closed.
type markdownFormatting struct{}

func (markdownFormatting) ID() string { return "formatting" }

// maxInlineSpan is the length threshold (in characters) above which
// inline formatting is reported as suspicious.
const maxInlineSpan = 120

func (markdownFormatting) Check(f *MarkdownFile, _ *MarkdownContext) []Diagnostic {
	if f.AST == nil {
		return nil
	}
	var diags []Diagnostic
	_ = ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Emphasis:
			if text := nodeText(v, f.Body); len([]rune(text)) > maxInlineSpan {
				diags = append(diags, Diagnostic{
					Path:    f.Path,
					Line:    f.NodeLine(v),
					Rule:    "formatting",
					Message: fmt.Sprintf("emphasis span %d chars — possible unclosed %s", len([]rune(text)), markerForLevel(v.Level)),
				})
			}
		case *ast.CodeSpan:
			if text := nodeText(v, f.Body); len([]rune(text)) > maxInlineSpan {
				diags = append(diags, Diagnostic{
					Path:    f.Path,
					Line:    f.NodeLine(v),
					Rule:    "formatting",
					Message: fmt.Sprintf("inline code span %d chars — possible unclosed backtick", len([]rune(text))),
				})
			}
		}
		return ast.WalkContinue, nil
	})
	return diags
}

func markerForLevel(level int) string {
	if level == 2 {
		return "**bold**"
	}
	return "*italic*"
}
