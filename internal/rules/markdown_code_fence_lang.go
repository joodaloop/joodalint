package rules

import (
	"bufio"
	"bytes"
	"strings"
)

func init() {
	RegisterMarkdown(&markdownCodeFenceLang{})
}

type markdownCodeFenceLang struct{}

func (markdownCodeFenceLang) ID() string { return "code-fence-lang" }

func (markdownCodeFenceLang) Check(f *MarkdownFile, _ *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
	scanner := bufio.NewScanner(bytes.NewReader(f.Content))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	inFence := false
	fenceChar := byte(0)
	fenceLen := 0
	fenceLine := 0

	line := 0
	for scanner.Scan() {
		line++
		trim := strings.TrimLeft(scanner.Text(), " \t")
		if len(trim) < 3 {
			continue
		}
		c := trim[0]
		if c != '`' && c != '~' {
			continue
		}
		n := 0
		for n < len(trim) && trim[n] == c {
			n++
		}
		if n < 3 {
			continue
		}

		rest := strings.TrimSpace(trim[n:])

		if inFence {
			if c == fenceChar && n >= fenceLen && rest == "" {
				inFence = false
			}
			continue
		}

		inFence = true
		fenceChar = c
		fenceLen = n
		fenceLine = line
		if rest == "" {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "code-fence-lang",
				Message: "code fence missing language tag",
			})
		}
	}
	if inFence {
		diags = append(diags, Diagnostic{
			Path: f.Path, Line: fenceLine, Rule: "code-fence-unclosed",
			Message: "code fence opened but never closed",
		})
	}
	return diags
}
