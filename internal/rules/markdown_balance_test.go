package rules

import "testing"

func TestBalance_Balanced(t *testing.T) {
	src := `(simple) [also] {curly} "quoted"` + "\n"
	diags := markdownBalance{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestBalance_UnclosedParen(t *testing.T) {
	diags := markdownBalance{}.Check(mdFile("oops (open\n"), nil)
	if !containsMsg(diags, `unclosed '('`) {
		t.Fatalf("want unclosed '(', got %v", messages(diags))
	}
}

func TestBalance_UnmatchedClose(t *testing.T) {
	diags := markdownBalance{}.Check(mdFile("oops )\n"), nil)
	if !containsMsg(diags, `unmatched closing ')'`) {
		t.Fatalf("want unmatched ')', got %v", messages(diags))
	}
}

func TestBalance_Mismatch(t *testing.T) {
	diags := markdownBalance{}.Check(mdFile("(foo]\n"), nil)
	if !containsMsg(diags, "mismatched") {
		t.Fatalf("want mismatched diag, got %v", messages(diags))
	}
}

func TestBalance_OddQuotes_NowOwnedByProseHygiene(t *testing.T) {
	// Quote balance moved to prose-hygiene; the balance rule should no
	// longer fire on odd quote counts.
	src := `He said "hello` + "\n"
	balDiags := markdownBalance{}.Check(mdFile(src), nil)
	if len(balDiags) != 0 {
		t.Fatalf("balance rule should no longer flag quotes, got %v", messages(balDiags))
	}
	diags := markdownProseHygieneAST{}.Check(mdFile(src), nil)
	if !containsMsg(diags, "unbalanced") {
		t.Fatalf("want unbalanced quotes from prose-hygiene, got %v", messages(diags))
	}
}

func TestBalance_EvenQuotes(t *testing.T) {
	src := `He said "hello world"` + "\n"
	assertNoDiags(t, markdownBalance{}.Check(mdFile(src), nil))
	if containsMsg(markdownProseHygieneAST{}.Check(mdFile(src), nil), "unbalanced") {
		t.Fatal("even quotes should not trigger unbalanced diag")
	}
}

func TestBalance_BackslashEscape(t *testing.T) {
	// Backslash skips the next char, so the escaped paren shouldn't open the stack.
	diags := markdownBalance{}.Check(mdFile(`foo \( bar`+"\n"), nil)
	assertNoDiags(t, diags)
}

func TestBalance_FrontmatterStripped(t *testing.T) {
	src := "---\ntitle: \"unclosed in frontmatter\nsubtitle: ok\n---\n\n(balanced)\n"
	diags := markdownBalance{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestBalance_LineNumberOfOpener(t *testing.T) {
	src := "line one\nline two (\nline three\n"
	diags := markdownBalance{}.Check(mdFile(src), nil)
	if len(diags) != 1 || diags[0].Line != 2 {
		t.Fatalf("want one diag on line 2, got %+v", diags)
	}
}

func TestBalance_NestedAndCurly(t *testing.T) {
	src := "({[ok]})\n"
	diags := markdownBalance{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestBalance_FenceIgnored(t *testing.T) {
	src := "before\n```\n( unclosed\n```\nafter\n"
	diags := markdownBalance{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestBalance_CodeSpanIgnored(t *testing.T) {
	diags := markdownBalance{}.Check(mdFile("see `( unclosed` here\n"), nil)
	assertNoDiags(t, diags)
}

func TestBalance_ID(t *testing.T) {
	if (markdownBalance{}).ID() != "balance" {
		t.Fatal("wrong ID")
	}
}
