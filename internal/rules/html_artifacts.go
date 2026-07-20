package rules

import (
	"fmt"
	"strings"
)

func init() {
	RegisterHTML(&htmlArtifacts{})
}

// htmlArtifacts flags rendered-output strings that should never survive
// Hugo's pipeline. It scans only visible text (HTMLFile.Text), which already
// excludes <script>, <style>, <pre>, and <code>, so legitimate code samples
// containing these patterns won't trip it.
type htmlArtifacts struct{}

func (htmlArtifacts) ID() string { return "rendered-artifacts" }

// literalPattern is a substring needle reported verbatim. The prose rules
// use taggedPattern instead, which carries the rule tag to report under.
type literalPattern struct {
	needle string
	msg    string
}

var renderedArtifactPatterns = []literalPattern{
	// Unparsed Markdown link/image delimiters leaking as literal text.
	{"(http", "leaked '(http' — likely broken link delimiter"},
	{")http", "leaked ')http' — likely broken link delimiter"},
	{"[http", "leaked '[http' — likely broken link delimiter"},
	{"]http", "leaked ']http' — likely broken link delimiter"},

	// HTML/comment markers that should be stripped or transformed.
	{"<!--", "literal '<!--' in rendered text"},
	{"-->", "literal '-->' in rendered text"},
	{"<--", "literal '<--' in rendered text"},
	{"<—", "literal '<—' in rendered text"},
	{"—>", "literal '—>' in rendered text"},
	{"<q>", "literal '<q>' in rendered text"},
	{"</q>", "literal '</q>' in rendered text"},
	{"</q<", "literal '</q<' in rendered text"},
	{"<del>", "literal '<del>' in rendered text"},

	// Leftover Pandoc/attribute syntax.
	{"**", "literal '**' in rendered text — unparsed bold"},

	// Misc broken inline syntax.
	{"/*", "literal '/*' in rendered text — stray code-comment marker"},
	{"*/", "literal '*/' in rendered text — stray code-comment marker"},

	// Unrendered Hugo shortcode delimiters leaking as text.
	{"{{<", "unrendered Hugo shortcode delimiter '{{<'"},
	{">}}", "unrendered Hugo shortcode delimiter '>}}'"},
	{"{{%", "unrendered Hugo shortcode delimiter '{{%'"},
	{"%}}", "unrendered Hugo shortcode delimiter '%}}'"},
}

func (htmlArtifacts) Check(f *HTMLFile, _ *HTMLContext) []Diagnostic {
	if f.Text == "" {
		return nil
	}
	var diags []Diagnostic
	for _, p := range renderedArtifactPatterns {
		if !strings.Contains(f.Text, p.needle) {
			continue
		}
		diags = append(diags, Diagnostic{
			Path:    f.Path,
			Rule:    "rendered-artifacts",
			Message: fmt.Sprintf("%s: %q", p.msg, p.needle),
		})
	}
	return diags
}
