package rules

import (
	"strings"
	"testing"
)

func TestProseHygiene_Cases(t *testing.T) {
	cases := []struct {
		in   string
		msg  string
		want bool
	}{
		// Repeated word.
		{"the the cat\n", "repeated word", true},
		{"The the cat\n", "repeated word", true},
		{"before [foo](/foo) middle `the the` after <em>x</em>\n", "repeated word", false},

		// Literal patterns.
		{"hello —— world\n", "double em dash", true},
		{"foo --- bar\n", "literal triple hyphen", true},
		{"it's '' a thing\n", "double apostrophe", true},
		{"`` two\n", "double backtick", true},
		{"foo.. bar\n", "double period", true},
		{"foo,, bar\n", "double comma", true},
		{"hi (there )\n", "space before closing paren", true},
		{"link [foo](//x.com)\n", "protocol-relative link", true},
		{` " ](url)` + "\n", "quote glued to link", true},
		{"text\n\n---\n\nmore\n", "literal triple hyphen", false},

		// Reversed link.
		{"see (text)[https://example.com] here\n", "reversed link syntax", true},
		{"(foo)[bar]\n", "reversed link syntax", true},
		{"oops ()[]\n", "reversed link syntax", true},
		{"normal [text](url) link\n", "reversed link syntax", false},
		{"two adjacent (paren) [bracket] groups with space\n", "reversed link syntax", false},
		{"citation (see [1]) reference\n", "reversed link syntax", false},

		// Spaced colon, plus-minus.
		{"note : here\n", "spaced colon", true},
		{"value +-3\n", "malformed plus-minus", true},
		{"value -+3\n", "malformed plus-minus", true},

		// Underscore emphasis.
		{"this is _emphasized_ text\n", "underscore emphasis", true},
		{"_leading_ word at line start\n", "underscore emphasis", true},
		{"some __strong__ text\n", "underscore emphasis", true},
		{"end with emphasis _here_\n", "underscore emphasis", true},
		{"with punct _foo_, more\n", "underscore emphasis", true},
		{"snake_case_var is fine\n", "underscore emphasis", false},
		{"a foo_bar_baz identifier\n", "underscore emphasis", false},
		{"link [x](https://example.com/some_path_here)\n", "underscore emphasis", false},
		{"inline `snake_case_thing` here\n", "underscore emphasis", false},
		{"<a href=\"/foo_bar\">x</a>\n", "underscore emphasis", false},
		{"plain prose with no emphasis\n", "underscore emphasis", false},

		// Bullet without space.
		{"-foo\n", "list bullet without space", true},
		{"+foo\n", "list bullet without space", true},
		{"  -bar\n", "list bullet without space", true},
		{"- foo\n", "list bullet without space", false},
		{"* foo\n", "list bullet without space", false},
		{"+ foo\n", "list bullet without space", false},
		{"---\n", "list bullet without space", false},
		{"*emphasis*\n", "list bullet without space", false},

		// Blockquote.
		{">quoted\n", "blockquote > without space", true},
		{"  >indented\n", "blockquote > without space", true},
		{"> quoted\n", "blockquote > without space", false},
		{">> nested\n", "blockquote > without space", false},
		{">\n", "blockquote > without space", false},

		// Spaced emphasis.
		{"this * text * here\n", "spaces inside emphasis markers", true},
		{"line ** bold ** end\n", "spaces inside emphasis markers", true},
		{"prefix * a b * suffix\n", "spaces inside emphasis markers", true},
		{"*foo* and * bad * here\n", "spaces inside emphasis markers", true},
		{"this *text* here\n", "spaces inside emphasis markers", false},
		{"strong **bold** here\n", "spaces inside emphasis markers", false},
		{"* * *\n", "spaces inside emphasis markers", false},
		{"* list item with *star*\n", "spaces inside emphasis markers", false},
		{"a 2 * 3 = 6 b\n", "spaces inside emphasis markers", false},

		// Headings.
		{" # Heading\n", "heading must start at the beginning", true},
		{"  ## Heading\n", "heading must start at the beginning", true},
		{"\t# Heading\n", "heading must start at the beginning", true},
		{"# Heading\n", "heading must start at the beginning", false},
		{"#Heading\n", "missing space after #", true},
		{"##Heading\n", "missing space after #", true},
		{"######Heading\n", "missing space after #", true},
		{"# Heading\n", "missing space after #", false},
		{"#\n", "missing space after #", false},
		{"#### \n", "missing space after #", false},

		// Broken HR, triple-star, odd indent.
		{"--\n", "broken horizontal rule", true},
		{"  --\n", "broken horizontal rule", true},
		{"--  \n", "broken horizontal rule", true},
		{"---\n", "broken horizontal rule", false},
		{"text -- with dashes\n", "broken horizontal rule", false},
		{"this ***word* is weird\n", "ambiguous triple-star", true},
		{"***foo* end\n", "ambiguous triple-star", true},
		{"this ***word*** is bold-italic\n", "ambiguous triple-star", false},
		{"a *foo* and **bar** mix\n", "ambiguous triple-star", false},
		{" - item\n", "odd indentation", true},
		{"   - item\n", "odd indentation", true},
		{" * item\n", "odd indentation", true},
		{"   + item\n", "odd indentation", true},
		{"- item\n", "odd indentation", false},
		{"  - item\n", "odd indentation", false},
		{"    - nested\n", "odd indentation", false},

		// Invisible characters.
		{"hello\u200Bworld\n", "zero-width space", true},
		{"foo\uFEFFbar\n", "byte-order mark", true},
		{"soft\u00ADhyphen\n", "soft hyphen", true},
		{"non\u00A0break\n", "non-breaking space", true},
		{"plain ascii text\nwith émojis café\n", "invisible character", false},

		// Mixed em/en dashes.
		{"foo —– bar\n", "malformed dash sequence", true},
		{"foo –— bar\n", "malformed dash sequence", true},
		{"foo ——– bar\n", "malformed dash sequence", true},
		{"foo ––– bar\n", "malformed dash sequence", true},
		{"single em — dash\n", "malformed dash sequence", false},
		{"single en – dash\n", "malformed dash sequence", false},

		// Floating quote.
		{"hello \" world\n", "floating/orphaned quote", true},
		{"orphan \"\n", "floating/orphaned quote", true},
		{"he said \"hello\" to me\n", "floating/orphaned quote", false},
		{"<a href=\"url\">link</a>\n", "floating/orphaned quote", false},

		// Missing space after punctuation.
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

		// Padded quotes.
		{`he said " hello " to me` + "\n", "padded spaces inside quotation marks", true},
		{`he said "hello" to me` + "\n", "padded spaces inside quotation marks", false},
		{`<a href="url">x</a>` + "\n", "padded spaces inside quotation marks", false},

		// Spaced percent.
		{"gain 10 % yearly\n", "space before percent", true},
		{"gain 10% yearly\n", "space before percent", false},

		// Spaced currency.
		{"costs $ 100 total\n", "space between currency symbol", true},
		{"costs £ 50\n", "space between currency symbol", true},
		{"costs $100 total\n", "space between currency symbol", false},

		// Spaced hash.
		{"see issue # 1 for details\n", "space after #", true},
		{"see issue #1 for details\n", "space after #", false},
		{"# 1\n", "space after #", false},
		{"## Heading\n", "space after #", false},
		{"### 1.2 Heading\n", "space after #", false},

		// Straight quotes for primes.
		{`she is 5'9" tall` + "\n", "straight quotes for feet/inches", true},
		{`it's '90s music` + "\n", "straight quotes for feet/inches", false},
		{`don't quote "this"` + "\n", "straight quotes for feet/inches", false},

		// Asymmetrical hyphen spacing.
		{"well- known method\n", "asymmetrical spacing around hyphen", true},
		{"well -known method\n", "asymmetrical spacing around hyphen", true},
		{"well-known method\n", "asymmetrical spacing around hyphen", false},
		{"well - known method\n", "asymmetrical spacing around hyphen", false},
		{"re-enter the room\n", "asymmetrical spacing around hyphen", false},

		// Hyphen as minus.
		{"temperature is -10 today\n", "hyphen used as minus", true},
		{"-10 below zero\n", "hyphen used as minus", true},
		{"pages 1-10 here\n", "hyphen used as minus", false},
		{"- list item\n", "hyphen used as minus", false},

		// Hyphen as numeric-range dash.
		{"pages 100-200 are blank\n", "hyphen in numeric range", true},
		{"the 1990-2000 era\n", "hyphen in numeric range", true},
		{"date 2020-01-01 here\n", "hyphen in numeric range", false},
		{"version 1.0-rc1 ships\n", "hyphen in numeric range", false},

		// Hugo shortcode spacing.
		{"{{<figure src=x>}}\n", "Hugo shortcode", true},
		{"{{< figure src=x>}}\n", "Hugo shortcode", true},
		{"{{<figure src=x >}}\n", "Hugo shortcode", true},
		{"{{%note%}}\n", "Hugo shortcode", true},
		{"{{< figure src=x >}}\n", "Hugo shortcode", false},
		{"{{% note %}}\n", "Hugo shortcode", false},
		{"{{< /figure >}}\n", "Hugo shortcode", false},
	}
	for _, tc := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(tc.in), nil)
		got := containsMsg(diags, tc.msg)
		if got != tc.want {
			t.Errorf("input %q: want contains(%q)=%v, got %v (diags=%v)", tc.in, tc.msg, tc.want, got, messages(diags))
		}
	}
}

func TestProseHygiene_StructuralSkips(t *testing.T) {
	cases := []struct {
		name, src string
	}{
		{"frontmatter", "---\ntitle: \"hello --- world\"\nthe the: bad\n---\n\nbody\n"},
		{"fenced code", "intro\n\n```go\nthe the\n---\n```\n\nafter\n"},
		{"style/script", "<style>\nthe the\n</style>\nbody\n<script>\nfoo foo\n</script>\n"},
	}
	for _, tc := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(tc.src), nil)
		if len(diags) != 0 {
			t.Errorf("%s: want no diags, got %v", tc.name, messages(diags))
		}
	}
}

func TestProseHygiene_LineNumbers(t *testing.T) {
	// Repeated words on lines 2 and 4.
	src := "line one\nthe the line two\nline three\nfoo foo line four\n"
	diags := markdownProseHygiene{}.Check(mdFile(src), nil)
	got := linesOf(diags)
	if len(got) != 2 || got[0] != 2 || got[1] != 4 {
		t.Fatalf("want lines [2 4], got %v (msgs %v)", got, messages(diags))
	}

	// Body line offset: frontmatter pushes the diag to line 5.
	diags = markdownProseHygiene{}.Check(mdFile("---\ntitle: hi\n---\n\nthe the cat\n"), nil)
	if len(diags) != 1 || diags[0].Line != 5 {
		t.Fatalf("want one diag at line 5, got %v at lines %v", messages(diags), linesOf(diags))
	}

	// Repeated-word message format.
	diags = markdownProseHygiene{}.Check(mdFile("the the cat\n"), nil)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, `"the the"`) {
		t.Fatalf("want one repeated-word diag, got %v", messages(diags))
	}
}

func TestProseHygiene_ID(t *testing.T) {
	if (markdownProseHygiene{}).ID() != "prose-hygiene" {
		t.Fatal("wrong ID")
	}
}
