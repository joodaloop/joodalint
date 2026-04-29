package rules

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

func init() {
	RegisterMarkdown(&markdownLinkHosts{})
}

type markdownLinkHosts struct{}

func (markdownLinkHosts) ID() string { return "link-host" }

var (
	mdLink      = regexp.MustCompile(`\]\(([^)]+)\)`)
	hostAllowed = regexp.MustCompile(`^[a-zA-Z0-9.\-]+$`)
	emailLocal  = regexp.MustCompile(`^[A-Za-z0-9!#$%&'*+/=?^_` + "`" + `{|}~.-]+$`)
	skipSchemes = map[string]bool{"tel": true, "javascript": true, "data": true}
)

func (markdownLinkHosts) Check(f *MarkdownFile, _ *MarkdownContext) []Diagnostic {
	var diags []Diagnostic
	scanner := bufio.NewScanner(bytes.NewReader(f.Content))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		for _, m := range mdLink.FindAllStringSubmatch(scanner.Text(), -1) {
			inner := strings.TrimSpace(m[1])
			if strings.HasPrefix(strings.ToLower(inner), "mailto:") {
				if msg := validateMailto(stripLinkTitle(inner)); msg != "" {
					diags = append(diags, Diagnostic{
						Path: f.Path, Line: line, Rule: "link-host",
						Message: fmt.Sprintf("%s: %s", msg, inner),
					})
				}
				continue
			}
			raw := stripTitle(m[1])
			if raw == "" || raw[0] == '/' || raw[0] == '#' {
				continue
			}
			u, err := url.Parse(raw)
			if err != nil {
				diags = append(diags, Diagnostic{
					Path: f.Path, Line: line, Rule: "link-host",
					Message: fmt.Sprintf("unparseable URL: %s", raw),
				})
				continue
			}
			if u.Scheme == "" || skipSchemes[u.Scheme] {
				continue
			}
			if msg := validateHost(u.Host); msg != "" {
				diags = append(diags, Diagnostic{
					Path: f.Path, Line: line, Rule: "link-host",
					Message: fmt.Sprintf("%s: %s", msg, raw),
				})
			}
		}
	}
	return diags
}

func stripTitle(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}

// stripLinkTitle removes a trailing markdown link title (` "..."` or ` '...'`)
// without splitting at internal whitespace, so spaces inside the URL itself
// remain visible to validators.
func stripLinkTitle(s string) string {
	for _, sep := range []string{` "`, ` '`, "\t\"", "\t'"} {
		if i := strings.Index(s, sep); i >= 0 {
			s = s[:i]
		}
	}
	return strings.TrimSpace(s)
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
