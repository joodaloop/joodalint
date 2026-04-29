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

## WHAT DOES IT DO?

### Build lints (`hugolint build`)
[ ] Check site build for orphans
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


### With-markdown AST 
- Warn on H1s (they should be in title: )
- Warn on any heading more than 4
- URLs
  - mailto: addresses that aren’t valid email syntax
  - Don't allow http:// 
  - Empty URLs or empty URL text
  - Don't allow relative links
  - Smart quotes inside URL
  - [text](url with space)
  - URLs with whitespace, smart quotes, or trailing punctuation accidentally included
  - Image alt text missing in `![](url)`, `![ ](url)`, `![image](url)`, `![img](url) `
  - ](// — protocol-relative link
- Spellcheck on prose with aspell with an personal dictionary

<!--- Walk the AST.
- For link/image/autolink nodes → pull URL from the node, run the URL-validation
function.
- For text nodes (skipping code/link descendants) → run the URL-finding regex on
the text, then run the same URL-validation function on each match.-->

### Post-markdown checks
- Headings must start at the beginning of the line
- Discourage underscore based formatting
- Discourage setext headings and trailing hash headings
- Lack of space after # on a new line
- HOoizontal rule either less or more than 3 characters ( ---, ***, ___)
- Inconsistent indent in nested list (2 vs 3 vs 4 spaces)
- Triple-star `***word*` — ambiguous, often not what the author wanted.
-  \* \_ \# \[ \]
- Emphasis adjacent to alphanumerics: `foo**bar**baz` doesn't render as
  emphasis in CommonMark (flanking rules); a frequent surprise.
- URLs
  - " ]( — quote glued to link
  - Smart quotes inside URL.
  - Duplicate trailing slashes, double slashes in paths
  - [text](non-URL character)
  - [text] (url)
  - [text](url "title)
  - ![alt(image.png)
  - Reversed link syntax ()[]
  - Check for malformed URLs
  - Catch protocol-relative URLs (//example.com) where you meant https://
  - URLs with whitespace, smart quotes, or trailing punctuation accidentally included
- Warn on lack of space list markers, and > on new lines
- Balancing parens, quotes, formatting (** \`~~) and shortcode stuff ({{<)
- `{{<shortcode>}}` without the required spaces
- Spaces inside emphasis markers

### Correctness
- Word repitition like "the the"
- Frontmatter lint with strict warnings for anything that doesn't match the declared schema *in any way*
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
