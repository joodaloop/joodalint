# joodalint
the sanest linter in the world

## CONFIGURATION
Put this in the folder from where you run `joodalint md` (Markdown prose lint), `joodalint build` (site build output checks), or `joodalint help` (general website assistance).

(Note: You don't have to specify a title in your schemas, our linting requires *all* your pages have one)

```yaml
paths:
  markdown_root: content # folder containing your content .md files
  build_root: public # folder that will contain the built site
  markdown_skip: [drafts, changelog.md, "notes/*", "*.wip.md"] # paths the `md`/`help` commands skip
  build_skip: [vendor, 404.html] # paths the `build` command skips
  # skip lists use .gitignore syntax, rooted at the content/build folder:
  #   drafts        skip the whole drafts/ subtree, wherever it appears
  #   notes/*       skip everything under the top-level notes/ folder
  #   *.wip.md      skip any file with this name, at any depth
  #   changelog.md  skip that file anywhere

links:
  site_hosts: [joodaloop.com, www.joodaloop.com] # your site's URLs

spelling:
  dict: ./spelling-dict.txt # your spellcheck dictionary, one word per line

rules:
  disable: [typography, number-format] # checks to turn off, globally
  # Names are the tags in the middle column of lint output, so you can copy
  # one straight from a message you want to stop seeing. Names that don't
  # match a rule are ignored, so a typo means the check keeps running.
  
# frontmatter schema for each section of your site
sections:
  root:
    description: { type: string, required: true }
    date: { type: date }
    type: { type: enum, values: [list, meta] }
    topics: { type: list, items: enum, values: [design, misc, personal, practical, software, websites] }
    lastmod: { type: date }
    layout: { type: enum, values: [workbench, "~"] }
    popular: { type: bool }
    aliases: { type: list }

  writing:
    description: { type: string, required: true }
    date: { type: date, required: true }
    type: { type: enum, required: true, values: [essay, list, notebook] }
    topics: { type: list, required: true, items: enum, values: [design, misc, personal, practical, software, websites] }
    lastmod: { type: date }
    last_update: { type: date }
    popular: { type: bool }

  riffs:
    date: { type: date, required: true }

# frontmatter schema for the _index.md pages in each section
index_pages:
  root:
    type: { type: enum, values: [list] }
```

## TURNING OFF CHECKS

To stop running a lint rule, copy its tag into `rules.disable`:

```yaml
rules:
  disable: [typography, quotes]
```

### Available tags

| Tag | What it covers |
|---|---|
| `invisible-chars` | Zero-width characters, BOMs, Windows-1252 mojibake |
| `md-syntax` | Broken markdown: bullet/blockquote/heading spacing, horizontal rules, setext headers |
| `emphasis` | `* text *`, `_underscore_`, unescaped `**` `~~` `==` `__` |
| `link-style` | Reference links, reversed `()[]` syntax, protocol-relative links |
| `shortcode` | `{{<shortcode>}}` missing its required spaces |
| `repeated-word` | "the the" |
| `spacing` | Space before comma/period/paren, spaced colon, asymmetric `/` and `-` |
| `quotes` | Orphaned, padded, unbalanced, smart quotes, `5'9"` primes |
| `typography` | Em/en dash misuse, malformed plus-minus |
| `number-format` | Currency and percent spacing, hyphen as minus, numeric ranges |
| `balance` | Unmatched brackets, parens, braces |
| `formatting` | Suspiciously long emphasis or code spans |
| `headings` | H1s, headings deeper than h4 |
| `frontmatter` | Schema violations |
| `spelling` | Spellcheck against your dictionary |
| `image-alt` | Missing or meaningless alt text |

URL checks are individually tagged: `empty-url`, `http-url`, `link-host`,
`link-punctuation`, `long-link-text`, `malformed-scheme-separator`,
`protocol-relative-url`, `relative-link`, `scheme-missing-colon`,
`site-local-url`, `spaces-around-link`, `unknown-scheme`, `url-chars`.

Build checks (`joodalint build`): `asset-src-exists`, `duplicate-id`,
`fragment-link-exists`, `head-metadata`, `image-format`, `image-size`,
`image-src-exists`, `orphan-file`, `relative-link-exists`,
`rendered-artifacts`.


## WHAT DOES IT DO?

### Helps you use your personal site better
- [x] Point out lack of posts 
  - [ ] And suggest pages that could be added instead
- [ ] lint for llm phrases with a shaming message
- [x] Graph frequency of publishing
- [x] The linter scans the _drafts directory. If there are more drafts than published posts, it points you to the most written ones.
- [ ] Print out word count graph too (?)

### Frontmatter lint for anything that doesn't match the declared schema
- [x] Always check for title
- [x] Description under 320 characters (Telegram: 340, Whatsapp: 230, Google: 160)
- [x] Warn if fields found that aren't in the config schema
- [x] Spellcheck text fields

### Build lints (`joodalint build`)
- [x] Throw a warning on excessive Javascript
- [x] Build size summary (bytes and file count per category — HTML, CSS, JS, JSON, XML, images, SVG, fonts, video, audio, PDF, wasm, other — with real gzipped sizes for the text formats and an estimated transfer total)
- [x] Flag large images (>500KB) and PNGs (with an estimated size for lossless webp conversion)
- [x] Checks site build for orphan files (not linked to from anywhere)
- [x] Check for presense of essential meta tags
- [x] Check that all internal links point to an existing file (`<a>` href, `<img>` src, `<link>`, `<script src>`, `<video>/<audio>` etc.)
  - [ ]	Check for a valid DOCTYPE, unclosed tags, and correct tag pairing.
  - [x] Validates that IDs are unique across the page
  - [ ] &amp;, &nbsp;, &#39; (unresolved HTML entities bleeding into plain text)
  - [ ] â€™ or Ã© (Mojibake / character encoding failures)
- [x] Detect custom shortcode-like fragments
    - {{<
    -	\>}}
    - {{%
    - %}}
- [x] HTML/comment markers that should be stripped or transformed
    - `<!--`
    - -->
    - <--
    - <— / —> 
    - `<del>`
    - `<q>`
    - `</q>`
    - `</q<`
  
### Markdown elements from AST 
- [x] Warn on H1s (they should be in frontmatter title)
- [x] Warn on any heading more than 4
- [x] Too long link text, code formatting, bold, italic, etc.
- [x] Word repetition like "the the"
- [ ] URLs
  - [ ] https://gwern.net.style-guide
  - [ ] Duplicate trailing slashes, double slashes in paths
  - [x] Catch mailto: addresses that aren’t valid email syntax
  - [x] Don't allow http:// 
  - [x] Empty URLs or empty/meaningless URL text/alt
  - [x] Don't allow relative links
  - [x] Catch non-URL-safe characters inside URL
  - [x] Discourage protocol-relative link
  - [x] Discourage spacing [ text ] in URL text
  - [x] Discourage and punctuation [documentation.](https://example.com) in URL text 

### AST-prose lints
- [x] Existence of ** \`~~ ==
- [x] Spellcheck on prose with aspell with an personal dictionary
  - [x] Special case for username checks (@loquitur_ponte)
- [x] Suffix handling (2nd, 50kg vs 50 kg)
- [x] Unbalanced parens and quotes
- [x] Repetitions
  - word.Word (missing space after punctuation)
  - -10 (hyphen) vs −10 (the actual, slightly wider minus sign character).
  - " : "— spaced colon
  - 10 % (unnecessary space before a percent sign)
  - $ ($ £ € ¥) 100 — space between currency symbol and number)
  - #1 vs # 1 (inconsistent spacing with the hash/number sign)
  - —— (double em dash)
  - ——– (em dash + en dash)
  - ————– (quadruple em + en)
  - --- (literal triple hyphen)
  - – 10 (space between hyphen/en-dash and number)
  - " . " — space around period
  - ! — space before exclamation mark
  - ? — space before question mark
  - '' (double apostrophe)
  - ,, (double commas)
  - ..  (double period)
  - `` (double backtick)
  - ——– variants generally
  -  ) — space before closing paren
  -  , — space before comma
  - " — floating/orphaned quote
  -  +- /  -+ — malformed plus-minus
  - word/ word or word /word (asymmetrical spacing around a forward slash)
  - " word " (padded spaces inside quotation marks)
  - (  — space after opening paren  
  - word- word or word -word (space around hypen)
  - 100-200 (using a standard hyphen instead of an en dash – for numerical ranges)

### Non-AST checks
- [x] Discourage using smart/curvy quotes in content directly
- [ ] URLs
  - [ ] (http
  - [ ] )http
  - [ ] [http
  - [ ] ]http
  - [ ] Duplicate trailing slashes, double slashes in paths
  - [ ] [Text](https://example.com “Title”)
  - [ ] [text](non-URL character)
  - [ ] [text](url with space)
  - [ ] [text] (url)
  - [ ] [text](url "title)
  - [ ] Discourage bare URLs in prose, they could start with either \n, " ", or "( but not []()"
  - [ ] [text] (/url)
  - [ ] ![alt(image.png)
  - [x] Reversed link syntax ()[]
- [x] Discourage reference links
- [x] Discourage Setext headings 
- [x] __ underscore emphasis detection
- [x] Broken Markdown
  - [x] Headings must start at the beginning of the line
  - [x] Lack of space after # on a new line
  - [x] Horizontal rule failures ( -- on new lines)
  - [x] Triple-star `***word*` — ambiguous, often not what the author wanted.
  - [x] Warn on lack of space after > on new lines
  - [x] Warn on lack of space after - on new lines
  - [x] Spaces inside emphasis markers (** text **)
  - [x] Odd number of spaces/tabs for lists
- [x] Invisible characters
- [x] `{{<shortcode>}}` without the required spaces `{{< shortcode >}}`
- [x] Doubled / malformed punctuation & dashes & suspicious spacing
  - 5'9" (using straight quotes) instead of 5′ 9″ (proper prime).
  - " ]( quote glued to link

## NEXT UP
- [ ] Improve the CLI output format
- [ ] Read https://gwern.net/style-guide#terminology-and-notation
- [ ] Add "markdown syntax found in HTML" lints (should it also catch "almost Markdown"?)
- [ ] Come up with ideas for `joodalint help` commands
  - [ ] Recommend better slugs
- [ ] Frontmatter checks
  - [ ] exactly one metadata block;
  - [ ] field order canonical;
  - [ ] no duplicate fields;
  - [ ] mandatory fields present;
  - [ ] title ≤13 words;
  - [ ] description 20–650 chars unless exempt;
  - [ ] dates after 2008-01-01 and before 'tomorrow';
  - [ ] modified ≥ created;
  - [ ] status, confidence, css-extension are enums;
  - [ ] importance: 0–10;
  - [ ] thumbnail-css only if thumbnail;
  - [ ] thumbnail exists and is not SVG for social-preview contexts;
  - [ ] thumbnail-text optional for new thumbnails, but if exists, can be compiled as Gwerndown.
  - [ ] css-extension should be a list of known page classes, not a free string.
- [ ] Symbols next to other symbols
  - [ ] > is ?

## KNOWN ISSUES

### MDX: escaped delimiters are masked anyway
`\{` and `\<` are the MDX escapes for literal delimiters, but `MaskMDX`
dispatches on `{` and `<` without checking for a preceding backslash, so
escaped content is painted over and never linted:

```
in:   Use \{bad  prose ?} here.
out:  Use \               here.
```

The double space and the space before `?` would both normally be flagged.
Fix: count consecutive preceding backslashes and skip masking when the
count is odd. Note `\\{` is an escaped backslash followed by a real
delimiter and must still mask, so a simple `b[i-1] == '\\'` test is wrong.

### MDX: four-space indentation is treated as code
MDX deliberately disables indented code blocks so that components can be
indented freely ([mdx-js/mdx#993](https://github.com/mdx-js/mdx/issues/993)).
Markdown nested inside a component may therefore legally start at four or
more spaces. Masking preserves that indentation and hands it to goldmark,
which is a CommonMark parser and classifies those lines as code — so they
are silently dropped from prose checks. The line-based pass separately
flags such headings via `headingIndented`, which is also wrong for MDX.

Do **not** fix this by dedenting: indentation still carries meaning for
nested lists and blockquotes in MDX, and flattening every line to three
spaces would collapse nested lists into siblings. The correct fix is to
build the MDX parser without goldmark's indented-code block parser
(`parser.WithBlockParsers`, the default set minus `NewCodeBlockParser()`),
leaving every other block parser intact, and to gate `headingIndented` on
an `IsMDX` flag.

### MDX page bundles report false relative-link errors
`isBundleIndex` in `markdown_urls.go` matches only `index.md` and
`_index.md`, so `index.mdx` / `_index.mdx` bundles fail the check and their
valid sibling resource links are reported as `relative-link`. Fix by
comparing the extension-stripped stem and reusing `config.IsMarkdownPath`,
so it can't drift from the same fix already applied to `SchemaFor`.

### Two alt-text tags are undocumented
`empty-image-alt` and `empty-link-text` are emitted from
`markdown_urls.go` but appear nowhere in the tag list above, so users can
see them in output and fail to disable them. `image-alt` should also be
described as covering generic or meaningless alt text rather than missing
alt text.

These were missed because the tag list was verified by grepping for
`Rule: "literal"`, and both are assigned to a variable before use. Any
future audit should collect `d.Rule` from real rule output instead of
matching on source shape.

### Empty headings report the wrong line
A heading with no inline content (`# ` on its own) is reported at the first
line of the body instead of its real line. With 3-line frontmatter an empty
`#` on line 284 reports as line 4; with 6-line frontmatter it reports as 7.
Non-empty headings in the same position are correct.

Cause: goldmark records **no** line segments for an empty ATX heading
(`h.Lines().Len() == 0`) and it has no text child, so `NodeLine` falls
through both lookups to its `return f.BodyStartLine` fallback. The position
isn't being miscalculated — it was never in the AST to begin with.

So "read the heading node's own position" is not a fix; there is nothing to
read. Recovering the position from the previous sibling doesn't work either:
two empty headings in a row both resolve to the same line, because the
second one's previous sibling is also position-less.

Planned fix: `ast.Walk` visits nodes in document order, so the heading rule
can carry a running byte offset — use a node's real position when it has
one, and when it doesn't, scan `Body` forward from that mark for the next
`^ {0,3}#{1,6}[ \t]*$` line, then advance the mark past it. Consecutive
empty headings then resolve correctly. The scan needs the same fenced-code
tracking the line-based pass already does, or a `#` inside a code fence can
be mistaken for the heading.

### MDX support is unvalidated against real content
`MaskMDX` has unit tests and one end-to-end fixture, but both cover only
the constructs that were thought of while writing it. It has never run
against a real MDX site.

The failure mode to watch for is prose going **missing**, not extra noise.
Extra diagnostics are obvious and self-correcting; a mask that overreaches
silently stops checking sentences and nothing reports it. If a run over an
MDX-heavy directory comes back suspiciously clean, that is the signal. Both
MDX issues above are instances of exactly this.

### Rule `ID()` is dead, and disagrees with the tags
Every rule implements `ID()`, but it is called nowhere in production — only
by tests that assert `ID() == "some literal"`, which test nothing but
themselves. The real user-facing vocabulary is `Diagnostic.Rule`, which is
what gets sorted, printed, and matched by `rules.disable`.

The two have drifted: `markdownURLs.ID()` returns `"url"` while the rule
emits thirteen different tags, and both prose rules still return
`"prose-hygiene"` after being split into ten. Either drop `ID()` from the
interfaces or redefine it as a group name that tags share a prefix with —
but it should not stay as a third vocabulary nobody reads.

### Unknown names in `rules.disable` are ignored
A misspelled rule name silently does nothing, so a check you believe is off
keeps running. This was a deliberate choice — it keeps configs valid across
renames — but it means "the rule won't turn off" almost always means a
typo. Validating with a suggestion was built and removed as more machinery
than the problem warranted; revisit if this bites in practice.

### Table delimiter rows are matched by the `typography` triple-hyphen check.

### Misc. 
- rules.disable filters output instead of gating the tests themselves
- MDX fence detection can desynchronize on fence-like lines with trailing text.
- Namespaced JSX such as <svg:rect> is not masked.
- joodalint help counts raw MDX syntax as draft words.
- No installation/fallback documentation for aspell or tidy-html5.
- Frontmatter schema types and constraints lack a reference.
