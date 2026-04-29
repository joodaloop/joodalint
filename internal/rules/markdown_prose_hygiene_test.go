package rules

import (
	"strings"
	"testing"
)

func TestProseHygiene_RepeatedWord(t *testing.T) {
	diags := markdownProseHygiene{}.Check(mdFile("the the cat\n"), nil)
	if len(diags) != 1 || !strings.Contains(diags[0].Message, `"the the"`) {
		t.Fatalf("want one repeated-word diag, got %v", messages(diags))
	}
	if diags[0].Line != 1 {
		t.Fatalf("want line 1, got %d", diags[0].Line)
	}
}

func TestProseHygiene_RepeatedWordCaseInsensitive(t *testing.T) {
	diags := markdownProseHygiene{}.Check(mdFile("The the cat\n"), nil)
	if len(diags) != 1 {
		t.Fatalf("want one diag, got %v", messages(diags))
	}
}

func TestProseHygiene_LiteralPatterns(t *testing.T) {
	cases := []struct {
		in, contains string
	}{
		{"hello —— world\n", "double em dash"},
		{"foo --- bar\n", "literal triple hyphen"},
		{"it's '' a thing\n", "double apostrophe"},
		{"`` two\n", "double backtick"},
		{"hi (there )\n", "space before closing paren"},
		{"empty []()\n", "empty link"},
		{"oops ()[]\n", "reversed link syntax"},
		{"empty ![]()\n", "empty image"},
		{"link [foo](//x.com)\n", "protocol-relative link"},
		{` " ](url)` + "\n", "quote glued to link"},
	}
	for _, tc := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(tc.in), nil)
		if !containsMsg(diags, tc.contains) {
			t.Errorf("input %q: missing %q in %v", tc.in, tc.contains, messages(diags))
		}
	}
}

func TestProseHygiene_HRLineDoesNotTrigger(t *testing.T) {
	diags := markdownProseHygiene{}.Check(mdFile("text\n\n---\n\nmore\n"), nil)
	if containsMsg(diags, "literal triple hyphen") {
		t.Fatalf("HR line should not trigger triple-hyphen warning: %v", messages(diags))
	}
}

func TestProseHygiene_FrontmatterSkipped(t *testing.T) {
	src := "---\ntitle: \"hello --- world\"\nthe the: bad\n---\n\nbody\n"
	diags := markdownProseHygiene{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestProseHygiene_FencedCodeSkipped(t *testing.T) {
	src := "intro\n\n```go\nthe the\n---\n```\n\nafter\n"
	diags := markdownProseHygiene{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestProseHygiene_StyleScriptBlocksSkipped(t *testing.T) {
	src := "<style>\nthe the\n</style>\nbody\n<script>\nfoo foo\n</script>\n"
	diags := markdownProseHygiene{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestProseHygiene_LinksAndCodeNotTokenizedAsRepeats(t *testing.T) {
	src := "before [foo](/foo) middle `the the` after <em>x</em>\n"
	diags := markdownProseHygiene{}.Check(mdFile(src), nil)
	if containsMsg(diags, "repeated word") {
		t.Fatalf("inline code/links should not produce repeats: %v", messages(diags))
	}
}

func TestProseHygiene_SpacedColon(t *testing.T) {
	diags := markdownProseHygiene{}.Check(mdFile("note : here\n"), nil)
	if !containsMsg(diags, "spaced colon") {
		t.Fatalf("expected spaced colon: %v", messages(diags))
	}
}

func TestProseHygiene_PlusMinus(t *testing.T) {
	for _, in := range []string{"value +-3\n", "value -+3\n"} {
		diags := markdownProseHygiene{}.Check(mdFile(in), nil)
		if !containsMsg(diags, "malformed plus-minus") {
			t.Errorf("input %q: expected plus-minus diagnostic, got %v", in, messages(diags))
		}
	}
}

func TestProseHygiene_LineNumbersAccurate(t *testing.T) {
	src := "line one\nthe the line two\nline three\nfoo foo line four\n"
	diags := markdownProseHygiene{}.Check(mdFile(src), nil)
	got := linesOf(diags)
	want := []int{2, 4}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("want lines %v, got %v (msgs %v)", want, got, messages(diags))
	}
}

func TestProseHygiene_ReversedLinkFlagged(t *testing.T) {
	cases := []string{
		"see (text)[https://example.com] here\n",
		"(foo)[bar]\n",
		"oops ()[]\n",
	}
	for _, in := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(in), nil)
		if !containsMsg(diags, "reversed link syntax") {
			t.Errorf("input %q: expected reversed-link diag, got %v", in, messages(diags))
		}
	}
}

func TestProseHygiene_ReversedLinkIgnoresValid(t *testing.T) {
	cases := []string{
		"normal [text](url) link\n",
		"two adjacent (paren) [bracket] groups with space\n",
		"citation (see [1]) reference\n",
	}
	for _, in := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(in), nil)
		if containsMsg(diags, "reversed link syntax") {
			t.Errorf("input %q: should not flag, got %v", in, messages(diags))
		}
	}
}

func TestProseHygiene_UnderscoreEmphasisFlagged(t *testing.T) {
	cases := []string{
		"this is _emphasized_ text\n",
		"_leading_ word at line start\n",
		"some __strong__ text\n",
		"end with emphasis _here_\n",
		"with punct _foo_, more\n",
	}
	for _, in := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(in), nil)
		if !containsMsg(diags, "underscore emphasis") {
			t.Errorf("input %q: expected underscore-emphasis diag, got %v", in, messages(diags))
		}
	}
}

func TestProseHygiene_UnderscoreEmphasisIgnoresNonProse(t *testing.T) {
	cases := []string{
		"snake_case_var is fine\n",
		"a foo_bar_baz identifier\n",
		"link [x](https://example.com/some_path_here)\n",
		"inline `snake_case_thing` here\n",
		"<a href=\"/foo_bar\">x</a>\n",
		"plain prose with no emphasis\n",
	}
	for _, in := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(in), nil)
		if containsMsg(diags, "underscore emphasis") {
			t.Errorf("input %q: should not flag, got %v", in, messages(diags))
		}
	}
}

func TestProseHygiene_BulletNoSpaceFlagged(t *testing.T) {
	cases := []string{
		"-foo\n",
		"+foo\n",
		"  -bar\n",
	}
	for _, in := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(in), nil)
		if !containsMsg(diags, "list bullet without space") {
			t.Errorf("input %q: expected bullet diag, got %v", in, messages(diags))
		}
	}
}

func TestProseHygiene_BulletNoSpaceIgnoresNonBullet(t *testing.T) {
	cases := []string{
		"- foo\n",
		"* foo\n",
		"+ foo\n",
		"---\n",
		"*emphasis*\n",
		"plain prose\n",
	}
	for _, in := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(in), nil)
		if containsMsg(diags, "list bullet without space") {
			t.Errorf("input %q: should not flag, got %v", in, messages(diags))
		}
	}
}

func TestProseHygiene_BlockquoteNoSpaceFlagged(t *testing.T) {
	cases := []string{
		">quoted\n",
		"  >indented\n",
	}
	for _, in := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(in), nil)
		if !containsMsg(diags, "blockquote > without space") {
			t.Errorf("input %q: expected blockquote diag, got %v", in, messages(diags))
		}
	}
}

func TestProseHygiene_BlockquoteNoSpaceIgnoresValid(t *testing.T) {
	cases := []string{
		"> quoted\n",
		">> nested\n",
		">\n",
		"plain prose\n",
	}
	for _, in := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(in), nil)
		if containsMsg(diags, "blockquote > without space") {
			t.Errorf("input %q: should not flag, got %v", in, messages(diags))
		}
	}
}

func TestProseHygiene_SpacedEmphasisFlagged(t *testing.T) {
	cases := []string{
		"this * text * here\n",
		"line ** bold ** end\n",
		"prefix * a b * suffix\n",
	}
	for _, in := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(in), nil)
		if !containsMsg(diags, "spaces inside emphasis markers") {
			t.Errorf("input %q: expected spaced-emphasis diag, got %v", in, messages(diags))
		}
	}
}

func TestProseHygiene_SpacedEmphasisAfterValidEmphasis(t *testing.T) {
	diags := markdownProseHygiene{}.Check(mdFile("*foo* and * bad * here\n"), nil)
	if !containsMsg(diags, "spaces inside emphasis markers") {
		t.Fatalf("want spaced-emphasis diag despite leading *foo*, got %v", messages(diags))
	}
}

func TestProseHygiene_SpacedEmphasisIgnoresValid(t *testing.T) {
	cases := []string{
		"this *text* here\n",
		"strong **bold** here\n",
		"* * *\n",
		"* list item with *star*\n",
		"plain prose, no emphasis\n",
		"a 2 * 3 = 6 b\n",
	}
	for _, in := range cases {
		diags := markdownProseHygiene{}.Check(mdFile(in), nil)
		if containsMsg(diags, "spaces inside emphasis markers") {
			t.Errorf("input %q: should not flag, got %v", in, messages(diags))
		}
	}
}

func TestProseHygiene_ID(t *testing.T) {
	if (markdownProseHygiene{}).ID() != "prose-hygiene" {
		t.Fatal("wrong ID")
	}
}
