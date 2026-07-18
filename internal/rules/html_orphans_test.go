package rules

import "testing"

func orphanCtx(linked map[string]bool) *HTMLContext {
	return &HTMLContext{LinkedPages: linked}
}

func TestIsEntryPoint(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"/", true},
		{"/index.html", true},
		{"/foo/", false},
		{"/foo.html", false},
	}
	for _, tc := range cases {
		if got := isEntryPoint(tc.in); got != tc.want {
			t.Errorf("isEntryPoint(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsWellKnown(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"/.hidden", true},
		{"/favicon.ico", true},
		{"/favicon.png", true},
		{"/apple-touch-icon.png", true},
		{"/apple-touch-icon-precomposed.png", true},
		{"/apple-touch-icon-120x120.png", true},
		{"/404.html", true},
		{"/robots.txt", true},
		{"/sitemap.xml", true},
		{"/sw.js", true},
		{"/manifest.json", true},
		{"/foo.html", false},
		{"/bar", false},
		// Convention names are only excused at the site root.
		{"/old/favicon.ico", false},
		{"/assets/apple-touch-icon.png", false},
		{"/backup/404.html", false},
		{"/vendor/sw.js", false},
		// Dotfiles stay excused at any depth.
		{"/notes/.hidden", true},
	}
	for _, tc := range cases {
		if got := isWellKnown(tc.in); got != tc.want {
			t.Errorf("isWellKnown(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestPageAliases(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"/foo/", []string{"/foo/", "/foo", "/foo/index.html"}},
		{"/foo.html", []string{"/foo.html"}},
	}
	for _, tc := range cases {
		got := pageAliases(tc.in)
		if len(got) != len(tc.want) {
			t.Fatalf("pageAliases(%q) = %v, want %v", tc.in, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("pageAliases(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

func TestReportOrphans_EntryPointNotOrphaned(t *testing.T) {
	files := []BuiltFile{
		{Path: "/site/public/index.html", URLPath: "/"},
	}
	ctx := orphanCtx(map[string]bool{})
	assertNoDiags(t, ReportOrphans(files, ctx))
}

func TestReportOrphans_WellKnownNotOrphaned(t *testing.T) {
	files := []BuiltFile{
		{Path: "/site/public/404.html", URLPath: "/404.html"},
		{Path: "/site/public/robots.txt", URLPath: "/robots.txt"},
		{Path: "/site/public/.hidden", URLPath: "/.hidden"},
	}
	ctx := orphanCtx(map[string]bool{})
	assertNoDiags(t, ReportOrphans(files, ctx))
}

func TestReportOrphans_LinkedNotOrphaned(t *testing.T) {
	files := []BuiltFile{
		{Path: "/site/public/foo/index.html", URLPath: "/foo/"},
	}
	ctx := orphanCtx(map[string]bool{"/foo/": true})
	assertNoDiags(t, ReportOrphans(files, ctx))
}

func TestReportOrphans_LinkedViaAliasNotOrphaned(t *testing.T) {
	files := []BuiltFile{
		{Path: "/site/public/foo/index.html", URLPath: "/foo/"},
	}
	ctx := orphanCtx(map[string]bool{"/foo": true})
	assertNoDiags(t, ReportOrphans(files, ctx))
}

func TestReportOrphans_UnlinkedIsOrphaned(t *testing.T) {
	files := []BuiltFile{
		{Path: "/site/public/bar.html", URLPath: "/bar.html"},
	}
	ctx := orphanCtx(map[string]bool{})
	diags := ReportOrphans(files, ctx)
	if !containsMsg(diags, "is not linked to") {
		t.Fatalf("want orphan-file diag, got %v", messages(diags))
	}
}

func TestReportOrphans_SkippedNotReported(t *testing.T) {
	skipped := builtFile("/site/public/vendor/lib.css", "/vendor/lib.css", 100)
	skipped.Skipped = true
	files := []BuiltFile{
		skipped,
		builtFile("/site/public/unlinked.css", "/unlinked.css", 100),
	}
	ctx := &HTMLContext{LinkedPages: map[string]bool{}}
	diags := ReportOrphans(files, ctx)
	if len(diags) != 1 || diags[0].Path != "/site/public/unlinked.css" {
		t.Fatalf("want only the non-skipped orphan, got %v", diags)
	}
}
