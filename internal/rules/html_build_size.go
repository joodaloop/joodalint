package rules

import (
	"compress/gzip"
	"fmt"
	"path"
	"strings"
)

// sizeCategories groups build output by extension for the size summary.
// gzip marks text formats that are served compressed, where the gzipped
// total is worth reporting (already-compressed media is left alone).
var sizeCategories = []struct {
	name string
	gzip bool
	exts []string
}{
	{"html", true, []string{".html", ".htm"}},
	{"css", true, []string{".css"}},
	{"js", true, []string{".js", ".mjs", ".cjs"}},
	{"json", true, []string{".json", ".webmanifest", ".map"}},
	{"xml", true, []string{".xml", ".rss", ".atom"}},
	{"txt", true, []string{".txt"}},
	{"images", false, []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".avif", ".ico", ".bmp", ".tiff"}},
	{"fonts", false, []string{".woff", ".woff2", ".ttf", ".otf", ".eot"}},
	{"video", false, []string{".mp4", ".webm", ".mov", ".m4v", ".ogv"}},
	{"audio", false, []string{".mp3", ".ogg", ".oga", ".wav", ".m4a", ".aac", ".flac", ".opus"}},
	{"pdf", false, []string{".pdf"}},
	{"wasm", false, []string{".wasm"}},
}

// SizeCategory is an index into sizeCategories, assigned once per built
// file at load time; consumers compare against it instead of re-matching
// extensions.
type SizeCategory uint8

// otherCategory marks files whose extension matches no category.
const otherCategory SizeCategory = 255

// extCategory maps a lowercased extension to its index in sizeCategories.
var extCategory = func() map[string]int {
	m := map[string]int{}
	for i, c := range sizeCategories {
		for _, e := range c.exts {
			m[e] = i
		}
	}
	return m
}()

var (
	catJS     = mustCategory("js")
	catImages = mustCategory("images")
)

func mustCategory(name string) SizeCategory {
	for i, c := range sizeCategories {
		if c.name == name {
			return SizeCategory(i)
		}
	}
	panic("unknown size category " + name)
}

// CategoryForPath classifies a built file by its extension.
func CategoryForPath(p string) SizeCategory {
	if i, ok := extCategory[strings.ToLower(path.Ext(p))]; ok {
		return SizeCategory(i)
	}
	return otherCategory
}

// Gzip reports whether the category is a text format whose gzipped size
// is tracked (already-compressed media is left alone).
func (c SizeCategory) Gzip() bool {
	return int(c) < len(sizeCategories) && sizeCategories[c].gzip
}

func formatSize(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%dKB", n/1024)
	}
	return fmt.Sprintf("%dB", n)
}

// GzipSize compresses b (default gzip level) and returns the compressed
// byte count.
func GzipSize(b []byte) int64 {
	var n countingWriter
	zw := gzip.NewWriter(&n)
	zw.Write(b)
	zw.Close()
	return int64(n)
}

type countingWriter int64

func (w *countingWriter) Write(p []byte) (int, error) {
	*w += countingWriter(len(p))
	return len(p), nil
}

// ReportSizeSummary prints total on-disk size per file category, plus the
// actual gzipped size for text categories. Files whose extension matches
// no category land in a trailing "other" row, so the total always covers
// the whole build.
func ReportSizeSummary(files []BuiltFile, color bool) {
	other := len(sizeCategories) // index of the catch-all bucket
	sizes := make([]int64, other+1)
	counts := make([]int, other+1)
	gzipSizes := make([]int64, other+1)
	for _, f := range files {
		if f.Skipped {
			continue
		}
		i := int(f.Category)
		if i > other {
			i = other
		}
		sizes[i] += f.Size
		counts[i]++
		gzipSizes[i] += f.GzipSize
	}

	const (
		reset  = "\x1b[0m"
		bold   = "\x1b[1m"
		yellow = "\x1b[33m"
		cyan   = "\x1b[36m"
		dim    = "\x1b[2m"
	)
	paint := func(s, code string) string {
		if !color {
			return s
		}
		return code + s + reset
	}

	fmt.Println(paint("\nBuild size summary:", bold))
	// transfer approximates what a full-site fetch sends over the wire:
	// gzipped size for the text categories, raw size for everything else.
	// A plain "total gzipped" would be misleading, since media categories
	// have no gzip figure to contribute.
	var total, transfer int64
	var totalCount int
	for i := 0; i <= other; i++ {
		total += sizes[i]
		totalCount += counts[i]
		if i < other && sizeCategories[i].gzip {
			transfer += gzipSizes[i]
		} else {
			transfer += sizes[i]
		}
		if counts[i] == 0 {
			continue
		}
		name, showGzip := "other", false
		if i < other {
			name, showGzip = sizeCategories[i].name, sizeCategories[i].gzip
		}
		files := "files"
		if counts[i] == 1 {
			files = "file"
		}
		gz := ""
		if showGzip {
			gz = "  " + paint(fmt.Sprintf("%s gzipped", formatSize(gzipSizes[i])), dim)
		}
		fmt.Printf("  %s  %s  %s%s\n",
			paint(fmt.Sprintf("%-6s", name), cyan),
			paint(fmt.Sprintf("%8s", formatSize(sizes[i])), yellow),
			paint(fmt.Sprintf("(%-9s", fmt.Sprintf("%d %s)", counts[i], files)), dim),
			gz)
	}
	fmt.Printf("  %s  %s  %s  %s\n",
		paint(fmt.Sprintf("%-6s", "total"), bold),
		paint(fmt.Sprintf("%8s", formatSize(total)), yellow),
		paint(fmt.Sprintf("(%-9s", fmt.Sprintf("%d files)", totalCount)), dim),
		paint(fmt.Sprintf("~%s transfer", formatSize(transfer)), dim))
}

const largeImageBytes = 500 * 1024

// webpLosslessRatio estimates lossless webp output size relative to PNG
// input (lossless webp averages ~26% smaller than PNG).
const webpLosslessRatio = 0.74

// ImageDiagnostics flags images over the size limit and PNGs that would
// shrink when converted to webp.
func ImageDiagnostics(files []BuiltFile) []Diagnostic {
	var diags []Diagnostic
	for _, f := range files {
		if f.Skipped || f.Category != catImages {
			continue
		}
		if f.Size > largeImageBytes {
			diags = append(diags, Diagnostic{
				Path: f.Path, Rule: "image-size",
				Message: fmt.Sprintf("large image: %s (limit 500KB)", formatSize(f.Size)),
			})
		}
		// Root-level convention icons (favicon.png, apple-touch-icon.png)
		// are deliberately PNG — Apple's touch-icon spec requires it — so
		// the webp suggestion is suppressed for them.
		if strings.EqualFold(path.Ext(f.Path), ".png") && !isWellKnown(f.URLPath) {
			est := int64(float64(f.Size) * webpLosslessRatio)
			diags = append(diags, Diagnostic{
				Path: f.Path, Rule: "image-format",
				Message: fmt.Sprintf("PNG image (%s), ~%s as lossless webp", formatSize(f.Size), formatSize(est)),
			})
		}
	}
	return diags
}
