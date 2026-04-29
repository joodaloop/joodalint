package rules

import "testing"

func TestHeadings_ATXH1(t *testing.T) {
	diags := markdownHeadings{}.Check(mdFile("# Title\nbody\n"), nil)
	if !containsMsg(diags, "h1 headings are not allowed") {
		t.Fatalf("want h1 diagnostic, got %v", messages(diags))
	}
	if len(diags) != 1 || diags[0].Line != 1 {
		t.Fatalf("want one diag on line 1, got %+v", diags)
	}
}

func TestHeadings_ATXMissingSpace(t *testing.T) {
	diags := markdownHeadings{}.Check(mdFile("#Title\n"), nil)
	if !containsMsg(diags, "no space after # in heading") {
		t.Fatalf("want missing-space diagnostic, got %v", messages(diags))
	}
	if len(diags) != 1 || diags[0].Line != 1 {
		t.Fatalf("want one diag on line 1, got %+v", diags)
	}
}

func TestHeadings_ATXMissingSpaceH2(t *testing.T) {
	diags := markdownHeadings{}.Check(mdFile("##Section\n"), nil)
	if !containsMsg(diags, "no space after # in heading") {
		t.Fatalf("want missing-space diagnostic, got %v", messages(diags))
	}
	if containsMsg(diags, "h1 headings are not allowed") {
		t.Fatalf("did not want h1 diagnostic, got %v", messages(diags))
	}
}

func TestHeadings_SetextH1(t *testing.T) {
	diags := markdownHeadings{}.Check(mdFile("Title\n=====\nbody\n"), nil)
	if !containsMsg(diags, "h1 headings are not allowed") {
		t.Fatalf("want setext h1 diagnostic, got %v", messages(diags))
	}
	if len(diags) != 1 || diags[0].Line != 1 {
		t.Fatalf("want one diag on heading text line 1, got %+v", diags)
	}
}

func TestHeadings_H2Allowed(t *testing.T) {
	src := "## Section\nTitle\n-----\n"
	diags := markdownHeadings{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestHeadings_FrontmatterSkipped(t *testing.T) {
	src := "---\ntitle: x\n---\nbody\n"
	diags := markdownHeadings{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestHeadings_FencedCodeSkipped(t *testing.T) {
	src := "```md\n# Not a heading\nTitle\n=====\n```\n"
	diags := markdownHeadings{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestHeadings_IndentedCodeSkipped(t *testing.T) {
	src := "    # Not a heading\n    ====\n"
	diags := markdownHeadings{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestHeadings_IndentedATX(t *testing.T) {
	diags := markdownHeadings{}.Check(mdFile("  ## Section\n"), nil)
	if !containsMsg(diags, "headings must start at the beginning of the line") {
		t.Fatalf("want indent diagnostic, got %v", messages(diags))
	}
	if len(diags) != 1 || diags[0].Line != 1 {
		t.Fatalf("want one diag on line 1, got %+v", diags)
	}
}

func TestHeadings_ID(t *testing.T) {
	if (markdownHeadings{}).ID() != "headings" {
		t.Fatal("wrong ID")
	}
}
