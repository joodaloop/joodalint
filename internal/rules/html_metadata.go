package rules

import (
	"fmt"
	"net/url"
	"strings"
)

func init() {
	RegisterHTML(&headMetadata{})
}

type headMetadata struct{}

func (headMetadata) ID() string { return "head-metadata" }

func (headMetadata) Check(f *HTMLFile, ctx *HTMLContext) []Diagnostic {
	var diags []Diagnostic
	add := func(msg string) {
		diags = append(diags, Diagnostic{Path: f.Path, Rule: "head-metadata", Message: msg})
	}

	if f.Title == "" {
		add("missing or empty <title>")
	}

	if strings.TrimSpace(f.Lang) == "" {
		add(`missing <html lang="...">`)
	}

	byName := map[string]string{}
	byProp := map[string]string{}
	var charset string
	var ctCharset string
	for _, m := range f.Metas {
		if m.Charset != "" {
			charset = strings.ToLower(strings.TrimSpace(m.Charset))
		}
		if strings.EqualFold(m.HTTPEquiv, "Content-Type") {
			ctCharset = parseCharsetParam(m.Content)
		}
		if m.Name != "" {
			byName[strings.ToLower(m.Name)] = m.Content
		}
		if m.Property != "" {
			byProp[strings.ToLower(m.Property)] = m.Content
		}
	}

	if charset == "" {
		add(`missing <meta charset="utf-8">`)
	} else if charset != "utf-8" {
		add(fmt.Sprintf("<meta charset=%q> should be utf-8", charset))
	}
	if ctCharset != "" && ctCharset != "utf-8" {
		add(fmt.Sprintf(`<meta http-equiv="Content-Type"> charset=%q should be utf-8`, ctCharset))
	}

	if v := strings.TrimSpace(byName["viewport"]); v == "" {
		add(`missing <meta name="viewport">`)
	} else if !strings.Contains(strings.ReplaceAll(v, " ", ""), "width=device-width") {
		add(fmt.Sprintf(`<meta name="viewport"> should include width=device-width (got %q)`, v))
	}

	requireName := func(key string) string {
		v := strings.TrimSpace(byName[key])
		if v == "" {
			add(fmt.Sprintf(`missing or empty <meta name=%q>`, key))
		}
		return v
	}
	requireProp := func(key string) string {
		v := strings.TrimSpace(byProp[key])
		if v == "" {
			add(fmt.Sprintf(`missing or empty <meta property=%q>`, key))
		}
		return v
	}

	desc := requireName("description")

	ogTitle := requireProp("og:title")
	ogDesc := requireProp("og:description")
	ogURL := requireProp("og:url")
	ogImage := requireProp("og:image")

	twCard := requireName("twitter:card")
	twTitle := requireName("twitter:title")
	twDesc := requireName("twitter:description")
	twImage := requireName("twitter:image")

	if ogTitle != "" && f.Title != "" && ogTitle != f.Title {
		add(fmt.Sprintf("og:title %q does not match <title> %q", ogTitle, f.Title))
	}
	if ogDesc != "" && desc != "" && ogDesc != desc {
		add(fmt.Sprintf("og:description does not match <meta name=\"description\">"))
	}
	if twTitle != "" && f.Title != "" && twTitle != f.Title {
		add(fmt.Sprintf("twitter:title %q does not match <title> %q", twTitle, f.Title))
	}
	if twDesc != "" && desc != "" && twDesc != desc {
		add("twitter:description does not match <meta name=\"description\">")
	}
	if twImage != "" && ogImage != "" && twImage != ogImage {
		add("twitter:image does not match og:image")
	}
	if twCard != "" && twCard != "summary_large_image" && twCard != "summary" && twCard != "app" && twCard != "player" {
		add(fmt.Sprintf("twitter:card has unexpected value %q", twCard))
	}

	if ogURL != "" {
		checkAbsoluteURL(ctx, f, "og:url", ogURL, false, add)
		expected := canonicalURLPath(f.URLPath)
		if got, ok := pathOfAbsoluteURL(ogURL, ctx); ok && got != expected {
			add(fmt.Sprintf("og:url path %q does not match page %q", got, expected))
		}
	}
	if ogImage != "" {
		checkAbsoluteURL(ctx, f, "og:image", ogImage, true, add)
	}
	if twImage != "" {
		checkAbsoluteURL(ctx, f, "twitter:image", twImage, true, add)
	}

	var altMD *HeadLink
	for i := range f.HeadLinks {
		l := f.HeadLinks[i]
		if strings.EqualFold(l.Rel, "alternate") && strings.EqualFold(l.Type, "text/markdown") {
			altMD = &f.HeadLinks[i]
			break
		}
	}
	if altMD == nil {
		add(`missing <link rel="alternate" type="text/markdown">`)
	} else if strings.TrimSpace(altMD.Href) == "" {
		add(`<link rel="alternate" type="text/markdown"> has empty href`)
	} else if isRelative(altMD.Href) {
		resolved, ok := resolve(f.URLPath, altMD.Href)
		if ok {
			ctx.MarkLinked(resolved)
			if !ctx.Pages[resolved] {
				add(fmt.Sprintf(`<link rel="alternate" type="text/markdown"> href %q resolves to %s which does not exist`, altMD.Href, resolved))
			}
		}
	}

	return diags
}

func parseCharsetParam(content string) string {
	for _, part := range strings.Split(content, ";") {
		part = strings.TrimSpace(part)
		if eq := strings.IndexByte(part, '='); eq >= 0 && strings.EqualFold(strings.TrimSpace(part[:eq]), "charset") {
			return strings.ToLower(strings.TrimSpace(part[eq+1:]))
		}
	}
	return ""
}

func canonicalURLPath(urlPath string) string {
	if strings.HasSuffix(urlPath, "/index.html") {
		return strings.TrimSuffix(urlPath, "index.html")
	}
	return urlPath
}

func siteHosts(ctx *HTMLContext) []string {
	if ctx == nil || ctx.Config == nil {
		return nil
	}
	return ctx.Config.Links.SiteHosts
}

func checkAbsoluteURL(ctx *HTMLContext, f *HTMLFile, label, raw string, allowAsset bool, add func(string)) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		add(fmt.Sprintf("%s %q must be an absolute URL", label, raw))
		return
	}
	hosts := siteHosts(ctx)
	if len(hosts) == 0 {
		return
	}
	matched := false
	for _, h := range hosts {
		if strings.EqualFold(u.Host, h) {
			matched = true
			break
		}
	}
	if !matched {
		add(fmt.Sprintf("%s host %q is not in configured site_hosts", label, u.Host))
		return
	}
	if !allowAsset {
		return
	}
	resolved, ok := resolve(f.URLPath, u.Path)
	if !ok {
		return
	}
	ctx.MarkLinked(resolved)
	if !ctx.Pages[resolved] {
		add(fmt.Sprintf("%s %q resolves to %s which does not exist", label, raw, resolved))
	}
}

func pathOfAbsoluteURL(raw string, ctx *HTMLContext) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "", false
	}
	hosts := siteHosts(ctx)
	if len(hosts) > 0 {
		matched := false
		for _, h := range hosts {
			if strings.EqualFold(u.Host, h) {
				matched = true
				break
			}
		}
		if !matched {
			return "", false
		}
	}
	p := u.Path
	if p == "" {
		p = "/"
	}
	return p, true
}
