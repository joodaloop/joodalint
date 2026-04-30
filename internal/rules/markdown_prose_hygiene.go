package rules

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

func init() {
	RegisterMarkdown(&markdownProseHygiene{})
}

type markdownProseHygiene struct{}

func (markdownProseHygiene) ID() string { return "prose-hygiene" }

var (
	wordSplit      = regexp.MustCompile(`[A-Za-z]+`)
	mdLinkURL      = regexp.MustCompile(`\]\([^)]*\)`)
	bareURL        = regexp.MustCompile(`https?://\S+`)
	htmlTag        = regexp.MustCompile(`<[^>]+>`)
	inlineCode     = regexp.MustCompile("`[^`]*`")
	spacedColon    = regexp.MustCompile(` : `)
	plusMinus      = regexp.MustCompile(` \+-|\s-\+`)
	hrLine         = regexp.MustCompile(`^\s*-{3,}\s*$`)
	fenceLine      = regexp.MustCompile("^\\s*(```|~~~)")
	underscoreEmph = regexp.MustCompile(`\s_{1,2}[^\s_][^_\n]*?_{1,2}(\s|[.,;:!?)\]]|$)`)
	reversedLink   = regexp.MustCompile(`\([^)\n]*\)\[[^\]\n]*\]`)
	bulletNoSpace  = regexp.MustCompile(`^ {0,3}[-+*][A-Za-z0-9]`)
	emphasisLine   = regexp.MustCompile(`^ {0,3}\*[^*\s][^*]*\*`)
	blockquoteNoSp = regexp.MustCompile(`^ {0,3}>[^\s>]`)
	spacedEmph     = regexp.MustCompile(`\*+\s+\S[^*\n]*\S\s+\*+`)
	listItemLine   = regexp.MustCompile(`^ {0,3}[-+*]\s`)
)

type literalPattern struct {
	needle string
	msg    string
}

var literalPatterns = []literalPattern{
	{"——", "double em dash"},
	{"---", "literal triple hyphen (use em dash —)"},
	{"''", "double apostrophe"},
	{"``", "double backtick"},
	{" )", "space before closing paren"},
	{"](//", "protocol-relative link"},
	{` " ](`, "quote glued to link"},
}

func (markdownProseHygiene) Check(f *MarkdownFile, _ *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
	scanner := bufio.NewScanner(bytes.NewReader(f.Content))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	inFence := false
	inFrontmatter := false
	inRawBlock := "" // "style" or "script" while inside; "" otherwise
	line := 0

	for scanner.Scan() {
		line++
		text := scanner.Text()

		// Frontmatter: opens with `---` on line 1, closes at next `---`.
		if line == 1 && strings.TrimSpace(text) == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter {
			if strings.TrimSpace(text) == "---" {
				inFrontmatter = false
			}
			continue
		}

		// Fenced code blocks.
		if fenceLine.MatchString(text) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}

		// Raw HTML <style>/<script> blocks (markdown allows inline HTML).
		lower := strings.ToLower(text)
		if inRawBlock != "" {
			if strings.Contains(lower, "</"+inRawBlock+">") {
				inRawBlock = ""
			}
			continue
		}
		if strings.Contains(lower, "<style") {
			if !strings.Contains(lower, "</style>") {
				inRawBlock = "style"
			}
			continue
		}
		if strings.Contains(lower, "<script") {
			if !strings.Contains(lower, "</script>") {
				inRawBlock = "script"
			}
			continue
		}

		// Word repetition. Strip link URLs, inline code, and HTML tags so
		// `[Foo](/foo)` and `<li>...</li>` don't tokenize as repeated words.
		prose := mdLinkURL.ReplaceAllString(text, "]")
		prose = bareURL.ReplaceAllString(prose, " ")
		prose = inlineCode.ReplaceAllString(prose, " ")
		prose = htmlTag.ReplaceAllString(prose, " ")
		idx := wordSplit.FindAllStringIndex(prose, -1)
		for i := 1; i < len(idx); i++ {
			gap := prose[idx[i-1][1]:idx[i][0]]
			if !strings.ContainsAny(gap, " \t") {
				continue
			}
			if strings.ContainsAny(gap, ".!?,;:&([])") {
				continue
			}
			a := strings.ToLower(prose[idx[i-1][0]:idx[i-1][1]])
			b := strings.ToLower(prose[idx[i][0]:idx[i][1]])
			if a != b {
				continue
			}
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: fmt.Sprintf("repeated word: %q", a+" "+a),
			})
		}

		// Literal substring patterns.
		for _, p := range literalPatterns {
			if strings.Contains(text, p.needle) {
				diags = append(diags, Diagnostic{
					Path: f.Path, Line: line, Rule: "prose-hygiene",
					Message: fmt.Sprintf("%s: %q", p.msg, p.needle),
				})
			}
		}

		// Skip horizontal-rule lines for the dash-heavy regex checks below
		// (the literal `---` check already fires only when other content is on the line —
		// strings.Contains on a pure `---` line will still match, so guard explicitly).
		if hrLine.MatchString(text) {
			// Strip the false positive emitted above for `---` on a pure HR line.
			if n := len(diags); n > 0 && strings.Contains(diags[n-1].Message, `"---"`) {
				diags = diags[:n-1]
			}
			continue
		}

		// Regex spacing patterns.
		if spacedColon.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "spaced colon ( : )",
			})
		}
		if plusMinus.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "malformed plus-minus (use ±)",
			})
		}
		if reversedLink.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "reversed link syntax (use [text](url))",
			})
		}
		if bulletNoSpace.MatchString(text) && !emphasisLine.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "list bullet without space after marker",
			})
		}
		if blockquoteNoSp.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "blockquote > without space after",
			})
		}
		if !listItemLine.MatchString(text) && spacedEmph.MatchString(prose) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "spaces inside emphasis markers (* text *)",
			})
		}
		if underscoreEmph.MatchString(" " + prose) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "underscore emphasis (use * instead)",
			})
		}
	}
	return diags
}
