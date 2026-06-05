package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Paths         Paths                           `yaml:"paths"`
	Sections      map[string]map[string]FieldSpec `yaml:"sections"`
	IndexSections map[string]map[string]FieldSpec `yaml:"index_pages"`
	Links         Links                           `yaml:"links"`
	Spelling      Spelling                        `yaml:"spelling"`

	mdMatcher    *ignore.GitIgnore
	buildMatcher *ignore.GitIgnore
}

type Links struct {
	SiteHosts []string `yaml:"site_hosts"`
}

type Spelling struct {
	Dict string `yaml:"dict"`
}

type Paths struct {
	MarkdownRoot string   `yaml:"markdown_root"`
	BuildRoot    string   `yaml:"build_root"`
	MarkdownSkip []string `yaml:"markdown_skip"`
	BuildSkip    []string `yaml:"build_skip"`
}

type FieldSpec struct {
	Type     string   `yaml:"type"`
	Required bool     `yaml:"required"`
	Values   []string `yaml:"values"`
	Items    string   `yaml:"items"`
	Min      any      `yaml:"min"`
	Max      any      `yaml:"max"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.Paths.MarkdownRoot == "" {
		c.Paths.MarkdownRoot = "content"
	}
	if c.Paths.BuildRoot == "" {
		c.Paths.BuildRoot = "public"
	}
	return &c, nil
}

// SchemaFor returns the schema to apply to a markdown file given its path.
// For files named _index.md it consults index_pages; otherwise sections.
// Section match is longest-prefix relative to MarkdownRoot. Files directly
// under MarkdownRoot use the special section key "root".
func (c *Config) SchemaFor(filePath string) (string, map[string]FieldSpec) {
	table := c.Sections
	if filepath.Base(filePath) == "_index.md" {
		table = c.IndexSections
	}
	rel, err := filepath.Rel(c.Paths.MarkdownRoot, filePath)
	if err != nil {
		return "", nil
	}
	rel = filepath.ToSlash(rel)
	if !strings.Contains(rel, "/") {
		if schema, ok := table["root"]; ok {
			return "root", schema
		}
	}
	var bestKey string
	for key := range table {
		if key == "root" {
			continue
		}
		if rel == key || strings.HasPrefix(rel, key+"/") {
			if len(key) > len(bestKey) {
				bestKey = key
			}
		}
	}
	if bestKey == "" {
		return "", nil
	}
	return bestKey, table[bestKey]
}

// SkipMarkdown reports whether a path (file or directory) under the markdown
// root should be skipped by the md/help commands. Patterns in markdown_skip use
// .gitignore syntax; see matchSkip.
func (c *Config) SkipMarkdown(root, p string) bool {
	if c.mdMatcher == nil {
		c.mdMatcher = ignore.CompileIgnoreLines(c.Paths.MarkdownSkip...)
	}
	return matchSkip(c.mdMatcher, root, p)
}

// SkipBuild reports whether a path (file or directory) under the build root
// should be skipped by the build command. Patterns in build_skip use .gitignore
// syntax; see matchSkip.
func (c *Config) SkipBuild(root, p string) bool {
	if c.buildMatcher == nil {
		c.buildMatcher = ignore.CompileIgnoreLines(c.Paths.BuildSkip...)
	}
	return matchSkip(c.buildMatcher, root, p)
}

// matchSkip reports whether p matches the gitignore matcher. p is matched as a
// path relative to root, so patterns are interpreted exactly like .gitignore
// entries rooted at the content/build folder: a bare name ("drafts", "*.wip.md")
// matches at any depth, "notes/*" is anchored to the root, and a matched
// directory skips its entire subtree.
func matchSkip(m *ignore.GitIgnore, root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return false
	}
	return m.MatchesPath(rel)
}
