package rules

import "testing"

func TestHeadingsAST_ATXH1(t *testing.T) {
	diags := markdownHeadingsAST{}.Check(mdFile("# Title\nbody\n"), nil)
	if !containsMsg(diags, "h1 headings are not allowed") {
		t.Fatalf("want h1 diagnostic, got %v", messages(diags))
	}
	if len(diags) != 1 || diags[0].Line != 1 {
		t.Fatalf("want one diag on line 1, got %+v", diags)
	}
}

func TestHeadingsAST_SetextH1(t *testing.T) {
	diags := markdownHeadingsAST{}.Check(mdFile("Title\n=====\nbody\n"), nil)
	if !containsMsg(diags, "h1 headings are not allowed") {
		t.Fatalf("want setext h1 diagnostic, got %v", messages(diags))
	}
	if len(diags) != 1 || diags[0].Line != 1 {
		t.Fatalf("want one diag on heading text line 1, got %+v", diags)
	}
}

func TestHeadingsAST_H2Allowed(t *testing.T) {
	diags := markdownHeadingsAST{}.Check(mdFile("## Section\nTitle\n-----\n"), nil)
	assertNoDiags(t, diags)
}

func TestHeadingsAST_FrontmatterSkipped(t *testing.T) {
	// SplitFrontmatter strips the YAML; goldmark sees only "body\n",
	// so there's no heading in the AST regardless of frontmatter content.
	diags := markdownHeadingsAST{}.Check(mdFile("---\ntitle: x\n---\nbody\n"), nil)
	assertNoDiags(t, diags)
}

func TestHeadingsAST_FencedCodeSkipped(t *testing.T) {
	src := "```md\n# Not a heading\nTitle\n=====\n```\n"
	diags := markdownHeadingsAST{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestHeadingsAST_FrontmatterLineOffset(t *testing.T) {
	// Body H1 reports the original-file line, not the body-relative line.
	src := "---\ntitle: x\n---\n\n# Real H1\n"
	diags := markdownHeadingsAST{}.Check(mdFile(src), nil)
	if len(diags) != 1 {
		t.Fatalf("want one diag, got %+v", diags)
	}
	if diags[0].Line != 5 {
		t.Fatalf("want line 5 (original-file line of `# Real H1`), got %d", diags[0].Line)
	}
}

func TestHeadingsAST_DeepHeadingFlagged(t *testing.T) {
	diags := markdownHeadingsAST{}.Check(mdFile("##### Too deep\nbody\n"), nil)
	if !containsMsg(diags, "h5 heading too deep") {
		t.Fatalf("want h5 too-deep diagnostic, got %v", messages(diags))
	}
}

func TestHeadingsAST_H6Flagged(t *testing.T) {
	diags := markdownHeadingsAST{}.Check(mdFile("###### Way too deep\n"), nil)
	if !containsMsg(diags, "h6 heading too deep") {
		t.Fatalf("want h6 too-deep diagnostic, got %v", messages(diags))
	}
}

func TestHeadingsAST_H4Allowed(t *testing.T) {
	diags := markdownHeadingsAST{}.Check(mdFile("#### Section\nbody\n"), nil)
	assertNoDiags(t, diags)
}

func TestHeadingsAST_ID(t *testing.T) {
	if (markdownHeadingsAST{}).ID() != "headings" {
		t.Fatal("wrong ID")
	}
}
