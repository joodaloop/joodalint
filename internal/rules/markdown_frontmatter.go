package rules

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/joodaloop/hugolint/internal/config"
)

func init() {
	RegisterFrontmatter(&markdownFrontmatter{})
}

type markdownFrontmatter struct{}

func (markdownFrontmatter) ID() string { return "frontmatter" }

func (markdownFrontmatter) Check(f *FrontmatterFile, ctx *FrontmatterContext) []Diagnostic {
	if ctx == nil || ctx.Config == nil {
		return nil
	}
	_, schema := ctx.Config.SchemaFor(f.Path)
	hasFrontmatter := f.Line0 > 0
	fmLine := f.Line0

	if !hasFrontmatter {
		return []Diagnostic{{
			Path: f.Path, Line: 1, Rule: "frontmatter",
			Message: "missing YAML frontmatter",
		}}
	}

	if f.ParseErr != nil {
		return []Diagnostic{{
			Path: f.Path, Line: fmLine, Rule: "frontmatter",
			Message: fmt.Sprintf("invalid YAML: %v", f.ParseErr),
		}}
	}
	parsed := f.Parsed

	var diags []Diagnostic
	diags = append(diags, checkTitleDescription(f.Path, fmLine, parsed, schema)...)

	if schema != nil {
		// Required + per-field validation, in deterministic order.
		keys := make([]string, 0, len(schema))
		for k := range schema {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, name := range keys {
			spec := schema[name]
			val, present := parsed[name]
			if !present {
				if spec.Required {
					diags = append(diags, Diagnostic{
						Path: f.Path, Line: fmLine, Rule: "frontmatter",
						Message: fmt.Sprintf("missing required field %q", name),
					})
				}
				continue
			}
			if msg := validate(name, val, spec); msg != "" {
				diags = append(diags, Diagnostic{
					Path: f.Path, Line: fmLine, Rule: "frontmatter",
					Message: msg,
				})
			}
		}
	}

	diags = append(diags, unknownFieldDiagnostics(f.Path, fmLine, parsed, schema)...)

	return diags
}

const descriptionMaxLen = 160

func checkTitleDescription(path string, line int, parsed map[string]any, schema map[string]config.FieldSpec) []Diagnostic {
	var diags []Diagnostic
	for _, name := range []string{"title", "description"} {
		_, schemaCovers := schema[name]
		val, present := parsed[name]
		if !present {
			if !schemaCovers {
				diags = append(diags, Diagnostic{
					Path: path, Line: line, Rule: "frontmatter",
					Message: fmt.Sprintf("missing required field %q", name),
				})
			}
			continue
		}
		if name == "description" {
			if s, ok := val.(string); ok && len([]rune(s)) > descriptionMaxLen {
				diags = append(diags, Diagnostic{
					Path: path, Line: line, Rule: "frontmatter",
					Message: fmt.Sprintf("field %q: length %d above max %d", name, len([]rune(s)), descriptionMaxLen),
				})
			}
		}
	}
	return diags
}

// SplitFrontmatter parses leading YAML frontmatter and returns the raw YAML
// (without fences), the body after the closing fence, the number of lines
// the frontmatter occupied, and the 1-based line where the frontmatter
// starts. If there is no frontmatter, fmRaw is nil, body == content,
// fmLines == 0, fmStartLine == 0.
func SplitFrontmatter(content []byte) (fmRaw, body []byte, fmLines, fmStartLine int) {
	var prefix int
	switch {
	case bytes.HasPrefix(content, []byte("---\n")):
		prefix = 4
	case bytes.HasPrefix(content, []byte("---\r\n")):
		prefix = 5
	default:
		return nil, content, 0, 0
	}
	rest := content[prefix:]
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return nil, content, 0, 0
	}
	fmRaw = rest[:end]
	after := rest[end+len("\n---"):]
	if i := bytes.IndexByte(after, '\n'); i >= 0 {
		body = after[i+1:]
	} else {
		body = nil
	}
	consumed := len(content) - len(body)
	fmLines = bytes.Count(content[:consumed], []byte("\n"))
	fmStartLine = 1
	return
}

// ParseFrontmatterYAML unmarshals raw frontmatter YAML into a map.
func ParseFrontmatterYAML(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func validate(name string, val any, spec config.FieldSpec) string {
	switch spec.Type {
	case "string":
		s, ok := val.(string)
		if !ok {
			return fmt.Sprintf("field %q: expected string, got %s", name, kindOf(val))
		}
		if spec.Min != nil {
			min, ok := asFloat(spec.Min)
			if !ok {
				return fmt.Sprintf("field %q: invalid min constraint %v for string field", name, spec.Min)
			}
			if len([]rune(s)) < int(min) {
				return fmt.Sprintf("field %q: length %d below min %d", name, len([]rune(s)), int(min))
			}
		}
		if spec.Max != nil {
			max, ok := asFloat(spec.Max)
			if !ok {
				return fmt.Sprintf("field %q: invalid max constraint %v for string field", name, spec.Max)
			}
			if len([]rune(s)) > int(max) {
				return fmt.Sprintf("field %q: length %d above max %d", name, len([]rune(s)), int(max))
			}
		}
	case "number":
		n, ok := asFloat(val)
		if !ok {
			return fmt.Sprintf("field %q: expected number, got %s", name, kindOf(val))
		}
		if spec.Min != nil {
			min, ok := asFloat(spec.Min)
			if !ok {
				return fmt.Sprintf("field %q: invalid min constraint %v for number field", name, spec.Min)
			}
			if n < min {
				return fmt.Sprintf("field %q: %v below min %v", name, n, min)
			}
		}
		if spec.Max != nil {
			max, ok := asFloat(spec.Max)
			if !ok {
				return fmt.Sprintf("field %q: invalid max constraint %v for number field", name, spec.Max)
			}
			if n > max {
				return fmt.Sprintf("field %q: %v above max %v", name, n, max)
			}
		}
	case "bool":
		if _, ok := val.(bool); !ok {
			return fmt.Sprintf("field %q: expected bool, got %s", name, kindOf(val))
		}
	case "date":
		t, ok := parseDate(val)
		if !ok {
			return fmt.Sprintf("field %q: expected date, got %s (%v)", name, kindOf(val), val)
		}
		if spec.Min != nil {
			min, ok := parseDate(spec.Min)
			if !ok {
				return fmt.Sprintf("field %q: invalid min constraint %v for date field", name, spec.Min)
			}
			if t.Before(min) {
				return fmt.Sprintf("field %q: %s before min %s", name, t.Format("2006-01-02"), min.Format("2006-01-02"))
			}
		}
		if spec.Max != nil {
			max, ok := parseDate(spec.Max)
			if !ok {
				return fmt.Sprintf("field %q: invalid max constraint %v for date field", name, spec.Max)
			}
			if t.After(max) {
				return fmt.Sprintf("field %q: %s after max %s", name, t.Format("2006-01-02"), max.Format("2006-01-02"))
			}
		}
	case "enum":
		s, ok := val.(string)
		if !ok {
			return fmt.Sprintf("field %q: expected enum string, got %s", name, kindOf(val))
		}
		if !contains(spec.Values, s) {
			return fmt.Sprintf("field %q: %q not in allowed values %v", name, s, spec.Values)
		}
	case "list":
		items, ok := val.([]any)
		if !ok {
			return fmt.Sprintf("field %q: expected list, got %s", name, kindOf(val))
		}
		for i, it := range items {
			if msg := validateItem(name, i, it, spec); msg != "" {
				return msg
			}
		}
	case "":
		// No type specified — only required-presence is enforced.
	default:
		return fmt.Sprintf("field %q: unknown spec type %q", name, spec.Type)
	}
	return ""
}

func validateItem(name string, i int, val any, spec config.FieldSpec) string {
	switch spec.Items {
	case "enum":
		s, ok := val.(string)
		if !ok {
			return fmt.Sprintf("field %q[%d]: expected enum string, got %s", name, i, kindOf(val))
		}
		if !contains(spec.Values, s) {
			return fmt.Sprintf("field %q[%d]: %q not in allowed values %v", name, i, s, spec.Values)
		}
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Sprintf("field %q[%d]: expected string, got %s", name, i, kindOf(val))
		}
	case "number":
		if _, ok := asFloat(val); !ok {
			return fmt.Sprintf("field %q[%d]: expected number, got %s", name, i, kindOf(val))
		}
	case "date":
		if _, ok := parseDate(val); !ok {
			return fmt.Sprintf("field %q[%d]: expected date, got %s (%v)", name, i, kindOf(val), val)
		}
	case "":
		// No item type specified.
	default:
		return fmt.Sprintf("field %q: unknown items type %q", name, spec.Items)
	}
	return ""
}

var dateLayouts = []string{
	"2006-01-02",
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
}

func isDate(v any) bool {
	_, ok := parseDate(v)
	return ok
}

func kindOf(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return "number"
	case time.Time:
		return "date"
	case []any:
		return "list"
	case map[string]any:
		return "map"
	}
	return fmt.Sprintf("%T", v)
}

func unknownFieldDiagnostics(path string, line int, parsed map[string]any, schema map[string]config.FieldSpec) []Diagnostic {
	unknown := make([]string, 0)
	for name := range parsed {
		if name == "title" || name == "description" {
			continue
		}
		if schema != nil {
			if _, ok := schema[name]; ok {
				continue
			}
		}
		unknown = append(unknown, name)
	}
	sort.Strings(unknown)
	var diags []Diagnostic
	for _, name := range unknown {
		diags = append(diags, Diagnostic{
			Path: path, Line: line, Rule: "frontmatter",
			Message: fmt.Sprintf("unknown field %q", name),
		})
	}
	return diags
}

func parseDate(v any) (time.Time, bool) {
	switch x := v.(type) {
	case time.Time:
		return x, true
	case string:
		for _, layout := range dateLayouts {
			if t, err := time.Parse(layout, x); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int8:
		return float64(x), true
	case int16:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint8:
		return float64(x), true
	case uint16:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	case float32:
		return float64(x), true
	case float64:
		return x, true
	}
	return 0, false
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
