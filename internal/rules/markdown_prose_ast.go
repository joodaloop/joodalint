package rules

import (
	"fmt"
	"strings"
)

func init() {
	RegisterMarkdownAST(&markdownProseHygieneAST{})
}

// markdownProseHygieneAST runs the prose-content half of the prose-hygiene
// rule against ProseBlocks rather than raw lines. The structural half
// (broken syntax, indented headings, malformed bullets, etc.) lives in
// markdownProseHygiene because those defects are precisely the cases where
// the AST won't surface the construct the author was trying to write.
//
// Diagnostics are tagged by the kind of defect they describe — quotes,
// spacing, emphasis, typography, number-format, repeated-word — not by
// which of the two implementations found them, so a tag means the same
// thing regardless of whether the check is line-based or AST-based.
// These tags are the user-facing names checks can be disabled by; treat
// renaming one as a breaking change.
type markdownProseHygieneAST struct{}

func (markdownProseHygieneAST) ID() string { return "prose-hygiene" }

// taggedPattern is a substring needle reported verbatim under a specific
// rule tag. Shared by both halves of the prose checks; the HTML rules use
// the untagged literalPattern instead.
type taggedPattern struct {
	needle string
	rule   string
	msg    string
}

var astLiteralPatterns = []taggedPattern{
	{"——", "typography", "double em dash"},
	{"---", "typography", "literal triple hyphen (use em dash —)"},
	{"''", "quotes", "double apostrophe"},
	{"``", "emphasis", "double backtick"},
	{"“", "quotes", "opening smart quote"},
	{"”", "quotes", "closing smart quote"},
	{",,", "spacing", "double comma"},
	{" )", "spacing", "space before closing paren"},
	{"( ", "spacing", "space after opening paren"},
	{" ,", "spacing", "space before comma"},
	{" . ", "spacing", "space around period"},
	{" ?", "spacing", "space before question mark"},
	{"**", "emphasis", "unescaped bold markers (**)"},
	{"~~", "emphasis", "unescaped strikethrough markers"},
	{"==", "emphasis", "unescaped highlight markers (==)"},
	{"__", "emphasis", "unescaped emphasis markers (__)"},
}

func (markdownProseHygieneAST) Check(f *MarkdownFile, _ *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
	for _, blk := range f.ProseBlocks {
		diags = append(diags, proseBlockChecks(f, blk)...)
		for _, sp := range blk.Spans {
			diags = append(diags, proseSpanChecks(f, sp)...)
		}
		diags = append(diags, proseQuoteChecks(f, blk)...)
	}
	return diags
}

// blockText concatenates a block's spans into a single string. Only
// whitespace bytes from the original body that fall between consecutive
// spans are inserted as separators — no synthetic whitespace is added.
type blockText struct {
	text string
	segs []blockSeg
}

type blockSeg struct {
	concatStart int
	bodyOffset  int
	length      int
}

func newBlockText(blk ProseBlock, body []byte) blockText {
	var sb strings.Builder
	var segs []blockSeg
	var prevEnd int
	for _, sp := range blk.Spans {
		if sb.Len() > 0 {
			for _, b := range body[prevEnd:sp.Offset] {
				if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
					sb.WriteByte(b)
				}
			}
		}
		segs = append(segs, blockSeg{
			concatStart: sb.Len(),
			bodyOffset:  sp.Offset,
			length:      len(sp.Text),
		})
		sb.Write(sp.Text)
		prevEnd = sp.Offset + len(sp.Text)
	}
	return blockText{text: sb.String(), segs: segs}
}

func (bt blockText) bodyOff(p int) int {
	for i := len(bt.segs) - 1; i >= 0; i-- {
		if bt.segs[i].concatStart <= p {
			inSeg := p - bt.segs[i].concatStart
			if inSeg > bt.segs[i].length {
				inSeg = bt.segs[i].length
			}
			return bt.segs[i].bodyOffset + inSeg
		}
	}
	if len(bt.segs) > 0 {
		return bt.segs[0].bodyOffset
	}
	return 0
}

// maskBareURLs replaces any http(s):// URL inside text with spaces of
// the same length. Goldmark's default parser leaves bare URLs in Text
// nodes; without masking, the spacing/punctuation regexes here would
// fire on URL paths.
func maskBareURLs(text string) string {
	if !strings.Contains(text, "://") {
		return text
	}
	b := []byte(text)
	for i := 0; i < len(b); i++ {
		if b[i] != 'h' && b[i] != 'H' {
			continue
		}
		rest := b[i:]
		var prefix int
		switch {
		case len(rest) >= 7 && strings.EqualFold(string(rest[:7]), "http://"):
			prefix = 7
		case len(rest) >= 8 && strings.EqualFold(string(rest[:8]), "https://"):
			prefix = 8
		default:
			continue
		}
		k := i + prefix
		for k < len(b) && b[k] > ' ' && b[k] != ')' && b[k] != ']' && b[k] != '>' && b[k] != '"' {
			k++
		}
		for m := i; m < k; m++ {
			b[m] = ' '
		}
		i = k - 1
	}
	return string(b)
}

// proseBlockChecks runs checks that need cross-span context or that
// would false-positive on bare URLs. Spans are concatenated using only
// whitespace bytes from the source as separators, then any http(s) URL
// is masked out.
func proseBlockChecks(f *MarkdownFile, blk ProseBlock) []Diagnostic {
	if len(blk.Spans) == 0 {
		return nil
	}
	bt := newBlockText(blk, f.Body)
	masked := maskBareURLs(bt.text)
	var diags []Diagnostic

	emit := func(pos int, rule, msg string) {
		diags = append(diags, Diagnostic{
			Path: f.Path, Line: f.LineAt(bt.bodyOff(pos)), Rule: rule,
			Message: msg,
		})
	}

	// Repeated word.
	idx := wordSplit.FindAllStringIndex(masked, -1)
	for i := 1; i < len(idx); i++ {
		gap := masked[idx[i-1][1]:idx[i][0]]
		if !strings.ContainsAny(gap, " \t") {
			continue
		}
		if strings.ContainsAny(gap, ".!?,;:&(\"[])") {
			continue
		}
		a := strings.ToLower(masked[idx[i-1][0]:idx[i-1][1]])
		b := strings.ToLower(masked[idx[i][0]:idx[i][1]])
		if a != b {
			continue
		}
		if a == "had" || a == "that" {
			continue
		}
		emit(idx[i][0], "repeated-word", fmt.Sprintf("repeated word: %q", a+" "+a))
	}

	// Cross-span / URL-sensitive regex checks.
	if loc := spacedEmph.FindStringIndex(masked); loc != nil {
		emit(loc[0], "emphasis", "spaces inside emphasis markers (* text *)")
	}
	if strings.Contains(masked, "_") {
		// underscoreEmph requires whitespace before the opening marker;
		// prepend a space so a leading `_foo_` still matches, then offset
		// any reported position by -1.
		if loc := underscoreEmph.FindStringIndex(" " + masked); loc != nil {
			pos := loc[0] - 1
			if pos < 0 {
				pos = 0
			}
			emit(pos, "emphasis", "underscore emphasis (use * instead)")
		}
	}
	if loc := asymSlash.FindStringIndex(masked); loc != nil {
		if !strings.HasPrefix(masked[loc[0]:loc[1]], "w/") {
			emit(loc[0], "spacing", "asymmetrical spacing around /")
		}
	}
	if loc := asymHyphen.FindStringIndex(masked); loc != nil {
		emit(loc[0], "spacing", "asymmetrical spacing around hyphen")
	}
	if loc := hyphenMinus.FindStringIndex(masked); loc != nil {
		emit(loc[0], "number-format", "hyphen used as minus sign (use −)")
	}
	if loc := hyphenRange.FindStringIndex(masked); loc != nil {
		emit(loc[0], "number-format", "hyphen in numeric range (use en dash –)")
	}
	if strings.Contains(masked, " !") {
		emit(strings.Index(masked, " !"), "spacing", "space before ! mark")
	}
	return diags
}

// proseSpanChecks runs the per-span literal and short-context regex
// checks. These are content-level needles that don't need cross-span
// awareness and don't false-positive on URLs.
func proseSpanChecks(f *MarkdownFile, sp ProseSpan) []Diagnostic {
	text := string(sp.Text)
	if text == "" {
		return nil
	}
	var diags []Diagnostic

	emit := func(rule, msg string) {
		diags = append(diags, Diagnostic{
			Path: f.Path, Line: f.LineAt(sp.Offset), Rule: rule,
			Message: msg,
		})
	}

	for _, p := range astLiteralPatterns {
		if strings.Contains(text, p.needle) {
			emit(p.rule, fmt.Sprintf("%s: %q", p.msg, p.needle))
		}
	}

	if strings.Contains(text, " : ") && spacedColon.MatchString(text) {
		emit("spacing", "spaced colon ( : )")
	}
	if strings.ContainsAny(text, "—–") && mixedDashes.MatchString(text) {
		emit("typography", "malformed dash sequence (mixed em/en or 3+ dashes)")
	}
	if strings.Contains(text, "+-") || strings.Contains(text, "-+") {
		if plusMinus.MatchString(text) {
			emit("typography", "malformed plus-minus (use ±)")
		}
	}
	if strings.Contains(text, " %") && spacedPercent.MatchString(text) {
		emit("number-format", "space before percent sign (10 %)")
	}
	if strings.ContainsAny(text, "$£€¥") && spacedCurrency.MatchString(text) {
		emit("number-format", "space between currency symbol and number ($ 100)")
	}
	if strings.Contains(text, "#") && spacedHash.MatchString(text) {
		emit("number-format", "space after # before number (# 1, prefer #1)")
	}
	if strings.ContainsAny(text, "–-") && spacedDashNum.MatchString(text) {
		emit("number-format", "space between hyphen/en-dash and number ( – 10)")
	}
	if missingSpacePunct.MatchString(maskBareURLs(text)) {
		emit("spacing", "missing space after punctuation")
	}
	return diags
}

// proseQuoteChecks consolidates all "what's wrong with the quotes in
// this block" logic: spaces around quote, orphaned quote, padded quote, and the
// odd-count balance check that previously lived in markdown_balance.
// Concatenating across spans is required for the balance check to
// recognise pairs that span inline elements (e.g. `"see [foo](u)"`).
func proseQuoteChecks(f *MarkdownFile, blk ProseBlock) []Diagnostic {
	if len(blk.Spans) == 0 {
		return nil
	}
	var diags []Diagnostic

	var raw strings.Builder
	for _, sp := range blk.Spans {
		raw.Write(sp.Text)
	}
	rawText := raw.String()

	if strings.Contains(rawText, "\"") {
		if spacesAroundQuote.MatchString(rawText) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: f.LineAt(blk.Spans[0].Offset), Rule: "quotes",
				Message: `spaces around quote ( " )`,
			})
		}
		if orphanedQuote.MatchString(rawText) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: f.LineAt(blk.Spans[0].Offset), Rule: "quotes",
				Message: "orphaned quote",
			})
		}
		if paddedQuote.MatchString(rawText) {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: f.LineAt(blk.Spans[0].Offset), Rule: "quotes",
				Message: `padded spaces inside quotation marks (" word ")`,
			})
		}
	}

	quoteOpen := false
	quoteOffset := 0
	for _, sp := range blk.Spans {
		for i := 0; i < len(sp.Text); i++ {
			c := sp.Text[i]
			if c == '\\' {
				i++
				continue
			}
			if c == '"' {
				if quoteOpen {
					quoteOpen = false
				} else {
					quoteOpen = true
					quoteOffset = sp.Offset + i
				}
			}
		}
	}
	if quoteOpen {
		diags = append(diags, Diagnostic{
			Path: f.Path, Line: f.LineAt(quoteOffset), Rule: "quotes",
			Message: `unbalanced '"' (odd count)`,
		})
	}
	return diags
}
