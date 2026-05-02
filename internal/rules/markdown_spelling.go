package rules

import (
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

	// Build a single buffer of post-parse prose. Spans are separated by
	// blank lines so words can't merge across boundaries; aspell stops
	// reading at NUL on stdin, so we must avoid that byte.
	var buf bytes.Buffer
	for _, blk := range f.ProseBlocks {
		for _, sp := range blk.Spans {
			buf.Write(sp.Text)
		}
		buf.WriteString("\n\n")
	}
	if buf.Len() == 0 {
		return nil
	}

	unknown, err := sharedSpeller.unknown(buf.Bytes())
	if err != nil {
		return []Diagnostic{{Path: f.Path, Rule: "spelling", Message: err.Error()}}
	}
	if len(unknown) == 0 {
		return nil
	}

	wordRe := buildWordRegex(unknown)
	var diags []Diagnostic
	seen := map[string]bool{}
	for _, blk := range f.ProseBlocks {
		for _, sp := range blk.Spans {
			for _, m := range wordRe.FindAllIndex(sp.Text, -1) {
				word := string(sp.Text[m[0]:m[1]])
				if seen[word] {
					continue
				}
				seen[word] = true
				diags = append(diags, Diagnostic{
					Path: f.Path, Line: f.LineAt(sp.Offset + m[0]), Rule: "spelling",
					Message: fmt.Sprintf("unknown word: %q", word),
				})
			}
		}
	}
	return diags
}
