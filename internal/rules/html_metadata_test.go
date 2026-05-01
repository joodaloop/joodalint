package rules

import (
	"testing"

	"github.com/joodaloop/hugolint/internal/config"
)

// goodMetaFile returns an HTMLFile with a complete, valid head metadata set
// pointing at /index.html on host pager.joodaloop.com.
func goodMetaFile() *HTMLFile {
	return &HTMLFile{
		Path:    "/site/public/index.html",
		URLPath: "/index.html",
		Title:   "Pager",
		Lang:    "en",
		Metas: []MetaTag{
			{Charset: "utf-8"},
			{HTTPEquiv: "Content-Type", Content: "text/html; charset=UTF-8"},
			{Name: "viewport", Content: "width=device-width, initial-scale=1.0"},
			{Name: "description", Content: "A very interesting page"},
			{Property: "og:url", Content: "https://pager.joodaloop.com/"},
			{Property: "og:title", Content: "Pager"},
			{Property: "og:description", Content: "A very interesting page"},
			{Property: "og:image", Content: "https://pager.joodaloop.com/assets/card.jpg"},
			{Name: "twitter:card", Content: "summary_large_image"},
			{Name: "twitter:title", Content: "Pager"},
			{Name: "twitter:description", Content: "A very interesting page"},
			{Name: "twitter:image", Content: "https://pager.joodaloop.com/assets/card.jpg"},
		},
		HeadLinks: []HeadLink{
			{Rel: "alternate", Type: "text/markdown", Href: "/index.md", Title: "Markdown version"},
		},
	}
}

func metaCtx() *HTMLContext {
	return &HTMLContext{
		Root: "/site/public",
		Pages: map[string]bool{
			"/":                true,
			"/index.html":      true,
			"/index.md":        true,
			"/assets/card.jpg": true,
		},
		Config: &config.Config{Links: config.Links{SiteHosts: []string{"pager.joodaloop.com"}}},
	}
}

func TestHeadMetadata_Valid(t *testing.T) {
	diags := headMetadata{}.Check(goodMetaFile(), metaCtx())
	assertNoDiags(t, diags)
}

func TestHeadMetadata_MissingTitle(t *testing.T) {
	f := goodMetaFile()
	f.Title = ""
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "missing or empty <title>") {
		t.Fatalf("want missing-title diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_MissingCharset(t *testing.T) {
	f := goodMetaFile()
	// drop the first meta (charset)
	f.Metas = f.Metas[1:]
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "missing <meta charset") {
		t.Fatalf("want missing-charset diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_NonUTF8Charset(t *testing.T) {
	f := goodMetaFile()
	f.Metas[0] = MetaTag{Charset: "iso-8859-1"}
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "should be utf-8") {
		t.Fatalf("want non-utf8 charset diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_ContentTypeCharsetMismatch(t *testing.T) {
	f := goodMetaFile()
	f.Metas[1] = MetaTag{HTTPEquiv: "Content-Type", Content: "text/html; charset=iso-8859-1"}
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "Content-Type") {
		t.Fatalf("want Content-Type charset diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_MissingViewport(t *testing.T) {
	f := goodMetaFile()
	f.Metas[2] = MetaTag{Name: "viewport", Content: ""}
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "viewport") {
		t.Fatalf("want viewport diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_ViewportMissingWidth(t *testing.T) {
	f := goodMetaFile()
	f.Metas[2] = MetaTag{Name: "viewport", Content: "initial-scale=1.0"}
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "width=device-width") {
		t.Fatalf("want width=device-width diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_MissingDescription(t *testing.T) {
	f := goodMetaFile()
	f.Metas[3] = MetaTag{Name: "description", Content: ""}
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, `name="description"`) {
		t.Fatalf("want missing description diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_MissingOGTags(t *testing.T) {
	f := goodMetaFile()
	// strip every og:* meta
	kept := f.Metas[:0]
	for _, m := range f.Metas {
		if len(m.Property) >= 3 && m.Property[:3] == "og:" {
			continue
		}
		kept = append(kept, m)
	}
	f.Metas = kept
	diags := headMetadata{}.Check(f, metaCtx())
	for _, key := range []string{"og:title", "og:description", "og:url", "og:image"} {
		if !containsMsg(diags, key) {
			t.Errorf("want diag mentioning %s, got %v", key, messages(diags))
		}
	}
}

func TestHeadMetadata_OGTitleMismatch(t *testing.T) {
	f := goodMetaFile()
	for i, m := range f.Metas {
		if m.Property == "og:title" {
			f.Metas[i].Content = "Different"
		}
	}
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "og:title") || !containsMsg(diags, "does not match") {
		t.Fatalf("want og:title mismatch diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_OGDescriptionMismatch(t *testing.T) {
	f := goodMetaFile()
	for i, m := range f.Metas {
		if m.Property == "og:description" {
			f.Metas[i].Content = "Different"
		}
	}
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "og:description does not match") {
		t.Fatalf("want og:description mismatch diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_TwitterMirrors(t *testing.T) {
	f := goodMetaFile()
	for i, m := range f.Metas {
		switch m.Name {
		case "twitter:title":
			f.Metas[i].Content = "Different"
		case "twitter:description":
			f.Metas[i].Content = "Different"
		}
	}
	diags := headMetadata{}.Check(f, metaCtx())
	for _, want := range []string{"twitter:title", "twitter:description does not match"} {
		if !containsMsg(diags, want) {
			t.Errorf("want diag containing %q, got %v", want, messages(diags))
		}
	}
}

func TestHeadMetadata_BadTwitterCard(t *testing.T) {
	f := goodMetaFile()
	for i, m := range f.Metas {
		if m.Name == "twitter:card" {
			f.Metas[i].Content = "bogus"
		}
	}
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "twitter:card has unexpected value") {
		t.Fatalf("want twitter:card diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_OGURLNotAbsolute(t *testing.T) {
	f := goodMetaFile()
	for i, m := range f.Metas {
		if m.Property == "og:url" {
			f.Metas[i].Content = "/index.html"
		}
	}
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "must be an absolute URL") {
		t.Fatalf("want absolute-URL diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_OGURLWrongHost(t *testing.T) {
	f := goodMetaFile()
	for i, m := range f.Metas {
		if m.Property == "og:url" {
			f.Metas[i].Content = "https://example.com/"
		}
	}
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "site_hosts") {
		t.Fatalf("want site_hosts diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_OGURLWrongPath(t *testing.T) {
	f := goodMetaFile()
	for i, m := range f.Metas {
		if m.Property == "og:url" {
			f.Metas[i].Content = "https://pager.joodaloop.com/wrong/"
		}
	}
	diags := headMetadata{}.Check(f, metaCtx())
	if !containsMsg(diags, "og:url path") {
		t.Fatalf("want og:url path diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_OGImageMissingAsset(t *testing.T) {
	f := goodMetaFile()
	ctx := metaCtx()
	delete(ctx.Pages, "/assets/card.jpg")
	diags := headMetadata{}.Check(f, ctx)
	if !containsMsg(diags, "og:image") || !containsMsg(diags, "does not exist") {
		t.Fatalf("want og:image missing-asset diag, got %v", messages(diags))
	}
}

func TestHeadMetadata_NoSiteHostsSkipsHostCheck(t *testing.T) {
	f := goodMetaFile()
	for i, m := range f.Metas {
		if m.Property == "og:url" {
			f.Metas[i].Content = "https://anything.example.com/index.html"
		}
	}
	ctx := metaCtx()
	ctx.Config = &config.Config{}
	diags := headMetadata{}.Check(f, ctx)
	if containsMsg(diags, "site_hosts") {
		t.Fatalf("did not expect site_hosts diag with no hosts configured: %v", messages(diags))
	}
}

func TestHeadMetadata_ID(t *testing.T) {
	if (headMetadata{}).ID() != "head-metadata" {
		t.Fatal("wrong ID")
	}
}

func TestHeadMetadata_MissingOrEmptyLang(t *testing.T) {
	for _, lang := range []string{"", "   "} {
		f := goodMetaFile()
		f.Lang = lang
		diags := headMetadata{}.Check(f, metaCtx())
		if !containsMsg(diags, `missing <html lang`) {
			t.Fatalf("want missing-lang diag for lang=%q, got %v", lang, messages(diags))
		}
	}
}
