package rules

import (
	"bytes"
	"strings"
	"testing"
)

func TestImageDiagnostics_LargeImage(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/images/big.jpg", "/images/big.jpg", 600 * 1024),
		builtFile("/site/public/images/small.jpg", "/images/small.jpg", 100 * 1024),
	}
	diags := ImageDiagnostics(files)
	if len(diags) != 1 {
		t.Fatalf("want 1 diag, got %v", messages(diags))
	}
	if diags[0].Rule != "image-size" || !strings.Contains(diags[0].Message, "600KB") {
		t.Errorf("unexpected diag: %+v", diags[0])
	}
}

func TestImageDiagnostics_PNGWithWebpEstimate(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/images/photo.png", "/images/photo.png", 100 * 1024),
	}
	diags := ImageDiagnostics(files)
	if len(diags) != 1 {
		t.Fatalf("want 1 diag, got %v", messages(diags))
	}
	if diags[0].Rule != "image-format" || !strings.Contains(diags[0].Message, "webp") {
		t.Errorf("unexpected diag: %+v", diags[0])
	}
	// 100KB * 0.74 = 74KB
	if !strings.Contains(diags[0].Message, "~74KB") {
		t.Errorf("want webp estimate ~74KB in %q", diags[0].Message)
	}
}

func TestImageDiagnostics_LargePNGGetsBoth(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/images/huge.png", "/images/huge.png", 1 << 20),
	}
	diags := ImageDiagnostics(files)
	if len(diags) != 2 {
		t.Fatalf("want 2 diags (size + format), got %v", messages(diags))
	}
}

func TestImageDiagnostics_NonImagesIgnored(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/app.js", "/app.js", 5 << 20),
		builtFile("/site/public/index.html", "/", 5 << 20),
	}
	diags := ImageDiagnostics(files)
	assertNoDiags(t, diags)
}

func TestImageDiagnostics_ConventionIconsNotFlaggedAsPNG(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/apple-touch-icon.png", "/apple-touch-icon.png", 10 * 1024),
		builtFile("/site/public/favicon.png", "/favicon.png", 10 * 1024),
		builtFile("/site/public/images/icon.png", "/images/icon.png", 10 * 1024),
	}
	diags := ImageDiagnostics(files)
	if len(diags) != 1 {
		t.Fatalf("want 1 diag (nested icon only), got %v", messages(diags))
	}
	if diags[0].Path != "/site/public/images/icon.png" {
		t.Errorf("flagged %q, want the non-convention PNG", diags[0].Path)
	}
}

func TestImageDiagnostics_SkippedFilesExcluded(t *testing.T) {
	f := builtFile("/site/public/images/legacy/old.png", "/images/legacy/old.png", 1<<20)
	f.Skipped = true
	diags := ImageDiagnostics([]BuiltFile{f})
	assertNoDiags(t, diags)
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0B"},
		{512, "512B"},
		{100 * 1024, "100KB"},
		{1 << 20, "1.0MB"},
		{(3 << 20) + (1 << 19), "3.5MB"},
	}
	for _, c := range cases {
		if got := formatSize(c.in); got != c.want {
			t.Errorf("formatSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestImageCategory(t *testing.T) {
	yes := []string{"a.png", "b.JPG", "images/c.webp", "e.avif"}
	no := []string{"a.js", "b.html", "c.json", "d.css", "noext", "d.svg"}
	for _, p := range yes {
		if CategoryForPath(p) != catImages {
			t.Errorf("CategoryForPath(%q) != catImages, want images", p)
		}
	}
	for _, p := range no {
		if CategoryForPath(p) == catImages {
			t.Errorf("CategoryForPath(%q) == catImages, want other category", p)
		}
	}
	if CategoryForPath("d.svg") != catSVG {
		t.Error(`CategoryForPath("d.svg") != catSVG`)
	}
}

func TestImageDiagnostics_LargeSVGFlagged(t *testing.T) {
	files := []BuiltFile{
		builtFile("/site/public/images/diagram.svg", "/images/diagram.svg", 600*1024),
	}
	diags := ImageDiagnostics(files)
	if len(diags) != 1 || diags[0].Rule != "image-size" {
		t.Fatalf("want 1 image-size diag for large SVG, got %v", messages(diags))
	}
}

func TestGzipSize(t *testing.T) {
	b := bytes.Repeat([]byte("hello world "), 1000)
	gz := GzipSize(b)
	if gz <= 0 || gz >= int64(len(b)) {
		t.Errorf("GzipSize = %d for %d repetitive bytes, want 0 < gz < input", gz, len(b))
	}
}

func TestCategoryGzip(t *testing.T) {
	yes := []string{"index.html", "app.js", "mod.mjs", "data.json", "style.css", "feed.xml", "robots.txt", "site.webmanifest", "icon.svg"}
	no := []string{"photo.png", "font.woff2", "clip.mp4", "doc.pdf", "mystery.bin"}
	for _, p := range yes {
		if !CategoryForPath(p).Gzip() {
			t.Errorf("CategoryForPath(%q).Gzip() = false, want true", p)
		}
	}
	for _, p := range no {
		if CategoryForPath(p).Gzip() {
			t.Errorf("CategoryForPath(%q).Gzip() = true, want false", p)
		}
	}
	if otherCategory.Gzip() {
		t.Error("otherCategory.Gzip() = true, want false")
	}
}
