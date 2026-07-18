package rules

import (
	"testing"
)

func TestCollectJSFiles_OnlyJS(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/foo.js", "/foo.js", 100),
		builtFile("/site/public/bar.css", "/bar.css", 200),
		builtFile("/site/public/baz.html", "/baz.html", 300),
	}
	ctx := &HTMLContext{LinkedPages: map[string]bool{"/foo.js": true}}

	got := collectJSFiles(files, ctx)
	if len(got) != 1 {
		t.Fatalf("want 1 JS file, got %d", len(got))
	}
	if got[0].urlPath != "/foo.js" {
		t.Errorf("urlPath = %q, want /foo.js", got[0].urlPath)
	}
}

func TestCollectJSFiles_ModuleExtensionsIncluded(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/mod.mjs", "/mod.mjs", 100),
		builtFile("/site/public/legacy.cjs", "/legacy.cjs", 200),
	}
	ctx := &HTMLContext{LinkedPages: map[string]bool{
		"/mod.mjs": true, "/legacy.cjs": true,
	}}

	got := collectJSFiles(files, ctx)
	if len(got) != 2 {
		t.Fatalf("want 2 JS files (.mjs/.cjs), got %d", len(got))
	}
}

func TestCollectJSFiles_OrphanExcluded(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/linked.js", "/linked.js", 100),
		builtFile("/site/public/orphan.js", "/orphan.js", 200),
	}
	ctx := &HTMLContext{LinkedPages: map[string]bool{"/linked.js": true}}

	got := collectJSFiles(files, ctx)
	if len(got) != 1 {
		t.Fatalf("want 1 non-orphaned JS, got %d", len(got))
	}
	if got[0].urlPath != "/linked.js" {
		t.Errorf("urlPath = %q, want /linked.js", got[0].urlPath)
	}
}

func TestCollectJSFiles_WellKnownIncluded(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/sw.js", "/sw.js", 50),
	}
	ctx := &HTMLContext{LinkedPages: map[string]bool{}}

	got := collectJSFiles(files, ctx)
	if len(got) != 1 {
		t.Fatalf("want 1 (well-known) JS, got %d", len(got))
	}
	if got[0].urlPath != "/sw.js" {
		t.Errorf("urlPath = %q, want /sw.js", got[0].urlPath)
	}
}

func TestCollectJSFiles_SortedBySize(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/small.js", "/small.js", 100),
		builtFile("/site/public/big.js", "/big.js", 500),
		builtFile("/site/public/mid.js", "/mid.js", 200),
	}
	ctx := &HTMLContext{LinkedPages: map[string]bool{
		"/small.js": true, "/big.js": true, "/mid.js": true,
	}}

	got := collectJSFiles(files, ctx)
	if len(got) != 3 {
		t.Fatalf("want 3 JS files, got %d", len(got))
	}
	if got[0].urlPath != "/big.js" {
		t.Errorf("first = %q, want /big.js", got[0].urlPath)
	}
	if got[1].urlPath != "/mid.js" {
		t.Errorf("second = %q, want /mid.js", got[1].urlPath)
	}
	if got[2].urlPath != "/small.js" {
		t.Errorf("third = %q, want /small.js", got[2].urlPath)
	}
}

func TestCollectJSFiles_EmptyWhenNoJS(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/index.html", "/", 0),
	}
	ctx := &HTMLContext{LinkedPages: map[string]bool{}}
	got := collectJSFiles(files, ctx)
	if len(got) != 0 {
		t.Fatalf("want 0 JS files, got %d", len(got))
	}
}

func TestCollectJSFiles_LinkedViaAlias(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/robots.txt", "/robots.txt", 100),
	}
	ctx := &HTMLContext{LinkedPages: map[string]bool{}}
	got := collectJSFiles(files, ctx)
	if len(got) != 0 {
		t.Fatalf("want 0 (not JS), got %d", len(got))
	}
}

func TestCollectJSFiles_SkippedExcluded(t *testing.T) {
	skipped := builtFile("/site/public/vendor/big.js", "/vendor/big.js", 900)
	skipped.Skipped = true
	files := []BuiltFile{
		skipped,
		builtFile("/site/public/app.js", "/app.js", 100),
	}
	ctx := &HTMLContext{LinkedPages: map[string]bool{
		"/vendor/big.js": true, "/app.js": true,
	}}
	got := collectJSFiles(files, ctx)
	if len(got) != 1 || got[0].urlPath != "/app.js" {
		t.Fatalf("want only non-skipped JS, got %+v", got)
	}
}
