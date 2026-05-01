package rules

import (
	"testing"

	"github.com/joodaloop/hugolint/internal/config"
)

func fmCtxWithSection(section string, schema map[string]config.FieldSpec) *FrontmatterContext {
	return &FrontmatterContext{Config: &config.Config{
		Paths:    config.Paths{MarkdownRoot: "content"},
		Sections: map[string]map[string]config.FieldSpec{section: schema},
	}}
}

func fmFile(path, src string) *FrontmatterFile {
	raw, _, _, line0 := SplitFrontmatter([]byte(src))
	return &FrontmatterFile{
		Path:   path,
		Raw:    raw,
		Parsed: ParseFrontmatterYAML(raw),
		Line0:  line0,
	}
}

func TestFrontmatter_NoConfigNoDiags(t *testing.T) {
	src := "---\ntitle: Hi\n---\nbody\n"
	diags := markdownFrontmatter{}.Check(fmFile("content/x.md", src), nil)
	assertNoDiags(t, diags)
}

func TestFrontmatter_NoSchemaJustUnknownChecks(t *testing.T) {
	// Without a matching schema, title/description are still required and
	// other fields are reported as unknown.
	cfg := &config.Config{Paths: config.Paths{MarkdownRoot: "content"}}
	ctx := &FrontmatterContext{Config: cfg}
	src := "---\ntitle: Hi\ndescription: A short description.\nstray: yes\n---\nbody\n"
	diags := markdownFrontmatter{}.Check(fmFile("content/x.md", src), ctx)
	if len(diags) != 1 || !containsMsg(diags, `unknown field "stray"`) {
		t.Fatalf("want 1 unknown-field diag for stray, got %v", messages(diags))
	}
}

func TestFrontmatter_MissingFrontmatter(t *testing.T) {
	ctx := fmCtxWithSection("root", map[string]config.FieldSpec{
		"title": {Type: "string", Required: true},
	})
	src := "no fm here\n"
	diags := markdownFrontmatter{}.Check(fmFile("content/x.md", src), ctx)
	if !containsMsg(diags, "missing YAML frontmatter") {
		t.Fatalf("want missing-frontmatter, got %v", messages(diags))
	}
}

func TestFrontmatter_MissingRequired(t *testing.T) {
	ctx := fmCtxWithSection("root", map[string]config.FieldSpec{
		"title": {Type: "string", Required: true},
	})
	src := "---\nother: x\n---\nbody\n"
	diags := markdownFrontmatter{}.Check(fmFile("content/x.md", src), ctx)
	if !containsMsg(diags, `missing required field "title"`) {
		t.Fatalf("want missing-required, got %v", messages(diags))
	}
}

func TestFrontmatter_InvalidYAML(t *testing.T) {
	ctx := fmCtxWithSection("root", map[string]config.FieldSpec{
		"title": {Type: "string"},
	})
	src := "---\ntitle: : oops\n---\nbody\n"
	diags := markdownFrontmatter{}.Check(fmFile("content/x.md", src), ctx)
	if !containsMsg(diags, "invalid YAML") {
		t.Fatalf("want invalid-YAML, got %v", messages(diags))
	}
}

func TestFrontmatter_StringMinMax(t *testing.T) {
	ctx := fmCtxWithSection("root", map[string]config.FieldSpec{
		"title": {Type: "string", Min: 5, Max: 10},
	})
	short := "---\ntitle: hi\ndescription: ok desc\n---\nbody\n"
	d := markdownFrontmatter{}.Check(fmFile("content/x.md", short), ctx)
	if !containsMsg(d, "below min") {
		t.Errorf("want below-min, got %v", messages(d))
	}
	long := "---\ntitle: thisistoolongforus\ndescription: ok desc\n---\nbody\n"
	d = markdownFrontmatter{}.Check(fmFile("content/x.md", long), ctx)
	if !containsMsg(d, "above max") {
		t.Errorf("want above-max, got %v", messages(d))
	}
	ok := "---\ntitle: hello\ndescription: ok desc\n---\nbody\n"
	d = markdownFrontmatter{}.Check(fmFile("content/x.md", ok), ctx)
	if len(d) != 0 {
		t.Errorf("want no diags, got %v", messages(d))
	}
}

func TestFrontmatter_EnumAndList(t *testing.T) {
	ctx := fmCtxWithSection("root", map[string]config.FieldSpec{
		"type":   {Type: "enum", Values: []string{"post", "note"}},
		"topics": {Type: "list", Items: "enum", Values: []string{"go", "web"}},
	})
	bad := "---\ntype: foo\ntopics: [go, bogus]\n---\nbody\n"
	d := markdownFrontmatter{}.Check(fmFile("content/x.md", bad), ctx)
	if !containsMsg(d, `"foo" not in allowed values`) {
		t.Errorf("want enum error, got %v", messages(d))
	}
	if !containsMsg(d, `"bogus" not in allowed values`) {
		t.Errorf("want list-item enum error, got %v", messages(d))
	}
}

func TestFrontmatter_DateValidation(t *testing.T) {
	ctx := fmCtxWithSection("root", map[string]config.FieldSpec{
		"date": {Type: "date"},
	})
	bad := "---\ndate: not-a-date\n---\nbody\n"
	d := markdownFrontmatter{}.Check(fmFile("content/x.md", bad), ctx)
	if !containsMsg(d, "expected date") {
		t.Errorf("want date error, got %v", messages(d))
	}
	good := "---\ntitle: t\ndescription: d\ndate: 2024-01-15\n---\nbody\n"
	d = markdownFrontmatter{}.Check(fmFile("content/x.md", good), ctx)
	if len(d) != 0 {
		t.Errorf("want no diags, got %v", messages(d))
	}
}

func TestFrontmatter_TypeMismatch(t *testing.T) {
	ctx := fmCtxWithSection("root", map[string]config.FieldSpec{
		"draft": {Type: "bool"},
		"count": {Type: "number"},
	})
	src := "---\ndraft: \"yes\"\ncount: \"x\"\n---\nbody\n"
	d := markdownFrontmatter{}.Check(fmFile("content/x.md", src), ctx)
	if !containsMsg(d, `field "draft": expected bool`) {
		t.Errorf("want bool error, got %v", messages(d))
	}
	if !containsMsg(d, `field "count": expected number`) {
		t.Errorf("want number error, got %v", messages(d))
	}
}

func TestFrontmatter_UnknownFieldDeterministic(t *testing.T) {
	ctx := fmCtxWithSection("root", map[string]config.FieldSpec{
		"title": {Type: "string"},
	})
	src := "---\ntitle: hi\ndescription: d\nzeta: 1\nalpha: 2\n---\nbody\n"
	d := markdownFrontmatter{}.Check(fmFile("content/x.md", src), ctx)
	if len(d) != 2 || d[0].Message > d[1].Message {
		t.Fatalf("want 2 sorted unknown-field diags, got %v", messages(d))
	}
}

func TestFrontmatter_AlwaysRequiresTitleAndDescription(t *testing.T) {
	cfg := &config.Config{Paths: config.Paths{MarkdownRoot: "content"}}
	ctx := &FrontmatterContext{Config: cfg}
	src := "---\nfoo: bar\n---\nbody\n"
	d := markdownFrontmatter{}.Check(fmFile("content/x.md", src), ctx)
	if !containsMsg(d, `missing required field "title"`) {
		t.Errorf("want missing title, got %v", messages(d))
	}
	if !containsMsg(d, `missing required field "description"`) {
		t.Errorf("want missing description, got %v", messages(d))
	}
}

func TestFrontmatter_DescriptionTooLong(t *testing.T) {
	cfg := &config.Config{Paths: config.Paths{MarkdownRoot: "content"}}
	ctx := &FrontmatterContext{Config: cfg}
	long := ""
	for i := 0; i < 161; i++ {
		long += "a"
	}
	src := "---\ntitle: hi\ndescription: " + long + "\n---\nbody\n"
	d := markdownFrontmatter{}.Check(fmFile("content/x.md", src), ctx)
	if !containsMsg(d, "above max 160") {
		t.Errorf("want description-too-long, got %v", messages(d))
	}
}

func TestFrontmatter_ID(t *testing.T) {
	if (markdownFrontmatter{}).ID() != "frontmatter" {
		t.Fatal("wrong ID")
	}
}
