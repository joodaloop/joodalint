package rules

import "github.com/joodaloop/hugolint/internal/config"

type Diagnostic struct {
	Path    string
	Line    int
	Rule    string
	Message string
}

type MarkdownFile struct {
	Path    string
	Content []byte
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

type MarkdownRule interface {
	ID() string
	Check(f *MarkdownFile, ctx *MarkdownContext) []Diagnostic
}

type HTMLRule interface {
	ID() string
	Check(f *HTMLFile, ctx *HTMLContext) []Diagnostic
}

var (
	markdownRules []MarkdownRule
	htmlRules     []HTMLRule
)

func RegisterMarkdown(r MarkdownRule) { markdownRules = append(markdownRules, r) }
func RegisterHTML(r HTMLRule)         { htmlRules = append(htmlRules, r) }

func Markdown() []MarkdownRule { return markdownRules }
func HTML() []HTMLRule         { return htmlRules }
