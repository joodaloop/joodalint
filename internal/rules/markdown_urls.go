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
}

type markdownURLs struct{}

func (markdownURLs) ID() string { return "url" }

var (
	schemeNoColon   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*//`)
	schemeSeparator = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9+.\-]*):(/+)`)
	hostAllowed     = regexp.MustCompile(`^[a-zA-Z0-9.\-]+$`)
	emailLocal      = regexp.MustCompile(`^[A-Za-z0-9!#$%&'*+/=?^_` + "`" + `{|}~.-]+$`)
	knownSchemes    = map[string]bool{"http": true, "https": true}
	skipSchemes     = map[string]bool{"tel": true, "javascript": true, "data": true}
)

func (markdownURLs) Check(f *MarkdownFile, ctx *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
	siteHosts := configuredSiteHosts(ctx)
	ast.Walk(f.AST, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		raw, ok := urlForValidation(n, f.Body)
		if !ok {
			return ast.WalkContinue, nil
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return ast.WalkContinue, nil
		}
		diags = append(diags, validateLinkURL(f.Path, f.NodeLine(n), raw, siteHosts)...)
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

// urlForValidation returns the destination URL for a Link, Image, or
// AutoLink node. AutoLink emails are returned with their "mailto:" prefix.
func urlForValidation(n ast.Node, source []byte) (string, bool) {
	switch v := n.(type) {
	case *ast.Link:
		return string(v.Destination), true
	case *ast.Image:
		return string(v.Destination), true
	case *ast.AutoLink:
		return string(v.URL(source)), true
	}
	return "", false
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
