package rules

import (
	"bufio"
	"bytes"
	"fmt"
	"sync"
)

func init() {
	RegisterMarkdown(&markdownSpelling{})
}

type markdownSpelling struct {
	errOnce sync.Once
}

func (*markdownSpelling) ID() string { return "spelling" }

func (m *markdownSpelling) Check(f *MarkdownFile, ctx *MarkdownContext) []Diagnostic {
	if ctx == nil || ctx.Config == nil || ctx.Config.Spelling.Dict == "" {
		return nil
	}
	sharedSpeller.ensureInit(ctx.Config)
	if sharedSpeller.initErr != nil {
		var d []Diagnostic
		m.errOnce.Do(func() {
			d = []Diagnostic{{Path: f.Path, Rule: "spelling", Message: sharedSpeller.initErr.Error()}}
		})
		return d
	}
	if !sharedSpeller.enabled {
		return nil
	}

	unknown, err := sharedSpeller.unknown(f.Body, "markdown")
	if err != nil {
		return []Diagnostic{{Path: f.Path, Rule: "spelling", Message: err.Error()}}
	}
	if len(unknown) == 0 {
		return nil
	}

	// Locate each unknown word's first occurrence in the original content
	// for line-numbered diagnostics.
	wordRe := buildWordRegex(unknown)
	var diags []Diagnostic
	seen := map[string]bool{}
	lineScanner := bufio.NewScanner(bytes.NewReader(f.Body))
	lineScanner.Buffer(make([]byte, 64*1024), 1024*1024)
	line := f.BodyStartLine - 1
	for lineScanner.Scan() {
		line++
		for _, m := range wordRe.FindAllString(lineScanner.Text(), -1) {
			if seen[m] {
				continue
			}
			seen[m] = true
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "spelling",
				Message: fmt.Sprintf("unknown word: %q", m),
			})
		}
	}
	return diags
}
