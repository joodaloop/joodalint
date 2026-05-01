package rules

import (
	"fmt"
	"sort"
	"sync"

	"github.com/joodaloop/hugolint/internal/config"
)

func init() {
	RegisterFrontmatter(&frontmatterSpelling{})
}

type frontmatterSpelling struct {
	errOnce sync.Once
}

func (*frontmatterSpelling) ID() string { return "spelling" }

func (r *frontmatterSpelling) Check(f *FrontmatterFile, ctx *FrontmatterContext) []Diagnostic {
	if ctx == nil || ctx.Config == nil || ctx.Config.Spelling.Dict == "" {
		return nil
	}
	if f.ParseErr != nil || len(f.Parsed) == 0 {
		return nil
	}

	_, schema := ctx.Config.SchemaFor(f.Path)
	fields := textFields(f.Parsed, schema)
	if len(fields) == 0 {
		return nil
	}

	sharedSpeller.ensureInit(ctx.Config)
	if sharedSpeller.initErr != nil {
		var d []Diagnostic
		r.errOnce.Do(func() {
			d = []Diagnostic{{Path: f.Path, Rule: "spelling", Message: sharedSpeller.initErr.Error()}}
		})
		return d
	}
	if !sharedSpeller.enabled {
		return nil
	}

	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	sort.Strings(names)

	var diags []Diagnostic
	for _, name := range names {
		val := fields[name]
		unknown, err := sharedSpeller.unknown([]byte(val), "")
		if err != nil {
			diags = append(diags, Diagnostic{Path: f.Path, Line: f.Line0, Rule: "spelling", Message: err.Error()})
			continue
		}
		if len(unknown) == 0 {
			continue
		}
		words := make([]string, 0, len(unknown))
		for w := range unknown {
			words = append(words, w)
		}
		sort.Strings(words)
		for _, w := range words {
			diags = append(diags, Diagnostic{
				Path: f.Path, Line: f.Line0, Rule: "spelling",
				Message: fmt.Sprintf("unknown word in %q: %q", name, w),
			})
		}
	}
	return diags
}

// textFields returns the string-valued frontmatter fields whose schema type
// is "text". title and description are always included since they're prose.
func textFields(parsed map[string]any, schema map[string]config.FieldSpec) map[string]string {
	out := map[string]string{}
	add := func(name string) {
		if v, ok := parsed[name].(string); ok && v != "" {
			out[name] = v
		}
	}
	add("title")
	add("description")
	for name, spec := range schema {
		if spec.Type == "text" {
			add(name)
		}
	}
	return out
}
