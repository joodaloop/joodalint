package rules

import (
	"bytes"
	"sync"

	"github.com/joodaloop/hugolint/internal/config"
	"github.com/yuin/goldmark/ast"
)

type Diagnostic struct {
	Path    string
	Line    int
	Rule    string
	Message string
}

type MarkdownFile struct {
	Path          string
	Content       []byte
	Body          []byte
	AST           ast.Node
	BodyStartLine int
	// ProseBlocks is the post-parse flattening of inline prose: each
	// top-level prose block (paragraph, heading, list-item text block,
	// HTML block) becomes one entry. Code blocks, code spans, link URLs,
	// and autolink targets are excluded; link text, image alt text, and
	// raw inline HTML are included. Spans carry byte offsets into Body
	// so callers resolve lines via LineAt at diagnostic time.
	ProseBlocks []ProseBlock
}

type ProseSpan struct {
	Text   []byte
	Offset int // byte offset into MarkdownFile.Body
}

type ProseBlock struct {
	Spans []ProseSpan
}

func (f *MarkdownFile) LineAt(offset int) int {
	if offset < 0 {
		offset = 0
	}
	if offset > len(f.Body) {
		offset = len(f.Body)
	}
	return bytes.Count(f.Body[:offset], []byte("\n")) + f.BodyStartLine
}

func (f *MarkdownFile) NodeLine(n ast.Node) int {
	if start, ok := earliestTextStart(n); ok {
		return f.LineAt(start)
	}
	for p := n; p != nil; p = p.Parent() {
		if p.Type() == ast.TypeBlock {
			if lines := p.Lines(); lines != nil && lines.Len() > 0 {
				return f.LineAt(lines.At(0).Start)
			}
		}
	}
	return f.BodyStartLine
}

func earliestTextStart(n ast.Node) (int, bool) {
	start, found := 0, false
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := c.(*ast.Text); ok {
			if !found || t.Segment.Start < start {
				start = t.Segment.Start
				found = true
			}
		}
		return ast.WalkContinue, nil
	})
	return start, found
}

type Asset struct {
	Tag  string
	Attr string
	URL  string
}

type MetaTag struct {
	Name     string // value of name=""
	Property string // value of property=""
	HTTPEquiv string // value of http-equiv=""
	Charset  string // value of charset=""
	Content  string // value of content=""
}

type HeadLink struct {
	Rel   string
	Type  string
	Href  string
	Title string
}

type HTMLFile struct {
	Path      string
	URLPath   string
	Links     []string
	Images    []string
	Assets    []Asset
	IDs       map[string]int
	Title     string
	Lang      string
	Metas     []MetaTag
	HeadLinks []HeadLink
	// Text is the concatenated text content of the document, excluding
	// content inside <script>, <style>, <pre>, and <code>. Comments are
	// also excluded (the tokenizer strips them).
	Text string
}

type HTMLContext struct {
	Root    string
	Pages   map[string]bool
	PageIDs map[string]map[string]int
	Config  *config.Config

	linkMu      sync.Mutex
	LinkedPages map[string]bool
}

func (c *HTMLContext) MarkLinked(target string) {
	c.linkMu.Lock()
	defer c.linkMu.Unlock()
	if c.LinkedPages == nil {
		c.LinkedPages = map[string]bool{}
	}
	c.LinkedPages[target] = true
}

type MarkdownContext struct {
	Config *config.Config
}

type FrontmatterFile struct {
	Path     string
	Parsed   map[string]any
	ParseErr error
	Line0    int
}

type FrontmatterContext struct {
	Config *config.Config
}

type MarkdownRule interface {
	ID() string
	Check(f *MarkdownFile, ctx *MarkdownContext) []Diagnostic
}

type FrontmatterRule interface {
	ID() string
	Check(f *FrontmatterFile, ctx *FrontmatterContext) []Diagnostic
}

type MarkdownASTRule interface {
	ID() string
	Check(f *MarkdownFile, ctx *MarkdownContext) []Diagnostic
}

type MarkdownTextRule interface {
	ID() string
	Check(f *MarkdownFile, ctx *MarkdownContext) []Diagnostic
}

type HTMLRule interface {
	ID() string
	Check(f *HTMLFile, ctx *HTMLContext) []Diagnostic
}

var (
	markdownRules    []MarkdownRule
	frontmatterRules []FrontmatterRule
	astRules         []MarkdownASTRule
	textRules        []MarkdownTextRule
	htmlRules        []HTMLRule
)

func RegisterMarkdown(r MarkdownRule)         { markdownRules = append(markdownRules, r) }
func RegisterFrontmatter(r FrontmatterRule)   { frontmatterRules = append(frontmatterRules, r) }
func RegisterMarkdownAST(r MarkdownASTRule)   { astRules = append(astRules, r) }
func RegisterMarkdownText(r MarkdownTextRule) { textRules = append(textRules, r) }
func RegisterHTML(r HTMLRule)                 { htmlRules = append(htmlRules, r) }

func Markdown() []MarkdownRule         { return markdownRules }
func Frontmatter() []FrontmatterRule   { return frontmatterRules }
func MarkdownAST() []MarkdownASTRule   { return astRules }
func MarkdownText() []MarkdownTextRule { return textRules }
func HTML() []HTMLRule                 { return htmlRules }
