package rules

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/joodaloop/hugolint/internal/config"
)

func spellingCfg(t *testing.T, words ...string) *config.Config {
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
		Sections: map[string]map[string]config.FieldSpec{},
	}
}

// resetSpeller clears sharedSpeller so each test re-inits with its own dict.
func resetSpeller() {
	sharedSpeller = &speller{}
}

func TestFrontmatterSpelling_NoDictNoDiags(t *testing.T) {
	resetSpeller()
	cfg := &config.Config{Paths: config.Paths{MarkdownRoot: "content"}}
	ctx := &FrontmatterContext{Config: cfg}
	src := "---\ntitle: zzqq\ndescription: hello\n---\nbody\n"
	d := (&frontmatterSpelling{}).Check(fmFile("content/x.md", src), ctx)
	assertNoDiags(t, d)
}

func TestFrontmatterSpelling_TitleAndDescription(t *testing.T) {
	resetSpeller()
	cfg := spellingCfg(t)
	ctx := &FrontmatterContext{Config: cfg}
	src := "---\ntitle: hello asdfqwer\ndescription: a fine zzqqxxyy line\n---\nbody\n"
	d := (&frontmatterSpelling{}).Check(fmFile("content/x.md", src), ctx)
	if !containsMsg(d, `unknown word in "title": "asdfqwer"`) {
		t.Errorf("want title diag, got %v", messages(d))
	}
	if !containsMsg(d, `unknown word in "description": "zzqqxxyy"`) {
		t.Errorf("want description diag, got %v", messages(d))
	}
}

func TestFrontmatterSpelling_TextTypeField(t *testing.T) {
	resetSpeller()
	cfg := spellingCfg(t)
	cfg.Sections["root"] = map[string]config.FieldSpec{
		"summary": {Type: "text"},
		"slug":    {Type: "string"},
	}
	ctx := &FrontmatterContext{Config: cfg}
	src := "---\ntitle: ok\ndescription: also fine\nsummary: contains floopynoodle\nslug: zzqqyyxx\n---\nbody\n"
	d := (&frontmatterSpelling{}).Check(fmFile("content/x.md", src), ctx)
	if !containsMsg(d, `unknown word in "summary": "floopynoodle"`) {
		t.Errorf("want summary diag, got %v", messages(d))
	}
	// slug is type "string", not "text" — should NOT be spellchecked.
	if containsMsg(d, `unknown word in "slug"`) {
		t.Errorf("string field should not be spellchecked, got %v", messages(d))
	}
}

func TestFrontmatterSpelling_PersonalDictAccepted(t *testing.T) {
	resetSpeller()
	cfg := spellingCfg(t, "floopynoodle")
	ctx := &FrontmatterContext{Config: cfg}
	src := "---\ntitle: floopynoodle\ndescription: ok line\n---\nbody\n"
	d := (&frontmatterSpelling{}).Check(fmFile("content/x.md", src), ctx)
	if containsMsg(d, "floopynoodle") {
		t.Errorf("personal-dict word should be accepted, got %v", messages(d))
	}
}

func TestFrontmatterSpelling_ParseErrSkipped(t *testing.T) {
	resetSpeller()
	cfg := spellingCfg(t)
	ctx := &FrontmatterContext{Config: cfg}
	src := "---\ntitle: : bad\n---\nbody\n"
	d := (&frontmatterSpelling{}).Check(fmFile("content/x.md", src), ctx)
	assertNoDiags(t, d)
}

func TestFrontmatterSpelling_ID(t *testing.T) {
	if (&frontmatterSpelling{}).ID() != "spelling" {
		t.Fatal("wrong ID")
	}
}

