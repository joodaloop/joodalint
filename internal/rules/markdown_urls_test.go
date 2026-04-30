package rules

import (
	"testing"

	"github.com/joodaloop/hugolint/internal/config"
)

// --- scheme syntax ---------------------------------------------------------

func TestURLs_SchemeMissingColon(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("see [x](https//example.com) here\n"), nil)
	if !containsMsg(diags, "scheme missing colon") {
		t.Fatalf("want scheme-missing-colon, got %v", messages(diags))
	}
}

func TestURLs_UnknownScheme(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("see [x](htps://example.com) here\n"), nil)
	if !containsMsg(diags, "unknown or mistyped scheme") {
		t.Fatalf("want unknown-scheme diag, got %v", messages(diags))
	}
}

func TestURLs_MalformedSeparator(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("see [x](https:///example.com) here\n"), nil)
	if !containsMsg(diags, "malformed scheme separator") {
		t.Fatalf("want malformed-scheme-separator, got %v", messages(diags))
	}
}

func TestURLs_GoodURLNoDiag(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("see [x](https://example.com/foo) here\n"), nil)
	assertNoDiags(t, diags)
}

// --- http warning + site-local --------------------------------------------

func TestURLs_HTTPWarns(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("see [x](http://example.com/foo) here\n"), nil)
	if !containsMsg(diags, "http:// URL") {
		t.Fatalf("want http warning, got %v", messages(diags))
	}
}

func TestURLs_AutoLinkChecked(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("see <http://example.com/foo> here\n"), nil)
	if !containsMsg(diags, "http:// URL") {
		t.Fatalf("want http warning on autolink, got %v", messages(diags))
	}
}

func TestURLs_ImageChecked(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("![alt](http://example.com/x.png)\n"), nil)
	if !containsMsg(diags, "http:// URL") {
		t.Fatalf("want http warning on image, got %v", messages(diags))
	}
}

func TestURLs_SiteLocalAbsolute(t *testing.T) {
	cfg := &config.Config{Links: config.Links{SiteHosts: []string{"example.com"}}}
	ctx := &MarkdownContext{Config: cfg}
	diags := markdownURLs{}.Check(mdFile("see [x](https://example.com/foo) here\n"), ctx)
	if !containsMsg(diags, "site-local absolute URL") {
		t.Fatalf("want site-local diag, got %v", messages(diags))
	}
}

func TestURLs_SiteLocalIgnoresExternalHost(t *testing.T) {
	cfg := &config.Config{Links: config.Links{SiteHosts: []string{"example.com"}}}
	ctx := &MarkdownContext{Config: cfg}
	diags := markdownURLs{}.Check(mdFile("see [x](https://other.com/foo) here\n"), ctx)
	assertNoDiags(t, diags)
}

func TestURLs_HTTPAndSiteLocalBothReport(t *testing.T) {
	cfg := &config.Config{Links: config.Links{SiteHosts: []string{"example.com"}}}
	ctx := &MarkdownContext{Config: cfg}
	diags := markdownURLs{}.Check(mdFile("see [x](http://example.com/foo) here\n"), ctx)
	if !containsMsg(diags, "http:// URL") {
		t.Fatalf("want http warning, got %v", messages(diags))
	}
	if !containsMsg(diags, "site-local absolute URL") {
		t.Fatalf("want site-local diag, got %v", messages(diags))
	}
}

// --- host validation ------------------------------------------------------

func TestURLs_ValidHostsOK(t *testing.T) {
	for _, in := range []string{
		"[a](https://example.com/path)\n",
		"[a](https://sub.example.co.uk)\n",
		"[a](https://example.com:8080/x)\n",
	} {
		diags := markdownURLs{}.Check(mdFile(in), nil)
		if len(diags) != 0 {
			t.Errorf("input %q should be valid, got %v", in, messages(diags))
		}
	}
}

func TestURLs_InvalidHosts(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"[a](https://example/path)\n", "host has no dot"},
		{"[a](https://exa..mple.com)\n", "consecutive dots"},
		{"[a](https://-bad.com)\n", "label starts or ends with hyphen"},
		{"[a](https://bad-.com)\n", "label starts or ends with hyphen"},
		{"[a](https://bad_host.com)\n", "invalid characters in host"},
	}
	for _, tc := range cases {
		diags := markdownURLs{}.Check(mdFile(tc.in), nil)
		if !containsMsg(diags, tc.want) {
			t.Errorf("input %q: want %q, got %v", tc.in, tc.want, messages(diags))
		}
	}
}

// --- skipped schemes ------------------------------------------------------

func TestURLs_SkippedSchemes(t *testing.T) {
	for _, in := range []string{
		"[a](mailto:foo@bar.com)\n",
		"[a](mailto:foo+tag@bar.com?subject=hi)\n",
		"[a](tel:+15551234)\n",
		"[a](javascript:alert\\(1\\))\n",
		"[a](data:text/plain,hi)\n",
	} {
		diags := markdownURLs{}.Check(mdFile(in), nil)
		if len(diags) != 0 {
			t.Errorf("input %q should be skipped, got %v", in, messages(diags))
		}
	}
}

// --- mailto validation ----------------------------------------------------

func TestURLs_InvalidMailto(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"[a](mailto:)\n", "empty mailto address"},
		{"[a](mailto:foo)\n", "mailto missing @"},
		{"[a](mailto:@bar.com)\n", "mailto empty local part"},
		{"[a](mailto:.foo@bar.com)\n", "mailto invalid local part"},
		{"[a](mailto:foo.@bar.com)\n", "mailto invalid local part"},
		{"[a](mailto:fo..o@bar.com)\n", "mailto invalid local part"},
		{"[a](mailto:foo,bar@bar.com)\n", "mailto invalid characters in local part"},
		{"[a](mailto:foo@bar)\n", "mailto host has no dot"},
		{"[a](mailto:foo@bad_host.com)\n", "mailto invalid characters in host"},
	}
	for _, tc := range cases {
		diags := markdownURLs{}.Check(mdFile(tc.in), nil)
		if !containsMsg(diags, tc.want) {
			t.Errorf("input %q: want %q, got %v", tc.in, tc.want, messages(diags))
		}
	}
}

// --- relative-link --------------------------------------------------------

func TestURLs_RelativePathFlagged(t *testing.T) {
	src := "see [foo](foo/bar.md) please\n"
	diags := markdownURLs{}.Check(mdFile(src), nil)
	if !containsMsg(diags, "relative link") {
		t.Fatalf("want relative-link diag, got %v", messages(diags))
	}
}

func TestURLs_DotPathFlagged(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("[foo](./bar.md)\n"), nil)
	if !containsMsg(diags, "relative link") {
		t.Fatalf("want relative-link diag, got %v", messages(diags))
	}
}

func TestURLs_RootRelativeOK(t *testing.T) {
	assertNoDiags(t, markdownURLs{}.Check(mdFile("[foo](/foo/bar)\n"), nil))
}

func TestURLs_FragmentOK(t *testing.T) {
	assertNoDiags(t, markdownURLs{}.Check(mdFile("[foo](#anchor)\n"), nil))
}

// --- ID -------------------------------------------------------------------

func TestURLs_ID(t *testing.T) {
	if (markdownURLs{}).ID() != "url" {
		t.Fatal("wrong ID")
	}
}
