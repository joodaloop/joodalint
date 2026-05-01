package rules

import "fmt"

func init() {
	RegisterMarkdown(&markdownBalance{})
}

type markdownBalance struct{}

func (markdownBalance) ID() string { return "balance" }

type openDelim struct {
	ch     byte
	offset int
}

// Quote balance lives in markdown_prose_ast.go (proseQuoteChecks)
// alongside the other quote-spacing checks; this rule sticks to
// bracket/paren/brace pairing.
func (markdownBalance) Check(f *MarkdownFile, _ *MarkdownContext) []Diagnostic {
	var diags []Diagnostic

	for _, blk := range f.ProseBlocks {
		var stack []openDelim

		for _, sp := range blk.Spans {
			text := sp.Text
			for i := 0; i < len(text); i++ {
				c := text[i]
				if c == '\\' {
					i++
					continue
				}
				switch c {
				case '(', '[', '{':
					stack = append(stack, openDelim{ch: c, offset: sp.Offset + i})
			case ')', ']', '}':
				want := matchOpener(c)
				if len(stack) == 0 {
					if c == ')' && (isNumberedListClose(text, i) || isEmoticonClose(text, i)) {
						continue
					}
						diags = append(diags, Diagnostic{
							Path: f.Path, Line: f.LineAt(sp.Offset + i), Rule: "balance",
							Message: fmt.Sprintf("unmatched closing %q", c),
						})
					} else {
						top := stack[len(stack)-1]
						stack = stack[:len(stack)-1]
						if top.ch != want {
							diags = append(diags, Diagnostic{
								Path: f.Path, Line: f.LineAt(sp.Offset + i), Rule: "balance",
								Message: fmt.Sprintf("mismatched: opened %q on line %d, closed with %q", top.ch, f.LineAt(top.offset), c),
							})
						}
					}
				}
			}
		}

		for _, o := range stack {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: f.LineAt(o.offset), Rule: "balance",
				Message: fmt.Sprintf("unclosed %q", o.ch),
			})
		}
	}
	return diags
}

func isNumberedListClose(text []byte, i int) bool {
	if i == 0 {
		return false
	}
	digitCount := 0
	for j := i - 1; j >= 0; j-- {
		if text[j] >= '0' && text[j] <= '9' {
			digitCount++
			continue
		}
		if text[j] == ' ' || text[j] == '\t' {
			return digitCount > 0
		}
		return false
	}
	return digitCount > 0
}

func isEmoticonClose(text []byte, i int) bool {
	return i > 0 && (text[i-1] == ':' || text[i-1] == ';')
}

func matchOpener(closer byte) byte {
	switch closer {
	case ')':
		return '('
	case ']':
		return '['
	case '}':
		return '{'
	}
	return 0
}
