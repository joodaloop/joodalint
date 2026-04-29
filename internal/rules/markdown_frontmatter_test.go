package rules

import (
	"testing"

	"github.com/joodaloop/hugolint/internal/config"
)

func ctxWithSection(section string, schema map[string]config.FieldSpec) *MarkdownContext {
	return &MarkdownContext{Config: &config.Config{
		Paths:    config.Paths{MarkdownRoot: "content"},
		Sections: map[string]map[string]config.FieldSpec{section: schema},
	}}
}

func TestFrontmatter_NoConfigNoDiags(t *testing.T) {
	src := "---\ntitle: Hi\n---\nbody\n"
	diags := markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(src)}, nil)
	assertNoDiags(t, diags)
}

func TestFrontmatter_NoSchemaJustUnknownChecks(t *testing.T) {
	// Without a matching schema, existing fields are reported as unknown.
	cfg := &config.Config{Paths: config.Paths{MarkdownRoot: "content"}}
	ctx := &MarkdownContext{Config: cfg}
	src := "---\ntitle: Hi\nstray: yes\n---\nbody\n"
	diags := markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(src)}, ctx)
	if len(diags) != 2 {
		t.Fatalf("want 2 unknown-field diags, got %v", messages(diags))
	}
}

func TestFrontmatter_MissingFrontmatter(t *testing.T) {
	ctx := ctxWithSection("root", map[string]config.FieldSpec{
		"title": {Type: "string", Required: true},
	})
	src := "no fm here\n"
	diags := markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(src)}, ctx)
	if !containsMsg(diags, "missing YAML frontmatter") {
		t.Fatalf("want missing-frontmatter, got %v", messages(diags))
	}
}

func TestFrontmatter_MissingRequired(t *testing.T) {
	ctx := ctxWithSection("root", map[string]config.FieldSpec{
		"title": {Type: "string", Required: true},
	})
	src := "---\nother: x\n---\nbody\n"
	diags := markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(src)}, ctx)
	if !containsMsg(diags, `missing required field "title"`) {
		t.Fatalf("want missing-required, got %v", messages(diags))
	}
}

func TestFrontmatter_InvalidYAML(t *testing.T) {
	ctx := ctxWithSection("root", map[string]config.FieldSpec{
		"title": {Type: "string"},
	})
	src := "---\ntitle: : oops\n---\nbody\n"
	diags := markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(src)}, ctx)
	if !containsMsg(diags, "invalid YAML") {
		t.Fatalf("want invalid-YAML, got %v", messages(diags))
	}
}

func TestFrontmatter_StringMinMax(t *testing.T) {
	ctx := ctxWithSection("root", map[string]config.FieldSpec{
		"title": {Type: "string", Min: 5, Max: 10},
	})
	short := "---\ntitle: hi\n---\nbody\n"
	d := markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(short)}, ctx)
	if !containsMsg(d, "below min") {
		t.Errorf("want below-min, got %v", messages(d))
	}
	long := "---\ntitle: thisistoolongforus\n---\nbody\n"
	d = markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(long)}, ctx)
	if !containsMsg(d, "above max") {
		t.Errorf("want above-max, got %v", messages(d))
	}
	ok := "---\ntitle: hello\n---\nbody\n"
	d = markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(ok)}, ctx)
	if len(d) != 0 {
		t.Errorf("want no diags, got %v", messages(d))
	}
}

func TestFrontmatter_EnumAndList(t *testing.T) {
	ctx := ctxWithSection("root", map[string]config.FieldSpec{
		"type":   {Type: "enum", Values: []string{"post", "note"}},
		"topics": {Type: "list", Items: "enum", Values: []string{"go", "web"}},
	})
	bad := "---\ntype: foo\ntopics: [go, bogus]\n---\nbody\n"
	d := markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(bad)}, ctx)
	if !containsMsg(d, `"foo" not in allowed values`) {
		t.Errorf("want enum error, got %v", messages(d))
	}
	if !containsMsg(d, `"bogus" not in allowed values`) {
		t.Errorf("want list-item enum error, got %v", messages(d))
	}
}

func TestFrontmatter_DateValidation(t *testing.T) {
	ctx := ctxWithSection("root", map[string]config.FieldSpec{
		"date": {Type: "date"},
	})
	bad := "---\ndate: not-a-date\n---\nbody\n"
	d := markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(bad)}, ctx)
	if !containsMsg(d, "expected date") {
		t.Errorf("want date error, got %v", messages(d))
	}
	good := "---\ndate: 2024-01-15\n---\nbody\n"
	d = markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(good)}, ctx)
	if len(d) != 0 {
		t.Errorf("want no diags, got %v", messages(d))
	}
}

func TestFrontmatter_TypeMismatch(t *testing.T) {
	ctx := ctxWithSection("root", map[string]config.FieldSpec{
		"draft": {Type: "bool"},
		"count": {Type: "number"},
	})
	src := "---\ndraft: \"yes\"\ncount: \"x\"\n---\nbody\n"
	d := markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(src)}, ctx)
	if !containsMsg(d, `field "draft": expected bool`) {
		t.Errorf("want bool error, got %v", messages(d))
	}
	if !containsMsg(d, `field "count": expected number`) {
		t.Errorf("want number error, got %v", messages(d))
	}
}

func TestFrontmatter_UnknownFieldDeterministic(t *testing.T) {
	ctx := ctxWithSection("root", map[string]config.FieldSpec{
		"title": {Type: "string"},
	})
	src := "---\ntitle: hi\nzeta: 1\nalpha: 2\n---\nbody\n"
	d := markdownFrontmatter{}.Check(&MarkdownFile{Path: "content/x.md", Content: []byte(src)}, ctx)
	if len(d) != 2 || d[0].Message > d[1].Message {
		t.Fatalf("want 2 sorted unknown-field diags, got %v", messages(d))
	}
}

func TestExtractFrontmatter(t *testing.T) {
	body, line, ok := extractFrontmatter([]byte("---\ntitle: x\n---\nrest\n"))
	if !ok || line != 1 || string(body) != "title: x" {
		t.Errorf("got body=%q line=%d ok=%v", body, line, ok)
	}
	if _, _, ok := extractFrontmatter([]byte("no fm")); ok {
		t.Error("expected no frontmatter")
	}
}

func TestStripFrontmatter(t *testing.T) {
	got := stripFrontmatter([]byte("---\ntitle: x\n---\nbody\n"))
	if string(got) != "body\n" {
		t.Errorf("got %q", got)
	}
	got = stripFrontmatter([]byte("body only\n"))
	if string(got) != "body only\n" {
		t.Errorf("got %q", got)
	}
}

func TestFrontmatter_ID(t *testing.T) {
	if (markdownFrontmatter{}).ID() != "frontmatter" {
		t.Fatal("wrong ID")
	}
}
