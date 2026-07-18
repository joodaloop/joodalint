# joodalint
the sanest linter in the world

## CONFIGURATION
Put this in the folder from where you run `joodalint md` (Markdown prose lint), `joodalint build` (site build output checks), or `joodaloop help` (general website assistance).

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
- [x] Build size summary (bytes and file count per category — HTML, CSS, JS, JSON, XML, images, fonts, video, audio, PDF, wasm, other — with real gzipped sizes for the text formats)
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
