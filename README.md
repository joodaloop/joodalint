# hugolint
the sanest linter in the world

## CONFIGURATION
Put this in the folder from where you run `hugolint md` or `hugolint build`
```yaml
paths:
  markdown_root: content # folder containing your content .md files
  build_root: public # folder that will contain the built site
  skip_dirs: [drafts] # folders within the `markdown_root` that shouldn't be linted

links:
  site_hosts: [joodaloop.com, www.joodaloop.com] # your site's URLs

spelling:
  dict: ./spelling-dict.txt # your spellcheck dictionary, one word per line
  
# frontmatter schema for each section of your site
sections:
  root:
    title: { type: string, required: true }
    date: { type: date }
    type: { type: enum, values: [list, meta] }
    topics: { type: list, items: enum, values: [design, misc, personal, practical, software, websites] }
    description: { type: string, min: 1, max: 160 }
    lastmod: { type: date }
    layout: { type: enum, values: [workbench, "~"] }
    popular: { type: bool }
    aliases: { type: list }

  writing:
    title: { type: string, required: true }
    date: { type: date, required: true }
    type: { type: enum, required: true, values: [essay, list, notebook] }
    topics: { type: list, required: true, items: enum, values: [design, misc, personal, practical, software, websites] }
    description: { type: string, min: 24, max: 160 }
    lastmod: { type: date }
    last_update: { type: date }
    popular: { type: bool }

  riffs:
    title: { type: string, required: true }
    date: { type: date, required: true }

# frontmatter schema for the _index.md pages in each section
index_pages:
  root:
    title: { type: string, required: true }
    type: { type: enum, values: [list] }
  writing:
    title: { type: string, required: true }
  riffs:
    title: { type: string, required: true }
```

## WHAT DOES IT DO?

### Frontmatter lint for anything that doesn't match the declared schema *in any way*

### Build lints (`hugolint build`)
- [ ] Checks site build for orphan files (not linked to from anywhere)
- [ ] Check for presense of essential meta tags
- [ ] Check that all internal links point to an existing file (`<a>` href, `<img>` src, `<link>`, `<script src>`, `<video>/<audio>` etc.)
- [ ] Run an HTML tidy/validator pass to catch escaping errors and malformed markup
- [ ] Detect custom shortcode-like fragments
    - {{<
    -	\>}}
    - {{%
    - %}}
- [ ] HTML/comment markers that should be stripped or transformed
    - `<!--`
    - -->
    - <--
    - <—
    - —>
    - `<del>`
    - `<q>`
    - `</q>`
    - `</q<`

### Raw body checks
- [ ] Discourage Setext headings
- [ ] Discourage reference links
- [ ] Catch emphasis flanking *foo*bar* parses as <em>foo</em>bar*
- [ ] Discourage using smart quotes in content directly

### With-markdown AST 
- [x] Warn on H1s (they should be in title: )
- [x] Warn on any heading more than 4
- [x] URLs
  - [x] Catch mailto: addresses that aren’t valid email syntax
  - [x] Don't allow http:// 
  - [x] Empty URLs or empty URL text/alt
  - [x] Don't allow relative links
  - [x] Catch non-URL-safe characters inside URL
  - [ ] Don't allow https://mydomain.com
  - [ ] Discourage protocol-relative link
  - [ ] Invisible characters
  - [ ] Dicourage spacing [ text ] and punctuation [documentation.](https://example.com) in URL text 
  - [ ] Too long link text
  - [ ] Too long code formatting

### Post-AST checks
- [ ] Broken Markdown
  - [ ] Headings must start at the beginning of the line
  - [ ] Lack of space after # on a new line
  - [ ] Horizontal rule failures ( --on new lines)
  - [ ] Failed list formatting (2 vs 3 vs 4 spaces)
  - [ ] Triple-star `***word*` — ambiguous, often not what the author wanted.
  - [ ]  \* \_ \# \[ \]
  - [ ] Warn on lack of space after > on new lines
  - [ ] Spaces inside emphasis markers
  - [ ] Odd number of spaces/tabs
- [ ] URLs
  - [ ] Discourage bare URLs in prose
  - [ ] " ]( — quote glued to link
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
  - [ ] ![alt(image.png)
  - [ ] Reversed link syntax ()[]
  - [ ] Check for malformed URLs
  - [ ] URLs with whitespace, smart quotes, or trailing punctuation accidentally included
- [ ] Balancing parens, quotes, formatting (** \`~~) and shortcode delimiters ({{<)
- [ ] `{{<shortcode>}}` without the required spaces
- [ ] Spellcheck on prose with aspell with an personal dictionary
- [ ] Word repetition like "the the"
- [ ] Suffix handling (2nd, 50kg vs 50 kg)
- [ ] Unparsed Markdown link/image delimiters leaking as literal text
- [ ] Doubled / malformed punctuation & dashes
  - [ ] —— (double em dash)
  - [ ] ——– (em dash + en dash)
  - [ ] ————– (quadruple em + en)
  - [ ] --- (literal triple hyphen)
  - [ ] '' (double apostrophe)
  - [ ] ,, (double commas)
  - [ ]   `` (double backtick)
  - [ ] ——– variants generally
- [ ] Suspicious spacing
  - [ ]  ) — space before closing paren
  - [ ] " — floating/orphaned quote
  - [ ] : — spaced colon
  - [ ]  +- /  -+ — malformed plus-minus
