package rules

import (
	"testing"
)

// TestProseHygiene_Cases covers the structural / line-shape half of
// prose-hygiene that has to stay line-based because it fires on
// constructs the AST cannot recognise (broken HRs, indented headings,
// malformed bullets, reference-link syntax, etc.). Content-prose cases
// live in markdown_prose_ast_test.go.
func TestProseHygiene_Cases(t *testing.T) {
	cases := []struct {
		in   string
		msg  string
		want bool
	}{
		// Reversed link.
		{"see (text)[https://example.com] here\n", "reversed link syntax", true},
		{"(foo)[bar]\n", "reversed link syntax", true},
		{"oops ()[]\n", "reversed link syntax", true},
		{"normal [text](url) link\n", "reversed link syntax", false},
		{"two adjacent (paren) [bracket] groups with space\n", "reversed link syntax", false},
		{"citation (see [1]) reference\n", "reversed link syntax", false},

		// Reference link — discouraged, flag [text][ref] and [text][].
		{"see [foo][bar] here\n", "Avoid using reference links", true},
		{"see [foo][] here\n", "Avoid using reference links", true},
		{"normal [text](https://example.com) inline link\n", "Avoid using reference links", false},
		{"a [text][ref] followed by [ref]: url\n", "Avoid using reference links", true},

		// Setext heading literal.
		{"Title\n=====\n", "Setext headers", true},
		{"not a === setext\n", "Setext headers", true},
		{"just plain prose\n", "Setext headers", false},

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

		// Broken HR, odd indent.
		{"--\n", "broken horizontal rule", true},
		{"  --\n", "broken horizontal rule", true},
		{"--  \n", "broken horizontal rule", true},
		{"---\n", "broken horizontal rule", false},
		{"text -- with dashes\n", "broken horizontal rule", false},
		{" - item\n", "odd indentation", true},
		{"   - item\n", "odd indentation", true},
		{" * item\n", "odd indentation", true},
		{"   + item\n", "odd indentation", true},
		{"- item\n", "odd indentation", false},
		{"  - item\n", "odd indentation", false},
		{"    - nested\n", "odd indentation", false},

		// Invisible characters (line-based).
		{"hello\u200Bworld\n", "zero-width space", true},
		{"foo\uFEFFbar\n", "byte-order mark", true},
		{"soft\u00ADhyphen\n", "soft hyphen", true},
		{"non\u00A0break\n", "non-breaking space", true},
		{"plain ascii text\nwith émojis café\n", "invisible character", false},

		// Structural literal needles.
		{"foo --- bar\n", "literal triple hyphen", true},
		{"text\n\n---\n\nmore\n", "literal triple hyphen", false},
		{"link [foo](//x.com)\n", "protocol-relative link", true},
		{` " ](url)` + "\n", "quote glued to link", true},

		// Straight quotes for primes.
		{`she is 5'9" tall` + "\n", "straight quotes for feet/inches", true},
		{`it's '90s music` + "\n", "straight quotes for feet/inches", false},
		{`don't quote "this"` + "\n", "straight quotes for feet/inches", false},

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

func TestProseHygiene_ID(t *testing.T) {
	if (markdownProseHygiene{}).ID() != "prose-hygiene" {
		t.Fatal("wrong ID")
	}
}
