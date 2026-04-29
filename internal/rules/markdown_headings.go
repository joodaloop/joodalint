package rules

import (
	"bufio"
	"bytes"
	"strings"
)

func init() {
	RegisterMarkdown(&markdownHeadings{})
}

type markdownHeadings struct{}

func (markdownHeadings) ID() string { return "headings" }

func (markdownHeadings) Check(f *MarkdownFile, _ *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
	scanner := bufio.NewScanner(bytes.NewReader(f.Content))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	inFrontmatter := false
	prevText := ""
	prevTextLine := 0
	line := 0

	for scanner.Scan() {
		line++
		text := scanner.Text()
		trimmed := strings.TrimSpace(text)

		if line == 1 && trimmed == "---" {
			inFrontmatter = true
			prevText = ""
			prevTextLine = 0
			continue
		}
		if inFrontmatter {
			if trimmed == "---" {
				inFrontmatter = false
			}
			continue
		}

		leftTrimmed := strings.TrimLeft(text, " \t")
		indent := len(text) - len(leftTrimmed)
		if indent <= 3 {
			if c, n, ok := markdownFence(leftTrimmed); ok {
				if inFence {
					if c == fenceChar && n >= fenceLen && strings.TrimSpace(leftTrimmed[n:]) == "" {
						inFence = false
					}
				} else {
					inFence = true
					fenceChar = c
					fenceLen = n
				}
				prevText = ""
				prevTextLine = 0
				continue
			}
		}
		if inFence {
			continue
		}

		if indent > 0 && indent <= 3 && (isATXHeading(leftTrimmed) || isATXHeadingMissingSpace(leftTrimmed, indent)) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "headings",
				Message: "headings must start at the beginning of the line",
			})
		}

		if isATXHeadingMissingSpace(leftTrimmed, indent) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "headings",
				Message: "no space after # in heading",
			})
		}

		if isATXH1(leftTrimmed, indent) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "headings",
				Message: "h1 headings are not allowed",
			})
		}

		if trimmed == "" {
			prevText = ""
			prevTextLine = 0
			continue
		}

		if isSetextH1Underline(leftTrimmed, indent) && prevText != "" {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: prevTextLine, Rule: "headings",
				Message: "h1 headings are not allowed",
			})
			prevText = ""
			prevTextLine = 0
			continue
		}

		prevText = text
		prevTextLine = line
	}

	return diags
}

func markdownFence(trim string) (byte, int, bool) {
	if len(trim) < 3 {
		return 0, 0, false
	}
	c := trim[0]
	if c != '`' && c != '~' {
		return 0, 0, false
	}
	n := 0
	for n < len(trim) && trim[n] == c {
		n++
	}
	if n < 3 {
		return 0, 0, false
	}
	return c, n, true
}

func isATXHeading(trim string) bool {
	if len(trim) == 0 || trim[0] != '#' {
		return false
	}
	i := 0
	for i < len(trim) && trim[i] == '#' {
		i++
	}
	if i > 6 {
		return false
	}
	if i == len(trim) {
		return true
	}
	return trim[i] == ' ' || trim[i] == '\t'
}

func isATXH1(trim string, indent int) bool {
	if indent > 3 || len(trim) < 2 || trim[0] != '#' || trim[1] == '#' {
		return false
	}
	return trim[1] == ' ' || trim[1] == '\t'
}

func isATXHeadingMissingSpace(trim string, indent int) bool {
	if indent > 3 || len(trim) < 2 || trim[0] != '#' {
		return false
	}
	i := 0
	for i < len(trim) && trim[i] == '#' {
		i++
	}
	if i == 0 || i >= len(trim) {
		return false
	}
	return trim[i] != ' ' && trim[i] != '\t'
}

func isSetextH1Underline(trim string, indent int) bool {
	if indent > 3 || trim == "" {
		return false
	}
	for i := 0; i < len(trim); i++ {
		if trim[i] != '=' {
			return false
		}
	}
	return true
}
