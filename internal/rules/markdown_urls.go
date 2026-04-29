package rules

import (
	"bufio"
	"bytes"
	"fmt"
	urlpkg "net/url"
	"regexp"
)

func init() {
	RegisterMarkdown(&markdownURLs{})
}

type markdownURLs struct{}

func (markdownURLs) ID() string { return "malformed-url" }

var (
	urlCandidate    = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.\-]*:?/+[^\s<>"'` + "`" + `)\]}]+`)
	trailingPunct   = regexp.MustCompile(`[.,;:!?]+$`)
	schemeNoColon   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*//`)
	schemeSeparator = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9+.\-]*):(/+)`)
	knownSchemes    = map[string]bool{"http": true, "https": true}
)

func (markdownURLs) Check(f *MarkdownFile, ctx *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
	siteHosts := configuredSiteHosts(ctx)
	scanner := bufio.NewScanner(bytes.NewReader(f.Content))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		for _, m := range urlCandidate.FindAllString(scanner.Text(), -1) {
			url := trailingPunct.ReplaceAllString(m, "")

			switch {
			case schemeNoColon.MatchString(url):
				diags = append(diags, Diagnostic{
					Path:    f.Path,
					Line:    line,
					Rule:    "scheme-missing-colon",
					Message: fmt.Sprintf("scheme missing colon: %s", url),
				})
			default:
				sm := schemeSeparator.FindStringSubmatch(url)
				if sm == nil {
					continue
				}
				scheme, slashes := sm[1], sm[2]
				if !knownSchemes[scheme] {
					diags = append(diags, Diagnostic{
						Path:    f.Path,
						Line:    line,
						Rule:    "unknown-scheme",
						Message: fmt.Sprintf("unknown or mistyped scheme: %s", url),
					})
				} else if len(slashes) != 2 {
					diags = append(diags, Diagnostic{
						Path:    f.Path,
						Line:    line,
						Rule:    "malformed-scheme-separator",
						Message: fmt.Sprintf("malformed scheme separator: %s", url),
					})
					continue
				}

				if scheme == "http" {
					diags = append(diags, Diagnostic{
						Path:    f.Path,
						Line:    line,
						Rule:    "http-url",
						Message: fmt.Sprintf("http:// URL: %s", url),
					})
				}

				u, err := urlpkg.Parse(url)
				if err == nil && siteHosts[u.Host] {
					diags = append(diags, Diagnostic{
						Path:    f.Path,
						Line:    line,
						Rule:    "site-local-url",
						Message: fmt.Sprintf("site-local absolute URL: %s (use %s)", url, rootRelativeURL(u)),
					})
				}
			}
		}
	}
	return diags
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
