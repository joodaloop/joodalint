The real bottlenecks are not the small rule checks. They’re the structural passes andexternal processes.

Biggest bottlenecks

1. tidy is the dominant cost in build.
    In internal/runner/tidy.go:18, the linter walks the build tree and then spawns one
    tidy process per HTML file at internal/runner/tidy.go:45. Process startup plus
    full HTML validation per page will dwarf the in-process string checks once the
    site is nontrivial.
2. Markdown spellcheck is the dominant cost in md when enabled.
    In internal/rules/markdown_spelling.go:57, every Markdown file launches a separate
    aspell process. After that, it scans the file again to recover line numbers at
    internal/rules/markdown_spelling.go:82. That is much more expensive than the pure-
    Go rules.
3. Repeated AST walking is the main in-process Markdown cost.
    Each Markdown file is parsed once in internal/runner/runner.go:63, which is good,
    but then the AST is walked separately by each AST rule:
    - headings: internal/rules/markdown_headings_ast.go:20
    - URLs: internal/rules/markdown_urls.go:75
    - image alt: internal/rules/markdown_urls.go:40
    - formatting: internal/rules/markdown_formatting.go:23
4. prose-hygiene is the heaviest pure-Go text rule.
    It scans every line and runs a large number of regexes/string checks per line in
    internal/rules/markdown_prose_hygiene.go:99. It is still cheaper than aspell, but
    among the in-process text rules it is the most expensive by far.
5. HTML loading is a necessary full parse, but not the main problem.
    build parses every HTML file once in internal/runner/runner.go:151 and tokenizes
    it in internal/runner/runner.go:178. That cost is real, but it’s mostly justified
    and likely still below tidy.

Repeated work

- Frontmatter YAML is parsed twice per Markdown file.
  It is parsed once when building FrontmatterFile at internal/runner/runner.go:53,
  via internal/rules/markdown_frontmatter.go:145, and then parsed again inside the
  rule at internal/rules/markdown_frontmatter.go:37. The second parse is redundant.
- Node line lookup is repeatedly expensive.
  NodeLine walks node descendants in internal/rules/rules.go:36, and
  earliestTextStart itself does an ast.Walk at internal/rules/rules.go:50. Then
  LineAt counts newlines from the start of the file every time at internal/rules/
  rules.go:26. For many diagnostics, line lookup is effectively O(node subtree +
  bytes before node), repeated over and over.
- nodeText rewalks subtrees repeatedly.
  Used in URL checks and formatting checks at internal/rules/markdown_urls.go:104,
  internal/rules/markdown_urls.go:50, and internal/rules/markdown_formatting.go:34.
  That means extra subtree traversals on top of the full AST walks.
- HTML link parsing/resolution logic is repeated across rules.
  relative-link-exists, fragment-link-exists, image-src-exists, asset-src-exists, CSS
  scanning, and metadata asset checks all re-parse or re-resolve URLs separately:
    - internal/rules/html_links.go:18
    - internal/rules/html_fragments.go:17
    - internal/rules/html_images.go:16
    - internal/rules/html_assets.go:16
    - internal/rules/html_orphans.go:70
    - internal/rules/html_metadata.go:149
- isRelative and resolve both parse the same URL.
  Example: internal/rules/html_links.go:21 calls isRelative, which does url.Parse at
  line 56, and then resolve, which does another url.Parse at line 64.
- loadHTML computes URL paths more than once per file.
  It builds u manually at internal/runner/runner.go:136, then separately calls
  urlPathFor(root, p) for allFiles at line 146 and again for the HTMLFile at line
  158.
- MarkLinked takes a mutex on every discovered link target.
  That is correct for concurrency, but on a large site the hot lock in internal/
  rules/rules.go:115 will add contention because several HTML rules call it
  frequently.

What is probably not a real bottleneck

- Simple contains-based HTML artifact checks in internal/rules/html_artifacts.go:52
- Orphan reporting itself in internal/rules/html_orphans.go:15
- Basic frontmatter field validation logic after parse
- The balance rule, which is just a linear scan of the file body

Bottom line
If you want the biggest speedups, the order is:

1. Reduce or batch tidy
2. Reduce or batch aspell
3. Eliminate repeated AST subtree walks and repeated line-number recomputation
4. Stop reparsing frontmatter YAML
5. Consolidate HTML URL normalization/parsing so each link is parsed once
