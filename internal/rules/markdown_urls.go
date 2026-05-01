package rules

import (
	"fmt"
	urlpkg "net/url"
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
)

func init() {
	RegisterMarkdownAST(&markdownURLs{})
	RegisterMarkdownAST(&markdownImageAlt{})
}

type markdownURLs struct{}

func (markdownURLs) ID() string { return "url" }

type markdownImageAlt struct{}

func (markdownImageAlt) ID() string { return "image-alt" }

// genericAlts are non-empty alt strings that convey no useful information.
// Empty/whitespace alts are reported by the empty-image-alt diagnostic
// inside markdownURLs.
const maxLinkTextLen = 120
const maxLinkPunctuationLen = 20

var genericAlts = map[string]bool{
	"image":      true,
	"img":        true,
	"picture":    true,
	"pic":        true,
	"photo":      true,
	"screenshot": true,
	"figure":     true,
	"alt":        true,
	"alt text":   true,
}

func (markdownImageAlt) Check(f *MarkdownFile, _ *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
	ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		img, ok := n.(*ast.Image)
		if !ok {
			return ast.WalkContinue, nil
		}
		raw := nodeText(img, f.Body)
		alt := strings.ToLower(strings.TrimSpace(raw))
		if genericAlts[alt] {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: f.NodeLine(img), Rule: "image-alt",
				Message: fmt.Sprintf("useless image alt text: %q", raw),
			})
		}
		return ast.WalkSkipChildren, nil
	})
	return diags
}

var (
	schemeNoColon   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*//`)
	schemeSeparator = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9+.\-]*):(/+)`)
	hostAllowed     = regexp.MustCompile(`^[a-zA-Z0-9.\-]+$`)
	emailLocal      = regexp.MustCompile(`^[A-Za-z0-9!#$%&'*+/=?^_` + "`" + `{|}~.-]+$`)
	knownSchemes    = map[string]bool{"http": true, "https": true}
	skipSchemes     = map[string]bool{"tel": true, "javascript": true, "data": true}
	// RFC 3986 set: unreserved + pct-encoded + gen-delims + sub-delims.
	// Anything outside this must be percent-encoded.
	urlSafeChar = regexp.MustCompile(`^[A-Za-z0-9\-._~%:/?#\[\]@!$&'()*+,;=]+$`)
)

func (markdownURLs) Check(f *MarkdownFile, ctx *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
	siteHosts := configuredSiteHosts(ctx)
	ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var raw, kind string
		switch v := n.(type) {
		case *ast.Link:
			raw, kind = string(v.Destination), "link"
		case *ast.Image:
			raw, kind = string(v.Destination), "image"
		case *ast.AutoLink:
			raw, kind = string(v.URL(f.Body)), "autolink"
		default:
			return ast.WalkContinue, nil
		}
		line := f.NodeLine(n)

		// Emptiness checks. Autolinks always have a URL and no separate
		// text node, so skip them here.
		if kind != "autolink" {
			if strings.TrimSpace(raw) == "" {
				diags = append(diags, Diagnostic{
					Path: f.Path, Line: line, Rule: "empty-url",
					Message: fmt.Sprintf("empty %s URL", kind),
				})
			}
			text := nodeText(n, f.Body)
			if strings.TrimSpace(text) == "" {
				rule, msg := "empty-link-text", "empty link text"
				if kind == "image" {
					rule, msg = "empty-image-alt", "empty image alt"
				}
				diags = append(diags, Diagnostic{
					Path: f.Path, Line: line, Rule: rule, Message: msg,
				})
			}
			if kind == "link" {
				if n := len([]rune(text)); n > maxLinkTextLen {
					diags = append(diags, Diagnostic{
						Path: f.Path, Line: line, Rule: "long-link-text",
						Message: fmt.Sprintf("link text %d chars — keep link text concise", n),
					})
				}
				if strings.TrimSpace(text) != text {
					diags = append(diags, Diagnostic{
						Path: f.Path, Line: line, Rule: "spaces-around-link",
						Message: "link text contains extra spaces",
					})
				}
				if text != "" && len([]rune(text)) < maxLinkPunctuationLen {
					if last := text[len(text)-1]; strings.ContainsRune(`.,;:!?'"()`, rune(last)) {
						if !(last == ')' && strings.ContainsRune(text[:len(text)-1], '(')) {
							diags = append(diags, Diagnostic{
								Path: f.Path, Line: line, Rule: "link-punctuation",
								Message: fmt.Sprintf("link includes trailing %q", string(last)),
							})
						}
					}
				}
			}
		}

		raw = strings.TrimSpace(raw)
		if raw == "" {
			return ast.WalkContinue, nil
		}
		diags = append(diags, validateLinkURL(f.Path, line, raw, siteHosts)...)
		return ast.WalkContinue, nil
	})
	return diags
}

func validateLinkURL(path string, line int, raw string, siteHosts map[string]bool) []Diagnostic {
	// mailto: validated as email
	if strings.HasPrefix(strings.ToLower(raw), "mailto:") {
		if msg := validateMailto(raw); msg != "" {
			return []Diagnostic{{
				Path: path, Line: line, Rule: "link-host",
				Message: fmt.Sprintf("%s: %s", msg, raw),
			}}
		}
		return nil
	}
	// Protocol-relative links are discouraged.
	if strings.HasPrefix(raw, "//") {
		return []Diagnostic{{
			Path: path, Line: line, Rule: "protocol-relative-url",
			Message: "specify a URL protocol",
		}}
	}
	// Root-relative and fragment links are fine.
	if raw[0] == '/' || raw[0] == '#' {
		return nil
	}
	// Scheme written without a colon (e.g. `https//example.com`).
	if schemeNoColon.MatchString(raw) {
		return []Diagnostic{{
			Path: path, Line: line, Rule: "scheme-missing-colon",
			Message: fmt.Sprintf("scheme missing colon: %s", raw),
		}}
	}
	u, err := urlpkg.Parse(raw)
	if err != nil {
		return []Diagnostic{{
			Path: path, Line: line, Rule: "link-host",
			Message: fmt.Sprintf("unparseable URL: %s", raw),
		}}
	}
	// No scheme — a plain relative path. Hugo content should use root-relative URLs.
	if u.Scheme == "" {
		return []Diagnostic{{
			Path: path, Line: line, Rule: "relative-link",
			Message: fmt.Sprintf("relative link: %s (use root-relative path starting with /)", raw),
		}}
	}
	if skipSchemes[u.Scheme] {
		return nil
	}

	var diags []Diagnostic
	if sm := schemeSeparator.FindStringSubmatch(raw); sm != nil {
		scheme, slashes := sm[1], sm[2]
		if !knownSchemes[scheme] {
			return append(diags, Diagnostic{
				Path: path, Line: line, Rule: "unknown-scheme",
				Message: fmt.Sprintf("unknown or mistyped scheme: %s", raw),
			})
		}
		if len(slashes) != 2 {
			return append(diags, Diagnostic{
				Path: path, Line: line, Rule: "malformed-scheme-separator",
				Message: fmt.Sprintf("malformed scheme separator: %s", raw),
			})
		}
	}

	if u.Scheme == "http" {
		diags = append(diags, Diagnostic{
			Path: path, Line: line, Rule: "http-url",
			Message: fmt.Sprintf("http:// URL: %s", raw),
		})
	}
	if bad := unsafeURLChars(raw); len(bad) > 0 {
		diags = append(diags, Diagnostic{
			Path: path, Line: line, Rule: "url-chars",
			Message: fmt.Sprintf("URL contains unencoded characters %s (percent-encode them): %s",
				quoteRunes(bad), raw),
		})
	}
	if siteHosts[u.Host] {
		diags = append(diags, Diagnostic{
			Path: path, Line: line, Rule: "site-local-url",
			Message: fmt.Sprintf("site-local absolute URL: %s (use %s)", raw, rootRelativeURL(u)),
		})
	}
	if msg := validateHost(u.Host); msg != "" {
		diags = append(diags, Diagnostic{
			Path: path, Line: line, Rule: "link-host",
			Message: fmt.Sprintf("%s: %s", msg, raw),
		})
	}
	return diags
}

// unsafeURLChars returns the unique set of characters in raw that fall
// outside the RFC 3986 unencoded-safe set (in first-seen order).
func unsafeURLChars(raw string) []rune {
	if urlSafeChar.MatchString(raw) {
		return nil
	}
	var bad []rune
	seen := map[rune]bool{}
	for _, r := range raw {
		if r < 0x80 && urlSafeChar.MatchString(string(r)) {
			continue
		}
		if seen[r] {
			continue
		}
		seen[r] = true
		bad = append(bad, r)
	}
	return bad
}

func quoteRunes(rs []rune) string {
	parts := make([]string, len(rs))
	for i, r := range rs {
		parts[i] = fmt.Sprintf("%q", string(r))
	}
	return strings.Join(parts, ", ")
}

// nodeText returns the concatenated text content of a node's descendants
// (used to inspect link text and image alt text).
func nodeText(n ast.Node, source []byte) string {
	var b strings.Builder
	ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if t, ok := c.(*ast.Text); ok {
			b.Write(t.Segment.Value(source))
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

func validateMailto(raw string) string {
	addr := strings.TrimPrefix(raw, "mailto:")
	if i := strings.IndexAny(addr, "?#"); i >= 0 {
		addr = addr[:i]
	}
	if addr == "" {
		return "empty mailto address"
	}
	at := strings.LastIndex(addr, "@")
	if at < 0 {
		return "mailto missing @"
	}
	local, domain := addr[:at], addr[at+1:]
	if local == "" {
		return "mailto empty local part"
	}
	if strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") || strings.Contains(local, "..") {
		return "mailto invalid local part"
	}
	if !emailLocal.MatchString(local) {
		return "mailto invalid characters in local part"
	}
	if msg := validateHost(domain); msg != "" {
		return "mailto " + msg
	}
	return ""
}

func validateHost(host string) string {
	if i := strings.LastIndex(host, ":"); i >= 0 && !strings.Contains(host[i:], "]") {
		host = host[:i]
	}
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	if host == "" {
		return "empty host"
	}
	if !hostAllowed.MatchString(host) {
		return "invalid characters in host"
	}
	if !strings.Contains(host, ".") {
		return "host has no dot"
	}
	if strings.Contains(host, "..") {
		return "consecutive dots in host"
	}
	for _, label := range strings.Split(host, ".") {
		if label == "" {
			return "empty label in host"
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return "label starts or ends with hyphen"
		}
	}
	return ""
}

func configuredSiteHosts(ctx *MarkdownContext) map[string]bool {
	if ctx == nil || ctx.Config == nil || len(ctx.Config.Links.SiteHosts) == 0 {
		return nil
	}
	hosts := make(map[string]bool, len(ctx.Config.Links.SiteHosts))
	for _, host := range ctx.Config.Links.SiteHosts {
		if host != "" {
			hosts[host] = true
		}
	}
	return hosts
}

func rootRelativeURL(u *urlpkg.URL) string {
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	if u.Fragment != "" {
		path += "#" + u.Fragment
	}
	return path
}
