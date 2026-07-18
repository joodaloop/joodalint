package rules

import (
	"os"
	"path"
	"regexp"
	"strings"
)

type BuiltFile struct {
	Path    string
	URLPath string
	// Category is the file's size-summary classification, assigned once
	// by CategoryForPath when the file is loaded.
	Category SizeCategory
	Size     int64
	// GzipSize is the gzipped byte count, computed at load time for
	// non-skipped text files; 0 for everything else.
	GzipSize int64
	// Skipped marks files matched by build_skip: still valid link targets,
	// but excluded from rules and size accounting.
	Skipped bool
}

// IsHTML reports whether the file is classified as an HTML page.
func (f BuiltFile) IsHTML() bool { return f.Category == catHTML }

// IsCSS reports whether the file is classified as a stylesheet.
func (f BuiltFile) IsCSS() bool { return f.Category == catCSS }

// isLinked reports whether a built file is reachable: an entry point, a
// well-known file, or linked to (under any alias) from another page.
func isLinked(f BuiltFile, ctx *HTMLContext) bool {
	if isEntryPoint(f.URLPath) || isWellKnown(f.URLPath) {
		return true
	}
	for _, alias := range pageAliases(f.URLPath) {
		if ctx.LinkedPages[alias] {
			return true
		}
	}
	return false
}

func ReportOrphans(files []BuiltFile, ctx *HTMLContext) []Diagnostic {
	var diags []Diagnostic
	for _, f := range files {
		if f.Skipped || isLinked(f, ctx) {
			continue
		}
		diags = append(diags, Diagnostic{
			Path:    f.Path,
			Rule:    "orphan-file",
			Message: "file is not linked to from any other page",
		})
	}
	return diags
}

func isEntryPoint(urlPath string) bool {
	return urlPath == "/" || urlPath == "" || urlPath == "/index.html"
}

func isWellKnown(urlPath string) bool {
	base := path.Base(urlPath)
	if strings.HasPrefix(base, ".") {
		return true
	}
	// The remaining names are excused because browsers and crawlers fetch
	// them by convention without any markup reference — but that convention
	// only covers the site root. A copy in a subfolder is never fetched
	// implicitly, so it goes through the normal orphan check.
	if path.Dir(urlPath) != "/" {
		return false
	}
	if strings.HasPrefix(base, "favicon.") || strings.HasPrefix(base, "apple-touch-icon") {
		return true
	}
	switch base {
	case "404.html", "robots.txt", "sitemap.xml", "sw.js", "manifest.json":
		return true
	}
	return false
}

func pageAliases(urlPath string) []string {
	aliases := []string{urlPath}
	if strings.HasSuffix(urlPath, "/") {
		aliases = append(aliases, strings.TrimSuffix(urlPath, "/"))
		aliases = append(aliases, urlPath+"index.html")
	}
	return aliases
}

var cssURLRegex = regexp.MustCompile(`url\(\s*['"]?([^'")\s]+)['"]?\s*\)`)

func ScanCSSLinks(files []BuiltFile, ctx *HTMLContext) error {
	for _, f := range files {
		// Skipped CSS contributes no outgoing links, matching skipped HTML
		// (which is never parsed at all).
		if f.Skipped || !f.IsCSS() {
			continue
		}
		b, err := os.ReadFile(f.Path)
		if err != nil {
			return err
		}
		for _, m := range cssURLRegex.FindAllSubmatch(b, -1) {
			ref := string(m[1])
			if strings.HasPrefix(ref, "data:") {
				continue
			}
			if !isRelative(ref) {
				continue
			}
			resolved, ok := resolve(f.URLPath, ref)
			if !ok {
				continue
			}
			ctx.MarkLinked(resolved)
		}
	}
	return nil
}
