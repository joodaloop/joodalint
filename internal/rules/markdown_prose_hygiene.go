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

// Regexes are split between this file and markdown_prose_ast.go: the
// structural/line-shape patterns stay here because they exist precisely
// to catch syntax that the AST won't recognise as the construct the
// author was attempting. The content-prose patterns live alongside
// the AST rule.
var (
	wordSplit         = regexp.MustCompile(`[A-Za-z]+`)
	spacedColon       = regexp.MustCompile(` : `)
	plusMinus         = regexp.MustCompile(` \+-|\s-\+`)
	fenceLine         = regexp.MustCompile("^\\s*(```|~~~)")
	underscoreEmph    = regexp.MustCompile(`\s_{1,2}[^\s_][^_\n]*?_{1,2}(\s|[.,;:!?)\]]|$)`)
	reversedLink      = regexp.MustCompile(`\([^)\n]*\)\[[^\]\n]*\]`)
	referenceLink     = regexp.MustCompile(`\[[^)\n]*\]\[[^\]\n]*\]`)
	bulletNoSpace     = regexp.MustCompile(`^ {0,3}[-+*][A-Za-z0-9]`)
	emphasisLine      = regexp.MustCompile(`^ {0,3}\*[^*\s][^*]*\*`)
	blockquoteNoSp    = regexp.MustCompile(`^ {0,3}>[^\s>]`)
	spacedEmph        = regexp.MustCompile(`\*+\s+\S[^*\n]*\S\s+\*+`)
	listItemLine      = regexp.MustCompile(`^ {0,3}[-+*]\s`)
	headingIndented   = regexp.MustCompile(`^[ \t]+#{1,6}[ \t]`)
	headingNoSpace    = regexp.MustCompile(`^#{1,6}[^ \t#]`)
	brokenHRDouble    = regexp.MustCompile(`^\s*--\s*$`)
	oddListIndent     = regexp.MustCompile(`^(?: |   )[-+*][ \t]`)
	mixedDashes       = regexp.MustCompile(`\x{2014}\x{2013}|\x{2013}\x{2014}|[\x{2014}\x{2013}]{3,}`)
	spacesAroundQuote = regexp.MustCompile(`[ \t]"[ \t]`)
	orphanedQuote     = regexp.MustCompile(`(^|[ \t])"$|^"([ \t]|$)`)
	shortcodeOpen     = regexp.MustCompile(`\{\{[<%][^\s<%}]`)
	shortcodeClose    = regexp.MustCompile(`[^\s<%{][>%]\}\}`)

	missingSpacePunct = regexp.MustCompile(`[a-z][.!?;,][A-Z][a-z]`)
	asymSlash         = regexp.MustCompile(`[A-Za-z]/ [A-Za-z]|[A-Za-z] /[A-Za-z]`)
	paddedQuote       = regexp.MustCompile(`(^|\s)"[ \t]+[^"\n]*?[ \t]+"(\s|[.,;:!?)\]]|$)`)
	spacedPercent     = regexp.MustCompile(`\d %`)
	spacedCurrency    = regexp.MustCompile(`[$£€¥] \d`)
	spacedHash        = regexp.MustCompile(`[^#]# \d`)
	spacedDashNum     = regexp.MustCompile(`[–-] \d`)
	straightPrimes    = regexp.MustCompile(`\d'\d+"`)
	asymHyphen        = regexp.MustCompile(`[A-Za-z]- [A-Za-z]|[A-Za-z] -[A-Za-z]`)
	hyphenMinus       = regexp.MustCompile(`(?:^|\s)-\d+(?:\s|$|[.,;:!?])`)
	hyphenRange       = regexp.MustCompile(`(?:^|[^-\d.])\d+-\d+(?:[^-\d.]|$)`)
)

var invisibleChars = map[rune]string{
	'\u200B': "zero-width space (U+200B)",
	'\u200C': "zero-width non-joiner (U+200C)",
	'\u200D': "zero-width joiner (U+200D)",
	'\u200E': "left-to-right mark (U+200E)",
	'\u200F': "right-to-left mark (U+200F)",
	'\u2060': "word joiner (U+2060)",
	'\uFEFF': "byte-order mark (U+FEFF)",
	'\u00AD': "soft hyphen (U+00AD)",
	'\u00A0': "non-breaking space (U+00A0)",
}

// nameC1Control returns a descriptive name for any C1 control character
// (U+0080\u2013U+009F). These are non-printable in Unicode but reused by
// Windows-1252 for punctuation (curly quotes, em/en dashes, ellipsis);
// when Win-1252 text is pasted into UTF-8 without conversion, the bytes
// survive as these control codepoints \u2014 a reliable mojibake signature.
func nameC1Control(r rune) (string, bool) {
	if r < 0x0080 || r > 0x009F {
		return "", false
	}
	return fmt.Sprintf("C1 control character (U+%04X, likely Windows-1252 mojibake)", r), true
}

// literalPatterns covers structural/source-level needles that must be
// scanned line-by-line: link/HR syntax that the AST won't expose as a
// "broken" version of the construct, plus the brittle setext header
// marker. Content-level needles (e.g. " ,", "——") live in
// markdown_prose_ast.go and run against ProseBlock spans.
var literalPatterns = []taggedPattern{
	{"](//", "link-style", "protocol-relative link"},
	{` " ](`, "link-style", "quote glued to link"},
	{`===`, "md-syntax", "Setext headers, brittle"},
}

func (markdownProseHygiene) Check(f *MarkdownFile, _ *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
	emit := func(line int, rule, msg string) {
		diags = append(diags, Diagnostic{Path: f.Path, Line: line, Rule: rule, Message: msg})
	}
	scanner := bufio.NewScanner(bytes.NewReader(f.Body))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	inFence := false
	inRawBlock := "" // "style" or "script" while inside; "" otherwise
	line := f.BodyStartLine - 1

	for scanner.Scan() {
		line++
		text := scanner.Text()

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

		// Invisible / zero-width characters.
		runes := []rune(text)
		for i, r := range runes {
			name, ok := invisibleChars[r]
			if !ok {
				name, ok = nameC1Control(r)
			}
			if ok {
				// U+200D is used legitimately in emoji ZWJ sequences
				// (e.g. 👨‍💻). Only flag it when not between emoji-like
				// codepoints.
				if r == '\u200D' && i > 0 && i < len(runes)-1 &&
					runes[i-1] > 0x7F && runes[i+1] > 0x7F {
					continue
				}
				emit(line, "invisible-chars", fmt.Sprintf("invisible character: %s", name))
				break
			}
		}

		// Literal substring patterns (structural needles only).
		for _, p := range literalPatterns {
			if strings.Contains(text, p.needle) {
				emit(line, p.rule, fmt.Sprintf("%s: %q", p.msg, p.needle))
			}
		}

		// Hugo shortcode spacing — keep line-based; applies uniformly.
		if strings.Contains(text, "{{") && (shortcodeOpen.MatchString(text) || shortcodeClose.MatchString(text)) {
			emit(line, "shortcode", "Hugo shortcode missing required spaces ({{< name >}})")
		}
		if strings.ContainsAny(text, "([") && reversedLink.MatchString(text) {
			emit(line, "link-style", "reversed link syntax (use [text](url))")
		}
		if strings.Count(text, "[") >= 2 && referenceLink.MatchString(text) {
			emit(line, "link-style", "Avoid using reference links (use [text](url))")
		}
		if len(text) > 0 && strings.ContainsAny(text[:min(4, len(text))], "-+*") &&
			bulletNoSpace.MatchString(text) && !emphasisLine.MatchString(text) {
			emit(line, "md-syntax", "list bullet without space after marker")
		}
		if strings.Contains(text, ">") && blockquoteNoSp.MatchString(text) {
			emit(line, "md-syntax", "blockquote > without space after")
		}
		if len(text) > 0 && (text[0] == ' ' || text[0] == '\t') && strings.Contains(text, "#") && headingIndented.MatchString(text) {
			emit(line, "md-syntax", "heading must start at the beginning of the line")
		}
		if strings.HasPrefix(text, "#") && headingNoSpace.MatchString(text) {
			emit(line, "md-syntax", "missing space after # in heading")
		}
		if strings.Contains(text, "--") && brokenHRDouble.MatchString(text) {
			emit(line, "md-syntax", "broken horizontal rule (use --- not --)")
		}
		if len(text) > 0 && (text[0] == ' ' || text[0] == '\t') && strings.ContainsAny(text, "-+*") && oddListIndent.MatchString(text) {
			emit(line, "md-syntax", "odd indentation before list marker")
		}
		if strings.Contains(text, "'") && strings.Contains(text, "\"") && straightPrimes.MatchString(text) {
			emit(line, "quotes", "straight quotes for feet/inches (use ′ ″)")
		}
	}
	return diags
}
