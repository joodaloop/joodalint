package rules

import (
	"sort"
	"strings"
	"testing"
)

// proseTags is the set of rule tags the prose checks may emit. It is the
// user-facing vocabulary for disabling checks, so adding a name here is a
// deliberate act: renaming or removing one breaks existing configs.
var proseTags = map[string]bool{
	"invisible-chars": true,
	"md-syntax":       true,
	"emphasis":        true,
	"link-style":      true,
	"shortcode":       true,
	"repeated-word":   true,
	"spacing":         true,
	"quotes":          true,
	"typography":      true,
	"number-format":   true,
}

// proseRuleIDs are the rules whose diagnostics must carry a prose tag.
var proseRuleIDs = map[string]bool{"prose-hygiene": true}

// proseTagSample exercises a wide spread of prose checks in one document
// so the tag assertions below see real diagnostics rather than a curated
// handful.
const proseTagSample = "---\ntitle: x\n---\n\n" +
	"Zero​width here.\n" +
	"\n" +
	"-nospace after bullet\n" +
	">no space after blockquote\n" +
	"#nospace after hash\n" +
	"--\n" +
	"\n" +
	"See [a][ref] and (text)[url] here.\n" +
	"\n" +
	"{{<shortcode>}}\n" +
	"\n" +
	"The the repeated word.\n" +
	"\n" +
	// `_emph _` has a space before the closing marker, so goldmark leaves
	// the underscores in the text rather than parsing them as emphasis —
	// which is precisely the case the underscore check exists to catch.
	"Spaced * emphasis * and _emph _ here and **bold**.\n" +
	"\n" +
	"Space , before comma and ( padded parens ).\n" +
	"\n" +
	"An “opening quote here.\n" +
	"\n" +
	// A trailing quote at end of line is what orphanedQuote matches.
	"He said \"\n" +
	"\n" +
	"Dashes —– mixed and 3'4\" primes.\n" +
	"\n" +
	"Costs $ 100 and 10 % and items 5-10.\n"

func proseDiagnostics(t *testing.T) []Diagnostic {
	t.Helper()
	f := mdFile(proseTagSample)
	ctx := &MarkdownContext{}

	var diags []Diagnostic
	for _, r := range Markdown() {
		if proseRuleIDs[r.ID()] {
			diags = append(diags, r.Check(f, ctx)...)
		}
	}
	for _, r := range MarkdownAST() {
		if proseRuleIDs[r.ID()] {
			diags = append(diags, r.Check(f, ctx)...)
		}
	}
	return diags
}

// TestProseTagsAreKnown is the guard that matters: no prose diagnostic may
// escape with a tag outside the documented set, and none may still carry
// the old catch-all "prose-hygiene" tag.
func TestProseTagsAreKnown(t *testing.T) {
	diags := proseDiagnostics(t)
	if len(diags) == 0 {
		t.Fatal("sample produced no diagnostics — it no longer exercises the checks")
	}
	for _, d := range diags {
		if d.Rule == "prose-hygiene" {
			t.Errorf("diagnostic still uses the catch-all tag: %q", d.Message)
			continue
		}
		if !proseTags[d.Rule] {
			t.Errorf("unknown rule tag %q on %q", d.Rule, d.Message)
		}
	}
}

// TestProseTagsCoverage checks the split actually partitions the checks
// rather than collapsing most of them into one bucket. Every tag the
// sample is written to trigger should appear.
func TestProseTagsCoverage(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range proseDiagnostics(t) {
		seen[d.Rule] = true
	}

	want := []string{
		"invisible-chars", "md-syntax", "emphasis", "link-style",
		"shortcode", "repeated-word", "spacing", "quotes",
		"typography", "number-format",
	}
	var missing []string
	for _, tag := range want {
		if !seen[tag] {
			missing = append(missing, tag)
		}
	}
	if len(missing) > 0 {
		var got []string
		for tag := range seen {
			got = append(got, tag)
		}
		sort.Strings(got)
		t.Errorf("tags never emitted: %v\n  emitted: %v", missing, got)
	}
}

// TestProseTagsAreStable pins specific messages to their tags. These are
// the assignments a reader would most plausibly get wrong when adding a
// check, so drift here should be deliberate.
func TestProseTagsAreStable(t *testing.T) {
	want := map[string]string{
		"invisible character":     "invisible-chars",
		"list bullet without":     "md-syntax",
		"blockquote > without":    "md-syntax",
		"reference links":         "link-style",
		"reversed link syntax":    "link-style",
		"Hugo shortcode":          "shortcode",
		"repeated word":           "repeated-word",
		"space before comma":      "spacing",
		"underscore emphasis":     "emphasis",
		"orphaned quote":          "quotes",
		"straight quotes":         "quotes",
		"malformed dash sequence": "typography",
		"currency symbol":         "number-format",
		"percent sign":            "number-format",
	}

	diags := proseDiagnostics(t)
	for substr, tag := range want {
		found := false
		for _, d := range diags {
			if strings.Contains(d.Message, substr) {
				found = true
				if d.Rule != tag {
					t.Errorf("%q tagged %q, want %q", d.Message, d.Rule, tag)
				}
			}
		}
		if !found {
			t.Errorf("sample never triggered a check matching %q", substr)
		}
	}
}
