package config

import "testing"

func TestSchemaFor_RootSection(t *testing.T) {
	c := &Config{
		Paths:    Paths{MarkdownRoot: "content"},
		Sections: map[string]map[string]FieldSpec{"root": {"title": {Type: "string"}}},
	}
	key, schema := c.SchemaFor("content/about.md")
	if key != "root" || schema == nil || schema["title"].Type != "string" {
		t.Fatalf("want root match, got key=%q schema=%v", key, schema)
	}
}

func TestSchemaFor_LongestPrefix(t *testing.T) {
	c := &Config{
		Paths: Paths{MarkdownRoot: "content"},
		Sections: map[string]map[string]FieldSpec{
			"writing":        {"a": {Type: "string"}},
			"writing/drafts": {"b": {Type: "string"}},
		},
	}
	key, schema := c.SchemaFor("content/writing/drafts/x.md")
	if key != "writing/drafts" || schema["b"].Type != "string" {
		t.Fatalf("want writing/drafts, got %q %v", key, schema)
	}
	key, _ = c.SchemaFor("content/writing/x.md")
	if key != "writing" {
		t.Fatalf("want writing, got %q", key)
	}
}

func TestSchemaFor_IndexPages(t *testing.T) {
	c := &Config{
		Paths:         Paths{MarkdownRoot: "content"},
		IndexSections: map[string]map[string]FieldSpec{"writing": {"title": {Type: "string"}}},
		Sections:      map[string]map[string]FieldSpec{"writing": {"author": {Type: "string"}}},
	}
	key, schema := c.SchemaFor("content/writing/_index.md")
	if key != "writing" || schema["title"].Type != "string" {
		t.Fatalf("want index schema, got %q %v", key, schema)
	}
	if _, ok := schema["author"]; ok {
		t.Fatal("index pages should not pull from Sections")
	}
}

func TestSchemaFor_NoMatch(t *testing.T) {
	c := &Config{
		Paths:    Paths{MarkdownRoot: "content"},
		Sections: map[string]map[string]FieldSpec{"writing": {"a": {Type: "string"}}},
	}
	key, schema := c.SchemaFor("content/other/x.md")
	if key != "" || schema != nil {
		t.Fatalf("want no match, got %q %v", key, schema)
	}
}

func TestSkipMarkdown(t *testing.T) {
	c := &Config{Paths: Paths{
		MarkdownRoot: "content",
		MarkdownSkip: []string{"drafts", "notes/*", "*.wip.md", "changelog.md"},
	}}
	cases := []struct {
		path string
		want bool
	}{
		{"content/drafts", true},            // bare folder name matches the dir
		{"content/drafts/x.md", true},       // ...and cascades to its subtree
		{"content/sub/drafts/y.md", true},   // bare name matches at any depth
		{"content/changelog.md", true},      // file name anywhere
		{"content/notes/foo.md", true},      // anchored notes/ subtree
		{"content/notes/nested/x.md", true}, // gitignore cascades into subdirs
		{"content/posts/idea.wip.md", true}, // suffix glob on base name
		{"content/about.md", false},
		{"content/other/notes-x.md", false}, // notes/ is anchored to root
	}
	for _, tc := range cases {
		if got := c.SkipMarkdown("content", tc.path); got != tc.want {
			t.Errorf("SkipMarkdown(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestSkipBuild(t *testing.T) {
	c := &Config{Paths: Paths{
		BuildRoot: "public",
		BuildSkip: []string{"vendor", "404.html"},
	}}
	if !c.SkipBuild("public", "public/vendor/lib.html") {
		t.Error("want skip for files under vendor/")
	}
	if !c.SkipBuild("public", "public/404.html") {
		t.Error("want skip for 404.html")
	}
	if c.SkipBuild("public", "public/index.html") {
		t.Error("want no skip for index.html")
	}
}

func TestSkipNoPatterns(t *testing.T) {
	c := &Config{Paths: Paths{MarkdownRoot: "content"}}
	if c.SkipMarkdown("content", "content/changelog.md") {
		t.Fatal("want no skip when markdown_skip is empty")
	}
}
