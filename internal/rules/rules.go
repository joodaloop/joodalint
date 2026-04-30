package rules

import (
	"bytes"

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
	if t, ok := n.(*ast.Text); ok {
		return f.LineAt(t.Segment.Start)
	}
	if lines := n.Lines(); lines != nil && lines.Len() > 0 {
		return f.LineAt(lines.At(0).Start)
	}
	return f.BodyStartLine
}

type Asset struct {
	Tag  string
	Attr string
	URL  string
}

type HTMLFile struct {
	Path    string
	URLPath string
	Links   []string
	Images  []string
	Assets  []Asset
	IDs     map[string]int
	// Text is the concatenated text content of the document, excluding
	// content inside <script>, <style>, <pre>, and <code>. Comments are
	// also excluded (the tokenizer strips them).
	Text string
}

type HTMLContext struct {
	Root    string
	Pages   map[string]bool
	PageIDs map[string]map[string]int
}

type MarkdownContext struct {
	Config *config.Config
}

type FrontmatterFile struct {
	Path   string
	Raw    []byte
	Parsed map[string]any
	Line0  int
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
