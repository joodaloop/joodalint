package rules

import (
	"testing"

	"github.com/joodaloop/hugolint/internal/config"
)

func TestURLs_SchemeMissingColon(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("see https//example.com here\n"), nil)
	if !containsMsg(diags, "scheme missing colon") {
		t.Fatalf("want scheme-missing-colon, got %v", messages(diags))
	}
}

func TestURLs_UnknownScheme(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("see htps://example.com here\n"), nil)
	if !containsMsg(diags, "unknown or mistyped scheme") {
		t.Fatalf("want unknown-scheme diag, got %v", messages(diags))
	}
}

func TestURLs_MalformedSeparator(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("see https:///example.com here\n"), nil)
	if !containsMsg(diags, "malformed scheme separator") {
		t.Fatalf("want malformed-scheme-separator, got %v", messages(diags))
	}
}

func TestURLs_GoodURLNoDiag(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("see https://example.com/foo here\n"), nil)
	assertNoDiags(t, diags)
}

func TestURLs_TrailingPunctTrimmed(t *testing.T) {
	// Trailing comma should be stripped before validation, so this is fine.
	diags := markdownURLs{}.Check(mdFile("see https://example.com/foo, then\n"), nil)
	assertNoDiags(t, diags)
}

func TestURLs_SiteLocalAbsolute(t *testing.T) {
	cfg := &config.Config{Links: config.Links{SiteHosts: []string{"example.com"}}}
	ctx := &MarkdownContext{Config: cfg}
	diags := markdownURLs{}.Check(mdFile("see https://example.com/foo here\n"), ctx)
	if !containsMsg(diags, "site-local absolute URL") {
		t.Fatalf("want site-local diag, got %v", messages(diags))
	}
}

func TestURLs_SiteLocalIgnoresExternalHost(t *testing.T) {
	cfg := &config.Config{Links: config.Links{SiteHosts: []string{"example.com"}}}
	ctx := &MarkdownContext{Config: cfg}
	diags := markdownURLs{}.Check(mdFile("see https://other.com/foo here\n"), ctx)
	assertNoDiags(t, diags)
}

func TestURLs_ID(t *testing.T) {
	if (markdownURLs{}).ID() != "malformed-url" {
		t.Fatal("wrong ID")
	}
}
