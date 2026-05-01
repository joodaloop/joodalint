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

// --- emptiness ------------------------------------------------------------

func TestURLs_EmptyLinkURL(t *testing.T) {
	for _, in := range []string{
		"see [text]() here\n",
		"see [text]( ) here\n", // goldmark normalizes whitespace-only dest to ""
	} {
		diags := markdownURLs{}.Check(mdFile(in), nil)
		if !containsMsg(diags, "empty link URL") {
			t.Errorf("input %q: want empty link URL diag, got %v", in, messages(diags))
		}
	}
}

func TestURLs_EmptyImageURL(t *testing.T) {
	for _, in := range []string{
		"see ![alt]() here\n",
		"see ![alt]( ) here\n",
	} {
		diags := markdownURLs{}.Check(mdFile(in), nil)
		if !containsMsg(diags, "empty image URL") {
			t.Errorf("input %q: want empty image URL diag, got %v", in, messages(diags))
		}
	}
}

func TestURLs_EmptyLinkText(t *testing.T) {
	for _, in := range []string{
		"see [](https://example.com) here\n",
		"see [ ](https://example.com) here\n",
	} {
		diags := markdownURLs{}.Check(mdFile(in), nil)
		if !containsMsg(diags, "empty link text") {
			t.Errorf("input %q: want empty link text diag, got %v", in, messages(diags))
		}
	}
}

func TestURLs_EmptyImageAlt(t *testing.T) {
	for _, in := range []string{
		"see ![](/foo.png) here\n",
		"see ![ ](/foo.png) here\n",
	} {
		diags := markdownURLs{}.Check(mdFile(in), nil)
		if !containsMsg(diags, "empty image alt") {
			t.Errorf("input %q: want empty image alt diag, got %v", in, messages(diags))
		}
	}
}

func TestURLs_BothEmpty(t *testing.T) {
	// `[]()` should produce both empty-url and empty-link-text.
	diags := markdownURLs{}.Check(mdFile("[]()\n"), nil)
	if !containsMsg(diags, "empty link URL") {
		t.Errorf("want empty link URL diag, got %v", messages(diags))
	}
	if !containsMsg(diags, "empty link text") {
		t.Errorf("want empty link text diag, got %v", messages(diags))
	}
}

func TestURLs_NonEmptyTextAndURLOK(t *testing.T) {
	for _, in := range []string{
		"[text](https://example.com)\n",
		"![real alt](/x.png)\n",
		"<https://example.com>\n",
	} {
		diags := markdownURLs{}.Check(mdFile(in), nil)
		for _, d := range diags {
			if d.Rule == "empty-url" || d.Rule == "empty-link-text" || d.Rule == "empty-image-alt" {
				t.Errorf("input %q: unexpected emptiness diag %q", in, d.Message)
			}
		}
	}
}

// --- url-chars whitelist --------------------------------------------------

func TestURLs_UnsafeCharsFlagged(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"[a](https://x.com/“hi”)\n", "unencoded characters"},      // smart quotes
		{"[a](https://x.com/\"foo)\n", "unencoded characters"},     // straight dquote
		{"[a](https://x.com/foo|bar)\n", "unencoded characters"},   // pipe
		{"[a](https://x.com/foo^bar)\n", "unencoded characters"},   // caret
		{"[a](https://x.com/foo`bar)\n", "unencoded characters"},   // backtick
		{"[a](https://x.com/foo{x})\n", "unencoded characters"},    // braces
		{"[a](https://x.com/foo\\bar)\n", "unencoded characters"},  // backslash
		{"[a](https://x.com/<foo)\n", "unencoded characters"},      // less-than
		{"[a](https://x.com/café)\n", "unencoded characters"},      // raw unicode
		{"[a](<https://x.com/foo bar>)\n", "unencoded characters"}, // angle-bracket form lets a space through
	}
	for _, tc := range cases {
		diags := markdownURLs{}.Check(mdFile(tc.in), nil)
		if !containsMsg(diags, tc.want) {
			t.Errorf("input %q: want url-chars diag, got %v", tc.in, messages(diags))
		}
	}
}

func TestURLs_SafeCharsOK(t *testing.T) {
	for _, in := range []string{
		"[a](https://x.com/foo/bar)\n",
		"[a](https://x.com/foo?q=1&r=2#frag)\n",
		"[a](https://x.com/path-with_chars.ext~v2)\n",
		"[a](https://x.com/already%20encoded)\n",
		"[a](https://x.com/(parens))\n",
		"[a](https://x.com/path/with(balanced)parens)\n",
	} {
		diags := markdownURLs{}.Check(mdFile(in), nil)
		if containsMsg(diags, "unencoded characters") {
			t.Errorf("input %q: should not flag, got %v", in, messages(diags))
		}
	}
}

// --- image-alt (merged from markdown_image_alt.go) ------------------------

func TestImageAlt_GenericAlts(t *testing.T) {
	cases := []string{
		"![image](/foo.png)\n",
		"![img](/foo.png)\n",
		"![picture](/foo.png)\n",
		"![pic](/foo.png)\n",
		"![photo](/foo.png)\n",
		"![screenshot](/foo.png)\n",
		"![figure](/foo.png)\n",
		"![alt](/foo.png)\n",
		"![alt text](/foo.png)\n",
		"![ Image ](/foo.png)\n",
	}
	for _, in := range cases {
		diags := markdownImageAlt{}.Check(mdFile(in), nil)
		if len(diags) == 0 {
			t.Errorf("input %q: expected diag", in)
		}
	}
}

// Empty/whitespace alts are owned by the empty-image-alt rule (markdownURLs);
// the image-alt rule no longer flags them.
func TestImageAlt_EmptyNotFlagged(t *testing.T) {
	for _, in := range []string{
		"![](/foo.png)\n",
		"![ ](/foo.png)\n",
	} {
		diags := markdownImageAlt{}.Check(mdFile(in), nil)
		if len(diags) != 0 {
			t.Errorf("input %q: expected no image-alt diag, got %v", in, messages(diags))
		}
	}
}

func TestImageAlt_DescriptiveAltOK(t *testing.T) {
	diags := markdownImageAlt{}.Check(mdFile("![A black cat sleeping](/cat.png)\n"), nil)
	assertNoDiags(t, diags)
}

func TestImageAlt_MultiplePerLine(t *testing.T) {
	src := "![pic](/a.png) and ![real text](/b.png) and ![image](/c.png)\n"
	diags := markdownImageAlt{}.Check(mdFile(src), nil)
	if len(diags) != 2 {
		t.Fatalf("want 2 diags, got %d: %v", len(diags), messages(diags))
	}
}

func TestImageAlt_LineNumber(t *testing.T) {
	src := "para\n\n![image](/x.png)\n"
	diags := markdownImageAlt{}.Check(mdFile(src), nil)
	if len(diags) != 1 || diags[0].Line != 3 {
		t.Fatalf("want line 3, got %+v", diags)
	}
}

func TestImageAlt_ID(t *testing.T) {
	if (markdownImageAlt{}).ID() != "image-alt" {
		t.Fatal("wrong ID")
	}
}

// --- ID -------------------------------------------------------------------

func TestURLs_ID(t *testing.T) {
	if (markdownURLs{}).ID() != "url" {
		t.Fatal("wrong ID")
	}
}

// --- protocol-relative ------------------------------------------------------

func TestURLs_ProtocolRelativeFlagged(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("[text](//example.com/foo)\n"), nil)
	if !containsMsg(diags, "specify a URL protocol") {
		t.Fatalf("want protocol-relative-url diag, got %v", messages(diags))
	}
}

func TestURLs_ProtocolRelativeImageFlagged(t *testing.T) {
	diags := markdownURLs{}.Check(mdFile("![alt](//example.com/img.png)\n"), nil)
	if !containsMsg(diags, "specify a URL protocol") {
		t.Fatalf("want protocol-relative-url diag for image, got %v", messages(diags))
	}
}

// --- spaces-around-link -----------------------------------------------------

func TestURLs_SpacesAroundLinkFlagged(t *testing.T) {
	cases := []string{
		"[ text ](https://example.com)\n",
		"[text ](https://example.com)\n",
		"[ text](https://example.com)\n",
	}
	for _, in := range cases {
		diags := markdownURLs{}.Check(mdFile(in), nil)
		if !containsMsg(diags, "link text contains extra spaces") {
			t.Errorf("input %q: want spaces-around-link diag, got %v", in, messages(diags))
		}
	}
}

func TestURLs_NoSpacesAroundLinkOK(t *testing.T) {
	assertNoDiags(t, markdownURLs{}.Check(mdFile("[text](https://example.com)\n"), nil))
}

// --- link-punctuation -------------------------------------------------------

func TestURLs_LinkTrailingPunctuationFlagged(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"[documentation.](https://example.com)\n", `link includes trailing "."`},
		{"[docs,](https://example.com)\n", `link includes trailing ","`},
		{"[docs;](https://example.com)\n", `link includes trailing ";"`},
		{"[docs:](https://example.com)\n", `link includes trailing ":"`},
		{"[docs!](https://example.com)\n", `link includes trailing "!"`},
		{"[docs?](https://example.com)\n", `link includes trailing "?"`},
	}
	for _, tc := range cases {
		diags := markdownURLs{}.Check(mdFile(tc.in), nil)
		if !containsMsg(diags, tc.want) {
			t.Errorf("input %q: want %q, got %v", tc.in, tc.want, messages(diags))
		}
	}
}

func TestURLs_LinkNoTrailingPunctuationOK(t *testing.T) {
	assertNoDiags(t, markdownURLs{}.Check(mdFile("[documentation](https://example.com)\n"), nil))
}

func TestURLs_LongLinkText(t *testing.T) {
	long := ""
	for i := 0; i < 121; i++ {
		long += "a"
	}
	src := "[" + long + "](https://example.com)\n"
	diags := markdownURLs{}.Check(mdFile(src), nil)
	if !containsMsg(diags, "keep link text concise") {
		t.Fatalf("want long-link-text diag, got %v", messages(diags))
	}
}

func TestURLs_ShortLinkTextOK(t *testing.T) {
	assertNoDiags(t, markdownURLs{}.Check(mdFile("[short text](https://example.com)\n"), nil))
}
