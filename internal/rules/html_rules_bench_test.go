package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joodaloop/joodalint/internal/config"
)

func benchHTMLFile(refs int) *HTMLFile {
	links := make([]string, 0, refs*3)
	images := make([]string, 0, refs)
	assets := make([]Asset, 0, refs*2)
	ids := map[string]int{"main": 1}
	metas := []MetaTag{
		{Charset: "utf-8"},
		{Name: "viewport", Content: "width=device-width, initial-scale=1"},
		{Name: "description", Content: "Benchmark page description."},
		{Name: "twitter:card", Content: "summary_large_image"},
		{Name: "twitter:title", Content: "Benchmark Page"},
		{Name: "twitter:description", Content: "Benchmark page description."},
		{Name: "twitter:image", Content: "https://example.com/img/hero.png"},
		{Property: "og:title", Content: "Benchmark Page"},
		{Property: "og:description", Content: "Benchmark page description."},
		{Property: "og:url", Content: "https://example.com/bench/"},
		{Property: "og:image", Content: "https://example.com/img/hero.png"},
	}
	headLinks := []HeadLink{
		{Rel: "canonical", Href: "https://example.com/bench/"},
		{Rel: "alternate", Type: "application/rss+xml", Href: "/index.xml"},
	}
	for i := 0; i < refs; i++ {
		links = append(links,
			fmt.Sprintf("/page/%d/", i),
			fmt.Sprintf("/page/%d/#frag-%d", i, i),
			fmt.Sprintf("https://external.example/%d", i),
		)
		images = append(images, fmt.Sprintf("/img/%d.png", i))
		assets = append(assets,
			Asset{Tag: "script", Attr: "src", URL: fmt.Sprintf("/js/%d.js", i)},
			Asset{Tag: "link", Attr: "href", URL: fmt.Sprintf("/css/%d.css", i)},
		)
		ids[fmt.Sprintf("frag-%d", i)] = 1
	}
	return &HTMLFile{
		Path:      "public/bench/index.html",
		URLPath:   "/bench/",
		Links:     links,
		Images:    images,
		Assets:    assets,
		IDs:       ids,
		Text:      "body text",
		Title:     "Benchmark Page",
		Lang:      "en",
		Metas:     metas,
		HeadLinks: headLinks,
	}
}

func benchHTMLContext(refs int) *HTMLContext {
	pages := map[string]bool{
		"/bench/":       true,
		"/img/hero.png": true,
		"/index.xml":    true,
	}
	pageIDs := map[string]map[string]int{
		"/bench/": {"main": 1},
	}
	for i := 0; i < refs; i++ {
		pages[fmt.Sprintf("/page/%d/", i)] = true
		pages[fmt.Sprintf("/img/%d.png", i)] = true
		pages[fmt.Sprintf("/js/%d.js", i)] = true
		pages[fmt.Sprintf("/css/%d.css", i)] = true
		pageIDs[fmt.Sprintf("/page/%d/", i)] = map[string]int{fmt.Sprintf("frag-%d", i): 1}
	}
	return &HTMLContext{
		Pages:       pages,
		PageIDs:     pageIDs,
		LinkedPages: map[string]bool{},
		Config: &config.Config{
			Links: config.Links{SiteHosts: []string{"example.com"}},
		},
	}
}

func BenchmarkHTMLRules(b *testing.B) {
	f := benchHTMLFile(100)

	type ruleEntry struct {
		id    string
		check func() []Diagnostic
	}
	var entries []ruleEntry
	for _, r := range HTML() {
		r := r
		entries = append(entries, ruleEntry{
			id: r.ID(),
			check: func() []Diagnostic {
				return r.Check(f, benchHTMLContext(100))
			},
		})
	}

	for _, e := range entries {
		b.Run(e.id, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = e.check()
			}
		})
	}
}

func BenchmarkReportOrphans(b *testing.B) {
	ctx := benchHTMLContext(100)
	files := make([]BuiltFile, 0, 401)
	files = append(files, builtFile("public/index.html", "/", 0))
	for i := 0; i < 100; i++ {
		files = append(files,
			builtFile(fmt.Sprintf("public/page/%d/index.html", i), fmt.Sprintf("/page/%d/", i), 0),
			builtFile(fmt.Sprintf("public/img/%d.png", i), fmt.Sprintf("/img/%d.png", i), 0),
			builtFile(fmt.Sprintf("public/js/%d.js", i), fmt.Sprintf("/js/%d.js", i), 0),
			builtFile(fmt.Sprintf("public/css/%d.css", i), fmt.Sprintf("/css/%d.css", i), 0),
		)
		ctx.LinkedPages[fmt.Sprintf("/page/%d/", i)] = true
		ctx.LinkedPages[fmt.Sprintf("/img/%d.png", i)] = true
		ctx.LinkedPages[fmt.Sprintf("/js/%d.js", i)] = true
		ctx.LinkedPages[fmt.Sprintf("/css/%d.css", i)] = true
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ReportOrphans(files, ctx)
	}
}

func BenchmarkScanCSSLinks(b *testing.B) {
	dir := b.TempDir()
	ctx := benchHTMLContext(100)
	files := make([]BuiltFile, 0, 20)
	for i := 0; i < 20; i++ {
		path := filepath.Join(dir, fmt.Sprintf("site-%02d.css", i))
		var css strings.Builder
		for j := 0; j < 50; j++ {
			fmt.Fprintf(&css, ".icon-%d-%d{background-image:url('/img/%d.png')}\n", i, j, j)
			fmt.Fprintf(&css, ".font-%d-%d{src:url('/css/%d.css')}\n", i, j, j)
		}
		if err := os.WriteFile(path, []byte(css.String()), 0o644); err != nil {
			b.Fatal(err)
		}
		files = append(files, builtFile(path, fmt.Sprintf("/css/site-%02d.css", i), 0))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.LinkedPages = map[string]bool{}
		if err := ScanCSSLinks(files, ctx); err != nil {
			b.Fatal(err)
		}
	}
}
