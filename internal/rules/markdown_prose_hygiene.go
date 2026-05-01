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
	wordSplit       = regexp.MustCompile(`[A-Za-z]+`)
	mdLinkURL       = regexp.MustCompile(`\]\([^)]*\)`)
	bareURL         = regexp.MustCompile(`https?://\S+`)
	htmlTag         = regexp.MustCompile(`<[^>]+>`)
	inlineCode      = regexp.MustCompile("`[^`]*`")
	spacedColon     = regexp.MustCompile(` : `)
	plusMinus       = regexp.MustCompile(` \+-|\s-\+`)
	hrLine          = regexp.MustCompile(`^\s*-{3,}\s*$`)
	fenceLine       = regexp.MustCompile("^\\s*(```|~~~)")
	underscoreEmph  = regexp.MustCompile(`\s_{1,2}[^\s_][^_\n]*?_{1,2}(\s|[.,;:!?)\]]|$)`)
	reversedLink    = regexp.MustCompile(`\([^)\n]*\)\[[^\]\n]*\]`)
	referenceLink   = regexp.MustCompile(`\[[^)\n]*\]\[[^\]\n]*\]`)
	bulletNoSpace   = regexp.MustCompile(`^ {0,3}[-+*][A-Za-z0-9]`)
	emphasisLine    = regexp.MustCompile(`^ {0,3}\*[^*\s][^*]*\*`)
	blockquoteNoSp  = regexp.MustCompile(`^ {0,3}>[^\s>]`)
	spacedEmph      = regexp.MustCompile(`\*+\s+\S[^*\n]*\S\s+\*+`)
	listItemLine    = regexp.MustCompile(`^ {0,3}[-+*]\s`)
	headingIndented = regexp.MustCompile(`^[ \t]+#{1,6}[ \t]`)
	headingNoSpace  = regexp.MustCompile(`^#{1,6}[^ \t#]`)
	brokenHRDouble  = regexp.MustCompile(`^\s*--\s*$`)
	tripleStarOpen  = regexp.MustCompile(`\*{3}[^\s*][^*\n]*\*`)
	oddListIndent   = regexp.MustCompile(`^(?: |   )[-+*][ \t]`)
	mixedDashes     = regexp.MustCompile(`\x{2014}\x{2013}|\x{2013}\x{2014}|[\x{2014}\x{2013}]{3,}`)
	floatingQuote   = regexp.MustCompile(`(^|\s)"(\s|$)`)
	shortcodeOpen   = regexp.MustCompile(`\{\{[<%][^\s<%}]`)
	shortcodeClose  = regexp.MustCompile(`[^\s<%{][>%]\}\}`)

	missingSpacePunct = regexp.MustCompile(`[a-z][.!?;,][A-Z][a-z]`)
	asymSlash         = regexp.MustCompile(`[A-Za-z]/ [A-Za-z]|[A-Za-z] /[A-Za-z]`)
	paddedQuote       = regexp.MustCompile(`"[ \t]+[^"\n]*?[ \t]+"`)
	spacedPercent     = regexp.MustCompile(`\d %`)
	spacedCurrency    = regexp.MustCompile(`[$£€¥] \d`)
	spacedHash        = regexp.MustCompile(`[^#]# \d`)
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

type literalPattern struct {
	needle string
	msg    string
}

var literalPatterns = []literalPattern{
	{"——", "double em dash"},
	{"---", "literal triple hyphen (use em dash —)"},
	{"''", "double apostrophe"},
	{"``", "double backtick"},
	{"“", "opening smart quote"},
	{"”", "closing smart quote"},
	{"‘", "opening curvy apostrophe"},
	{"’", "closing curvy apostrophe"},
	{",,", "double comma"},
	{".. ", "double period"},
	{" )", "space before closing paren"},
	{"( ", "space after opening paren"},
	{" ,", "space before comma"},
	{" .", "space before period"},
	{".  ", "double space after period"},
	{" !", "space before exclamation mark"},
	{" ?", "space before question mark"},
	{"](//", "protocol-relative link"},
	{` " ](`, "quote glued to link"},
	{`===`, "Setext headers, brittle"},
}

func (markdownProseHygiene) Check(f *MarkdownFile, _ *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
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

		// Invisible / zero-width characters.
		for _, r := range text {
			if name, ok := invisibleChars[r]; ok {
				diags = append(diags, Diagnostic{
					Path: f.Path, Line: line, Rule: "prose-hygiene",
					Message: fmt.Sprintf("invisible character: %s", name),
				})
				break
			}
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
		if mixedDashes.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "malformed dash sequence (mixed em/en or 3+ dashes)",
			})
		}
		if floatingQuote.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: `floating/orphaned quote (")`,
			})
		}
		if shortcodeOpen.MatchString(text) || shortcodeClose.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "Hugo shortcode missing required spaces ({{< name >}})",
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
		if referenceLink.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "Avoid using reference links (use [text](url))",
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
		if headingIndented.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "heading must start at the beginning of the line",
			})
		}
		if headingNoSpace.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "missing space after # in heading",
			})
		}
		if brokenHRDouble.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "broken horizontal rule (use --- not --)",
			})
		}
		if oddListIndent.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "odd indentation before list marker",
			})
		}
		if missingSpacePunct.MatchString(prose) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "missing space after punctuation",
			})
		}
		if asymSlash.MatchString(prose) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "asymmetrical spacing around /",
			})
		}
		if paddedQuote.MatchString(prose) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: `padded spaces inside quotation marks (" word ")`,
			})
		}
		if spacedPercent.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "space before percent sign (10 %)",
			})
		}
		if spacedCurrency.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "space between currency symbol and number ($ 100)",
			})
		}
		if spacedHash.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "space after # before number (# 1, prefer #1)",
			})
		}
		if straightPrimes.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "straight quotes for feet/inches (use ′ ″)",
			})
		}
		if asymHyphen.MatchString(prose) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "asymmetrical spacing around hyphen",
			})
		}
		if hyphenMinus.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "hyphen used as minus sign (use −)",
			})
		}
		if hyphenRange.MatchString(text) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "hyphen in numeric range (use en dash –)",
			})
		}
		for _, m := range tripleStarOpen.FindAllStringIndex(text, -1) {
			end := m[1]
			if end < len(text) && text[end] == '*' {
				continue
			}
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: line, Rule: "prose-hygiene",
				Message: "ambiguous triple-star emphasis (***word*)",
			})
			break
		}
	}
	return diags
}
