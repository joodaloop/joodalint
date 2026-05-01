package rules

import (
	"strings"
	"testing"
)

// TestProseHygieneAST_Cases covers the prose-content half of the
// prose-hygiene rule that runs against ProseBlocks. These cases all
// concern text the AST has already isolated from inline syntax (link
// URLs, code spans, raw HTML), so the checks should not see false
// positives from those constructs.
func TestProseHygieneAST_Cases(t *testing.T) {
	cases := []struct {
		in   string
		msg  string
		want bool
	}{
		// Repeated word — now per-block, so spans either side of an
		// inline element ("the [the](u)") still trip the check.
		{"the the cat\n", "repeated word", true},
		{"The the cat\n", "repeated word", true},
		{"the [the](/foo) cat\n", "repeated word", true},
		{"before [foo](/foo) middle `the the` after <em>x</em>\n", "repeated word", false},

		// Literal patterns moved to AST.
		{"hello —— world\n", "double em dash", true},
		{"it's '' a thing\n", "double apostrophe", true},
		{"`` two\n", "double backtick", true},
		{"foo,, bar\n", "double comma", true},
		{"hi (there )\n", "space before closing paren", true},

		// Link URLs and code spans should NOT contribute literal-needle
		// false positives — the AST keeps them out of prose.
		{"see [link]( https://x ) here\n", "space after opening paren", false},

		// Remaining literal patterns — all should be flagged in plain prose.
		{"hello , world\n", "space before comma", true},
		{"no comma here\n", "space before comma", false},
		{"word . Next\n", "space around period", true},
		{"word. Next\n", "space around period", false},
		// The '!' check runs at block level (proseBlockChecks) because goldmark
		// splits text at '!' trigger-char boundaries in its inline parser.
		{"yikes ! Run\n", "space before exclamation mark", true},
		{"no exclamation here\n", "space before exclamation mark", false},
		{"what ? That\n", "space before question mark", true},
		{"what? That\n", "space before question mark", false},
		{"lingering ** here\n", "unescaped bold markers", true},
		{"lingering ~~ here\n", "unescaped strikethrough markers", true},
		{"lingering __ here\n", "unescaped emphasis markers", true},

		// Unescaped markers inside code spans — not flagged.
		{"code `**foo**` here\n", "unescaped bold markers", false},
		{"code `~~foo~~` here\n", "unescaped strikethrough markers", false},
		{"code `__foo__` here\n", "unescaped emphasis markers", false},

		// Spaced colon, plus-minus.
		{"note : here\n", "spaced colon", true},
		{"value +-3\n", "malformed plus-minus", true},
		{"value -+3\n", "malformed plus-minus", true},

		// Underscore emphasis (literal `_` surviving into prose text).
		{"snake_case_var is fine\n", "underscore emphasis", false},
		{"a foo_bar_baz identifier\n", "underscore emphasis", false},
		{"link [x](https://example.com/some_path_here)\n", "underscore emphasis", false},
		{"inline `snake_case_thing` here\n", "underscore emphasis", false},
		{"plain prose with no emphasis\n", "underscore emphasis", false},

		// Spaced emphasis.
		{"this * text * here\n", "spaces inside emphasis markers", true},
		{"line ** bold ** end\n", "spaces inside emphasis markers", true},
		{"prefix * a b * suffix\n", "spaces inside emphasis markers", true},
		{"*foo* and * bad * here\n", "spaces inside emphasis markers", true},
		{"this *text* here\n", "spaces inside emphasis markers", false},
		{"strong **bold** here\n", "spaces inside emphasis markers", false},
		{"a 2 * 3 = 6 b\n", "spaces inside emphasis markers", false},

		// Mixed em/en dashes.
		{"foo —– bar\n", "malformed dash sequence", true},
		{"foo –— bar\n", "malformed dash sequence", true},
		{"foo ——– bar\n", "malformed dash sequence", true},
		{"foo ––– bar\n", "malformed dash sequence", true},
		{"single em — dash\n", "malformed dash sequence", false},
		{"single en – dash\n", "malformed dash sequence", false},

		// Floating quote / padded quote.
		{"hello \" world\n", "floating/orphaned quote", true},
		{"orphan \"\n", "floating/orphaned quote", true},
		{"he said \"hello\" to me\n", "floating/orphaned quote", false},
		{"<a href=\"url\">link</a>\n", "floating/orphaned quote", false},
		{`he said " hello " to me` + "\n", "padded spaces inside quotation marks", true},
		{`he said "hello" to me` + "\n", "padded spaces inside quotation marks", false},
		{`<a href="url">x</a>` + "\n", "padded spaces inside quotation marks", false},
		{`replaced the "GUIDE" type with "ESSAY" instead.` + "\n", "padded spaces inside quotation marks", false},
		{`Replaced "Colophon" with "Changelog" in the heading` + "\n", "padded spaces inside quotation marks", false},

		// Quote balance (moved here from the balance rule).
		{`He said "hello` + "\n", "unbalanced", true},
		{`He said "hello world"` + "\n", "unbalanced", false},

		// Missing space after punctuation. Link URLs and code spans must
		// not contribute false positives.
		{"end.Then start.\n", "missing space after punctuation", true},
		{"oops,Now what\n", "missing space after punctuation", true},
		{"see foo.go file\n", "missing space after punctuation", false},
		{"version 1.2.3 here\n", "missing space after punctuation", false},
		{"the U.S.A. flag\n", "missing space after punctuation", false},
		{"see https://example.com/foo.Bar here\n", "missing space after punctuation", false},
		{"code `foo.Bar` here\n", "missing space after punctuation", false},

		// Asymmetrical slash spacing.
		{"cat/ dog\n", "asymmetrical spacing around /", true},
		{"cat /dog\n", "asymmetrical spacing around /", true},
		{"cat / dog\n", "asymmetrical spacing around /", false},
		{"cat/dog\n", "asymmetrical spacing around /", false},
		{"see https://example.com/path here\n", "asymmetrical spacing around /", false},

		// Spaced percent / currency / hash.
		{"gain 10 % yearly\n", "space before percent", true},
		{"gain 10% yearly\n", "space before percent", false},
		{"costs $ 100 total\n", "space between currency symbol", true},
		{"costs £ 50\n", "space between currency symbol", true},
		{"costs $100 total\n", "space between currency symbol", false},
		{"see issue # 1 for details\n", "space after #", true},
		{"see issue #1 for details\n", "space after #", false},

		// Asymmetrical hyphen spacing.
		{"well- known method\n", "asymmetrical spacing around hyphen", true},
		{"well -known method\n", "asymmetrical spacing around hyphen", true},
		{"well-known method\n", "asymmetrical spacing around hyphen", false},
		{"well - known method\n", "asymmetrical spacing around hyphen", false},
		{"re-enter the room\n", "asymmetrical spacing around hyphen", false},
		{"high quality\n", "asymmetrical spacing around hyphen", false},

		// Smart quotes in plain prose — should be flagged.
		{"hello \u201cworld\u201d here\n", "opening smart quote", true},
		{"hello \u201cworld\u201d here\n", "closing smart quote", true},

		// Smart quote inside link text (included in ProseBlocks) — flagged.
		{"see [\u201cfoo\u201d](https://example.com) link\n", "opening smart quote", true},

		// Smart quote inside code span (skipped by FlattenProse) — not flagged.
		{"see `\u201cfoo\u201d` here\n", "opening smart quote", false},
		{"see `\u201cfoo\u201d` here\n", "closing smart quote", false},

		// Hyphen as minus / range. Link URLs must not contribute.
		{"temperature is -10 today\n", "hyphen used as minus", true},
		{"-10 below zero\n", "hyphen used as minus", true},
		{"pages 1-10 here\n", "hyphen used as minus", false},
		{"see [foo](https://example.com/path-2024-01-thing) here\n", "hyphen used as minus", false},
		{"see https://example.com/foo-2024 here\n", "hyphen used as minus", false},
		{"pages 100-200 are blank\n", "hyphen in numeric range", true},
		{"the 1990-2000 era\n", "hyphen in numeric range", true},
		{"date 2020-01-01 here\n", "hyphen in numeric range", false},
		{"version 1.0-rc1 ships\n", "hyphen in numeric range", false},
		{"see [link](https://example.com/2020-2024/post) here\n", "hyphen in numeric range", false},
		{"see https://example.com/1990-2000/era here\n", "hyphen in numeric range", false},
		{"in `pages 100-200` code\n", "hyphen in numeric range", false},
	}
	for _, tc := range cases {
		diags := markdownProseHygieneAST{}.Check(mdFile(tc.in), nil)
		got := containsMsg(diags, tc.msg)
		if got != tc.want {
			t.Errorf("input %q: want contains(%q)=%v, got %v (diags=%v)", tc.in, tc.msg, tc.want, got, messages(diags))
		}
	}
}

func TestProseHygieneAST_StructuralSkips(t *testing.T) {
	cases := []struct {
		name, src string
	}{
		{"frontmatter", "---\ntitle: \"hello --- world\"\nthe the: bad\n---\n\nbody\n"},
		{"fenced code", "intro\n\n```go\nthe the\n---\n```\n\nafter\n"},
		{"style/script", "<style>\nthe the\n</style>\nbody\n<script>\nfoo foo\n</script>\n"},
	}
	for _, tc := range cases {
		diags := markdownProseHygieneAST{}.Check(mdFile(tc.src), nil)
		if len(diags) != 0 {
			t.Errorf("%s: want no diags, got %v", tc.name, messages(diags))
		}
	}
}

func TestProseHygieneAST_LineNumbers(t *testing.T) {
	// Repeated words on lines 2 and 4 inside a single paragraph: the
	// per-block concat still attributes each diag to the right source
	// line via the span byte offset.
	src := "line one\nthe the line two\nline three\nfoo foo line four\n"
	diags := markdownProseHygieneAST{}.Check(mdFile(src), nil)
	got := linesOf(diags)
	if len(got) != 2 || got[0] != 2 || got[1] != 4 {
		t.Fatalf("want lines [2 4], got %v (msgs %v)", got, messages(diags))
	}

	// Body line offset: frontmatter pushes the diag down.
	diags = markdownProseHygieneAST{}.Check(mdFile("---\ntitle: hi\n---\n\nthe the cat\n"), nil)
	if len(diags) != 1 || diags[0].Line != 5 {
		t.Fatalf("want one diag at line 5, got %v at lines %v", messages(diags), linesOf(diags))
	}

	// Repeated-word message format.
	diags = markdownProseHygieneAST{}.Check(mdFile("the the cat\n"), nil)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, `"the the"`) {
		t.Fatalf("want one repeated-word diag, got %v", messages(diags))
	}
}

func TestProseHygieneAST_ID(t *testing.T) {
	if (markdownProseHygieneAST{}).ID() != "prose-hygiene" {
		t.Fatal("wrong ID")
	}
}
