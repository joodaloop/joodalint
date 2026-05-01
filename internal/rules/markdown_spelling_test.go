package rules

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/joodaloop/hugolint/internal/config"
)

func bodySpellingCfg(t *testing.T, words ...string) *config.Config {
	t.Helper()
	if _, err := exec.LookPath("aspell"); err != nil {
		t.Skip("aspell not installed")
	}
	dir := t.TempDir()
	dict := filepath.Join(dir, "dict.txt")
	body := ""
	for _, w := range words {
		body += w + "\n"
	}
	if err := os.WriteFile(dict, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return &config.Config{
		Paths:    config.Paths{MarkdownRoot: "content"},
		Spelling: config.Spelling{Dict: dict},
	}
}

func TestBodySpelling_NoDictNoDiags(t *testing.T) {
	resetSpeller()
	cfg := &config.Config{Paths: config.Paths{MarkdownRoot: "content"}}
	ctx := &MarkdownContext{Config: cfg}
	diags := (&markdownSpelling{}).Check(mdFile("zzqqyy asdfqwer\n"), ctx)
	assertNoDiags(t, diags)
}

func TestBodySpelling_UnknownWords(t *testing.T) {
	resetSpeller()
	cfg := bodySpellingCfg(t)
	ctx := &MarkdownContext{Config: cfg}
	diags := (&markdownSpelling{}).Check(mdFile("The asdfqwer is zzqqyy here.\n"), ctx)
	if !containsMsg(diags, `unknown word: "asdfqwer"`) {
		t.Errorf("want asdfqwer diag, got %v", messages(diags))
	}
	if !containsMsg(diags, `unknown word: "zzqqyy"`) {
		t.Errorf("want zzqqyy diag, got %v", messages(diags))
	}
}

func TestBodySpelling_PersonalDictAccepted(t *testing.T) {
	resetSpeller()
	cfg := bodySpellingCfg(t, "asdfqwer")
	ctx := &MarkdownContext{Config: cfg}
	diags := (&markdownSpelling{}).Check(mdFile("The asdfqwer is zzqqyy here.\n"), ctx)
	if containsMsg(diags, "asdfqwer") {
		t.Errorf("personal-dict word should be accepted, got %v", messages(diags))
	}
	if !containsMsg(diags, `unknown word: "zzqqyy"`) {
		t.Errorf("want zzqqyy diag, got %v", messages(diags))
	}
}

func TestBodySpelling_FencedCodeSkipped(t *testing.T) {
	resetSpeller()
	cfg := bodySpellingCfg(t)
	ctx := &MarkdownContext{Config: cfg}
	src := "intro\n\n```go\nasdfqwer\n```\n\nafter\n"
	diags := (&markdownSpelling{}).Check(mdFile(src), ctx)
	if containsMsg(diags, "asdfqwer") {
		t.Errorf("fenced code word should be skipped, got %v", messages(diags))
	}
}

func TestBodySpelling_InlineCodeSkipped(t *testing.T) {
	resetSpeller()
	cfg := bodySpellingCfg(t)
	ctx := &MarkdownContext{Config: cfg}
	diags := (&markdownSpelling{}).Check(mdFile("see `asdfqwer` here.\n"), ctx)
	if containsMsg(diags, "asdfqwer") {
		t.Errorf("inline code word should be skipped, got %v", messages(diags))
	}
}

func TestBodySpelling_FrontmatterSkipped(t *testing.T) {
	resetSpeller()
	cfg := bodySpellingCfg(t)
	ctx := &MarkdownContext{Config: cfg}
	src := "---\ntitle: asdfqwer\n---\n\nbody here.\n"
	diags := (&markdownSpelling{}).Check(mdFile(src), ctx)
	if containsMsg(diags, "asdfqwer") {
		t.Errorf("frontmatter word should be skipped, got %v", messages(diags))
	}
}

func TestBodySpelling_UnknownWordAfterManySpans(t *testing.T) {
	resetSpeller()
	cfg := bodySpellingCfg(t)
	ctx := &MarkdownContext{Config: cfg}
	src := "First paragraph with ordinary words.\n\n" +
		"## Heading\n\n" +
		"Second paragraph, also ordinary, spread across **bold** and *italic* spans.\n\n" +
		"Third paragraph contains the typo asdfqwer near the end.\n"
	diags := (&markdownSpelling{}).Check(mdFile(src), ctx)
	if !containsMsg(diags, `unknown word: "asdfqwer"`) {
		t.Errorf("want asdfqwer diag from late-in-file span, got %v", messages(diags))
	}
}

func TestBodySpelling_ID(t *testing.T) {
	if (&markdownSpelling{}).ID() != "spelling" {
		t.Fatal("wrong ID")
	}
}