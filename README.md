# HUGOLINT
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

### Build lints (`hugolint build`)
- Check that all relative links lead somewhere (`<a>` href, `<img>` src, `<link>`, `<script src>`, `<video>/<audio>` etc.)
- Run an HTML tidy/validator pass to catch escaping errors and malformed markup
- Detect custom shortcode-like fragments
  - {{<
  -	\>}}
  - {{%
  - %}}
- Unparsed Markdown link/image delimiters leaking as literal text
  - (http
  - )http
  - [http
  - ]http
- HTML/comment markers that should be stripped or transformed
  - `<!--`
  - -->
  - <--
  - <—
  - —>
  - `<del>`
  - `<q>`
  - `</q>`
  - `</q<`

### Markdown lints (`hugolint md`)
- Spelling linting with hunspell or aspell with an dict.txt
- Best practices Markdown
  - Warn on h1s (they should be in title: )
  - Warn on underscore based formatting
  - Headings must start at the beginning of the line
  - Warn on lack of space after #, list markers, and > on new lines
- Frontmatter validity
- URL checks
  - mailto: addresses that aren’t valid email syntax
  - Don't allow http:// 
  - Catch protocol-relative URLs (//example.com) where you meant https://
  - Empty URLs
  - Don't allow relative links
  - Reversed link syntax ()[]
  - Check for malformed URLs
- Balance linting to match parens and quotes
- Spaces inside emphasis markers
- Code fences missing closers, or a language tag
- Image alt text missing in `![](url)`, `![ ](url)`, `![image](url)`, `![img](url) `
- Word repitition like "the the"
- Doubled / malformed punctuation & dashes
  - —— (double em dash)
  - ——– (em dash + en dash)
  - ————– (quadruple em + en)
  - --- (literal triple hyphen)
  - '' (double apostrophe)
  -   `` (double backtick)
  - ——– variants generally
- Suspicious spacing
  -  ) — space before closing paren
  - " — floating/orphaned quote
  - : — spaced colon
  -  +- /  -+ — malformed plus-minus
  - ]() — empty link
  - ![]( — empty image
  - ](// — protocol-relative link
  -  " ]( — quote glued to link

## WHAT DOES IT DO?

- Unbalanced **/backticks
- URLs with whitespace, smart quotes, or trailing punctuation accidentally included
- Duplicate trailing slashes, double slashes in paths
