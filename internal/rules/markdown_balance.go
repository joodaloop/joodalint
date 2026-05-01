package rules

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

func init() {
	RegisterMarkdown(&markdownBalance{})
}

type markdownBalance struct{}

func (markdownBalance) ID() string { return "balance" }

type openDelim struct {
	ch   byte
	line int
}

func (markdownBalance) Check(f *MarkdownFile, _ *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
	scanner := bufio.NewScanner(bytes.NewReader(f.Body))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var stack []openDelim
	quoteOpen := false
	quoteLine := 0

	inFence := false
	line := f.BodyStartLine - 1

	for scanner.Scan() {
		line++
		text := scanner.Text()
		trimmed := strings.TrimLeft(text, " \t")
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		inCodeSpan := false
		for i := 0; i < len(text); i++ {
			c := text[i]
			if c == '\\' {
				i++
				continue
			}

			if c == '`' {
				inCodeSpan = !inCodeSpan
				continue
			}
			if inCodeSpan {
				continue
			}

			switch c {
			case '(', '[', '{':
				stack = append(stack, openDelim{ch: c, line: line})
			case ')', ']', '}':
				want := matchOpener(c)
				if len(stack) == 0 {
					diags = append(diags, Diagnostic{
						Path: f.Path, Line: line, Rule: "balance",
						Message: fmt.Sprintf("unmatched closing %q", c),
					})
				} else {
					top := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					if top.ch != want {
						diags = append(diags, Diagnostic{
							Path: f.Path, Line: line, Rule: "balance",
							Message: fmt.Sprintf("mismatched: opened %q on line %d, closed with %q", top.ch, top.line, c),
						})
					}
				}
			case '"':
				if quoteOpen {
					quoteOpen = false
				} else {
					quoteOpen = true
					quoteLine = line
				}
			}
		}
	}

	for _, o := range stack {
		diags = append(diags, Diagnostic{
			Path: f.Path, Line: o.line, Rule: "balance",
			Message: fmt.Sprintf("unclosed %q", o.ch),
		})
	}
	if quoteOpen {
		diags = append(diags, Diagnostic{
			Path: f.Path, Line: quoteLine, Rule: "balance",
			Message: `unbalanced '"' (odd count)`,
		})
	}
	return diags
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
