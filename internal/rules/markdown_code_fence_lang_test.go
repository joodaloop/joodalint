package rules

import "testing"

func TestCodeFenceLang_MissingTag(t *testing.T) {
	src := "intro\n```\ncode\n```\n"
	diags := markdownCodeFenceLang{}.Check(mdFile(src), nil)
	if len(diags) != 1 || diags[0].Line != 2 {
		t.Fatalf("want one diag on line 2, got %+v", diags)
	}
}

func TestCodeFenceLang_PresentTag(t *testing.T) {
	src := "intro\n```go\nfmt.Println()\n```\n"
	diags := markdownCodeFenceLang{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestCodeFenceLang_TildeFence(t *testing.T) {
	src := "~~~\ncode\n~~~\n"
	diags := markdownCodeFenceLang{}.Check(mdFile(src), nil)
	if len(diags) != 1 {
		t.Fatalf("want one diag, got %+v", diags)
	}
}

func TestCodeFenceLang_ClosingFenceWithLangNotFlagged(t *testing.T) {
	// A closing fence is the same character but typically empty;
	// it shouldn't produce an extra diagnostic for itself.
	src := "```js\ncode\n```\n"
	diags := markdownCodeFenceLang{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestCodeFenceLang_NestedDifferentChar(t *testing.T) {
	// A `~~~` line inside a ``` block is just text, not a fence.
	src := "```\n~~~\ninside\n~~~\n```\n"
	diags := markdownCodeFenceLang{}.Check(mdFile(src), nil)
	if len(diags) != 1 || diags[0].Line != 1 {
		t.Fatalf("want one diag on line 1, got %+v", diags)
	}
}

func TestCodeFenceLang_IndentedFence(t *testing.T) {
	src := "  ```\n  code\n  ```\n"
	diags := markdownCodeFenceLang{}.Check(mdFile(src), nil)
	if len(diags) != 1 {
		t.Fatalf("want one diag, got %+v", diags)
	}
}

func TestCodeFenceLang_Unclosed(t *testing.T) {
	src := "intro\n```go\ncode\nmore code\n"
	diags := markdownCodeFenceLang{}.Check(mdFile(src), nil)
	if len(diags) != 1 || diags[0].Rule != "code-fence-unclosed" || diags[0].Line != 2 {
		t.Fatalf("want one code-fence-unclosed diag on line 2, got %+v", diags)
	}
}

func TestCodeFenceLang_UnclosedNoLang(t *testing.T) {
	// Both missing-lang and unclosed should fire.
	src := "```\ncode\n"
	diags := markdownCodeFenceLang{}.Check(mdFile(src), nil)
	if len(diags) != 2 {
		t.Fatalf("want two diags, got %+v", diags)
	}
	rules := map[string]int{}
	for _, d := range diags {
		rules[d.Rule]++
	}
	if rules["code-fence-lang"] != 1 || rules["code-fence-unclosed"] != 1 {
		t.Fatalf("want one of each rule, got %+v", rules)
	}
}

func TestCodeFenceLang_ClosedNotFlagged(t *testing.T) {
	src := "```go\ncode\n```\n"
	diags := markdownCodeFenceLang{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestCodeFenceLang_ID(t *testing.T) {
	if (markdownCodeFenceLang{}).ID() != "code-fence-lang" {
		t.Fatal("wrong ID")
	}
}
