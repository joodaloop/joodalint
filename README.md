# hugolint
the sanest linter in the world

## CONFIGURATION
Put this in the folder from where you run `hugolint md` or `hugolint build`

(Note: You don't have to specify title or description in your schemas, our linting requires *all* your pages to have those two fields.)

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
    date: { type: date }
    type: { type: enum, values: [list, meta] }
    topics: { type: list, items: enum, values: [design, misc, personal, practical, software, websites] }
    lastmod: { type: date }
    layout: { type: enum, values: [workbench, "~"] }
    popular: { type: bool }
    aliases: { type: list }

  writing:
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

## WHAT DOES IT DO?

### Frontmatter lint for anything that doesn't match the declared schema
- [x] Always check for title and description
- [x] Warn if fields found that aren't in the config schema

### Build lints (`hugolint build`)
- [x] Checks site build for orphan files (not linked to from anywhere)
- [x] Check for presense of essential meta tags
- [x] Check that all internal links point to an existing file (`<a>` href, `<img>` src, `<link>`, `<script src>`, `<video>/<audio>` etc.)
- [ ] Run an HTML tidy/validator pass to catch escaping errors and malformed markup
  - [ ]	Check for a valid DOCTYPE, unclosed tags, and correct tag pairing.
  - [ ] Validates that IDs are unique across the page
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
    - <—
    - —>
    - `<del>`
    - `<q>`
    - `</q>`
    - `</q<`

### With-markdown AST 
- [x] Warn on H1s (they should be in frontmatter title)
- [x] Warn on any heading more than 4
- [x] Too long link text, code formatting, bold, italic, etc.
- [x] URLs
  - [x] Catch mailto: addresses that aren’t valid email syntax
  - [x] Don't allow http:// 
  - [x] Empty URLs or empty URL text/alt
  - [x] Don't allow relative links
  - [x] Catch non-URL-safe characters inside URL
  - [ ] Duplicate trailing slashes, double slashes in paths
  - [x] Discourage protocol-relative link
  - [x] Discourage spacing [ text ] in URL text
  - [x] Discourage and punctuation [documentation.](https://example.com) in URL text 

### AST-prose lints
- [ ] Existence of ** \`~~
- [ ] Spellcheck on prose with aspell with an personal dictionary
- [ ] Suffix handling (2nd, 50kg vs 50 kg)

### Non-AST checks
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
- [x] Word repetition like "the the"
- [x] Unbalanced parens and quotes
- [x] Discourage Setext headings 
- [x] Discourage using smart/curvy quotes in content directly
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
  - —— (double em dash)
  - ——– (em dash + en dash)
  - ————– (quadruple em + en)
  - --- (literal triple hyphen)
  - '' (double apostrophe)
  - ,, (double commas)
  - ..  (double period)
  - `` (double backtick)
  - ——– variants generally
  -  ) — space before closing paren
  -  , — space before comma
  - " — floating/orphaned quote
  - : — spaced colon
  -  +- /  -+ — malformed plus-minus
  - word.Word (missing space after punctuation)
  - word/ word or word /word (asymmetrical spacing around a forward slash)
  - " word " (padded spaces inside quotation marks)
  - 10 % (unnecessary space before a percent sign)
  - $ 100 (space between currency symbol and number)
  - #1 vs # 1 (inconsistent spacing with the hash/number sign)
  - 5'9" (using straight quotes) instead of 5′ 9″ (proper prime).
  - word- word or word -word (space around hypen)
  - -10 (hyphen) vs −10 (the actual, slightly wider minus sign character).
  - 100-200 (using a standard hyphen instead of an en dash – for numerical ranges)
