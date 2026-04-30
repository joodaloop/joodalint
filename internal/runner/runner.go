package runner

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"golang.org/x/net/html"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"

	"github.com/joodaloop/hugolint/internal/config"
	"github.com/joodaloop/hugolint/internal/rules"
)

func Markdown(cfg *config.Config) (int, error) {
	root := cfg.Paths.MarkdownRoot
	paths, err := walk(root, func(path string, d fs.DirEntry) bool {
		if d.IsDir() && cfg.SkipDir(d.Name()) {
			return false
		}
		if d.IsDir() {
			return true
		}
		return strings.HasSuffix(path, ".md")
	})
	if err != nil {
		return 0, err
	}

	mdCtx := &rules.MarkdownContext{Config: cfg}
	fmCtx := &rules.FrontmatterContext{Config: cfg}
	mdParser := goldmark.New().Parser()
	legacy := rules.Markdown()
	fmRules := rules.Frontmatter()
	astRules := rules.MarkdownAST()
	textRules := rules.MarkdownText()

	diags := runParallel(paths, func(p string) []rules.Diagnostic {
		b, err := os.ReadFile(p)
		if err != nil {
			return []rules.Diagnostic{{Path: p, Rule: "io", Message: err.Error()}}
		}

		fmRaw, body, fmLines, fmStartLine := rules.SplitFrontmatter(b)
		ff := &rules.FrontmatterFile{
			Path:   p,
			Raw:    fmRaw,
			Parsed: rules.ParseFrontmatterYAML(fmRaw),
			Line0:  fmStartLine,
		}
		mf := &rules.MarkdownFile{
			Path:          p,
			Content:       b,
			Body:          body,
			AST:           mdParser.Parse(text.NewReader(body)),
			BodyStartLine: fmLines + 1,
		}

		var out []rules.Diagnostic
		for _, r := range fmRules {
			out = append(out, r.Check(ff, fmCtx)...)
		}
		for _, r := range astRules {
			out = append(out, r.Check(mf, mdCtx)...)
		}
		for _, r := range textRules {
			out = append(out, r.Check(mf, mdCtx)...)
		}
		for _, r := range legacy {
			out = append(out, r.Check(mf, mdCtx)...)
		}
		return out
	})

	report(diags)
	return len(diags), nil
}

func Build(cfg *config.Config, root string) (int, error) {
	files, allFiles, pages, pageIDs, err := loadHTML(root)
	if err != nil {
		return 0, err
	}
	ctx := &rules.HTMLContext{Root: root, Pages: pages, PageIDs: pageIDs, Config: cfg, LinkedPages: map[string]bool{}}

	rs := rules.HTML()
	diags := runFiles(files, func(f *rules.HTMLFile) []rules.Diagnostic {
		var out []rules.Diagnostic
		for _, r := range rs {
			out = append(out, r.Check(f, ctx)...)
		}
		return out
	})

	if err := rules.ScanCSSLinks(allFiles, ctx); err != nil {
		return 0, err
	}
	diags = append(diags, rules.ReportOrphans(allFiles, ctx)...)

	tidyDiags, err := tidyDiagnostics(root)
	if err != nil {
		return 0, err
	}
	diags = append(diags, tidyDiags...)

	report(diags)
	return len(diags), nil
}

func loadHTML(root string) ([]*rules.HTMLFile, []rules.BuiltFile, map[string]bool, map[string]map[string]int, error) {
	pages := make(map[string]bool)
	pageIDs := make(map[string]map[string]int)
	var files []*rules.HTMLFile
	var allFiles []rules.BuiltFile

	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		u := "/" + filepath.ToSlash(rel)
		pages[u] = true
		var alts []string
		if strings.HasSuffix(u, "/index.html") {
			withSlash := strings.TrimSuffix(u, "index.html")
			pages[withSlash] = true
			pages[strings.TrimSuffix(withSlash, "/")] = true
			alts = []string{withSlash, strings.TrimSuffix(withSlash, "/")}
		}

		allFiles = append(allFiles, rules.BuiltFile{Path: p, URLPath: urlPathFor(root, p)})

		if !strings.HasSuffix(p, ".html") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		links, images, assets, ids, text, title, lang, metas, headLinks := parseHTML(b)
		files = append(files, &rules.HTMLFile{
			Path:      p,
			URLPath:   urlPathFor(root, p),
			Links:     links,
			Images:    images,
			Assets:    assets,
			IDs:       ids,
			Text:      text,
			Title:     title,
			Lang:      lang,
			Metas:     metas,
			HeadLinks: headLinks,
		})
		pageIDs[u] = ids
		for _, alt := range alts {
			pageIDs[alt] = ids
		}
		return nil
	})
	return files, allFiles, pages, pageIDs, err
}

func parseHTML(content []byte) (links, images []string, assets []rules.Asset, ids map[string]int, text, title, lang string, metas []rules.MetaTag, headLinks []rules.HeadLink) {
	ids = map[string]int{}
	var textBuf, titleBuf bytes.Buffer
	skipDepth := 0 // depth inside <script>/<style>/<pre>/<code>
	headDepth := 0
	titleDepth := 0

	isSkipTag := func(tag string) bool {
		switch tag {
		case "script", "style", "pre", "code":
			return true
		}
		return false
	}

	z := html.NewTokenizer(bytes.NewReader(content))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}

		switch tt {
		case html.TextToken:
			if titleDepth > 0 {
				titleBuf.Write(z.Text())
			}
			if skipDepth == 0 {
				textBuf.Write(z.Text())
				// Separator between text nodes so pattern scans can't
				// straddle tag boundaries (e.g. "Sun (" + <a>http://...</a>
				// must not look like a leaked "(http").
				textBuf.WriteByte(0)
			}
			continue
		case html.EndTagToken:
			name, _ := z.TagName()
			tagStr := string(name)
			if isSkipTag(tagStr) && skipDepth > 0 {
				skipDepth--
			}
			if tagStr == "head" && headDepth > 0 {
				headDepth--
			}
			if tagStr == "title" && titleDepth > 0 {
				titleDepth--
			}
			continue
		case html.StartTagToken, html.SelfClosingTagToken:
			// fall through to attribute extraction
		default:
			continue
		}

		name, hasAttr := z.TagName()
		tag := string(name)
		if tt == html.StartTagToken && isSkipTag(tag) {
			skipDepth++
		}
		if tt == html.StartTagToken && tag == "head" {
			headDepth++
		}
		if tt == html.StartTagToken && tag == "title" {
			titleDepth++
		}
		var href, src, id, metaName, metaProp, metaEquiv, metaCharset, metaContent, linkRel, linkType, linkTitle, htmlLang string
		if hasAttr {
			for {
				k, v, more := z.TagAttr()
				switch string(k) {
				case "href":
					href = string(v)
				case "src":
					src = string(v)
				case "id":
					id = string(v)
				case "name":
					metaName = string(v)
				case "property":
					metaProp = string(v)
				case "http-equiv":
					metaEquiv = string(v)
				case "charset":
					metaCharset = string(v)
				case "content":
					metaContent = string(v)
				case "rel":
					linkRel = string(v)
				case "type":
					linkType = string(v)
				case "title":
					linkTitle = string(v)
				case "lang":
					htmlLang = string(v)
				}
				if !more {
					break
				}
			}
		}
		if tag == "meta" {
			metas = append(metas, rules.MetaTag{
				Name:      metaName,
				Property:  metaProp,
				HTTPEquiv: metaEquiv,
				Charset:   metaCharset,
				Content:   metaContent,
			})
		}
		if tag == "link" && headDepth > 0 {
			headLinks = append(headLinks, rules.HeadLink{
				Rel:   linkRel,
				Type:  linkType,
				Href:  href,
				Title: linkTitle,
			})
		}
		switch tag {
		case "a":
			if href != "" {
				links = append(links, href)
			}
		case "img":
			if src != "" {
				images = append(images, src)
			}
		case "link":
			if href != "" {
				assets = append(assets, rules.Asset{Tag: "link", Attr: "href", URL: href})
			}
		case "script", "source", "video", "audio":
			if src != "" {
				assets = append(assets, rules.Asset{Tag: tag, Attr: "src", URL: src})
			}
		}
		if tag == "html" {
			lang = strings.TrimSpace(htmlLang)
		}
		if id != "" {
			ids[id]++
		}
	}
	text = textBuf.String()
	title = strings.TrimSpace(titleBuf.String())
	return
}

func urlPathFor(root, file string) string {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return "/"
	}
	u := "/" + filepath.ToSlash(rel)
	if strings.HasSuffix(u, "/index.html") {
		return strings.TrimSuffix(u, "index.html")
	}
	return u
}

func walk(root string, keep func(string, fs.DirEntry) bool) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !keep(path, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

func runParallel(paths []string, fn func(string) []rules.Diagnostic) []rules.Diagnostic {
	jobs := make(chan string)
	results := make(chan []rules.Diagnostic)

	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				results <- fn(p)
			}
		}()
	}

	go func() {
		for _, p := range paths {
			jobs <- p
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var all []rules.Diagnostic
	for ds := range results {
		all = append(all, ds...)
	}
	return all
}

func runFiles[T any](items []T, fn func(T) []rules.Diagnostic) []rules.Diagnostic {
	jobs := make(chan T)
	results := make(chan []rules.Diagnostic)

	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := range jobs {
				results <- fn(it)
			}
		}()
	}

	go func() {
		for _, it := range items {
			jobs <- it
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var all []rules.Diagnostic
	for ds := range results {
		all = append(all, ds...)
	}
	return all
}

func report(diags []rules.Diagnostic) {
	sort.Slice(diags, func(i, j int) bool {
		if diags[i].Path != diags[j].Path {
			return diags[i].Path < diags[j].Path
		}
		return diags[i].Line < diags[j].Line
	})

	color := stdoutIsTTY()
	const (
		reset   = "\x1b[0m"
		dim     = "\x1b[2m"
		bold    = "\x1b[1m"
		cyan    = "\x1b[36m"
		yellow  = "\x1b[33m"
		magenta = "\x1b[35m"
		green   = "\x1b[32m"
		red     = "\x1b[31m"
	)
	paint := func(s, code string) string {
		if !color {
			return s
		}
		return code + s + reset
	}

	// Compute column widths for grid alignment.
	locWidth, ruleWidth := 0, 0
	locs := make([]string, len(diags))
	for i, d := range diags {
		loc := d.Path
		if d.Line > 0 {
			loc = fmt.Sprintf("%s:%d", d.Path, d.Line)
		}
		locs[i] = loc
		if w := visibleLen(loc); w > locWidth {
			locWidth = w
		}
		if w := len(d.Rule); w > ruleWidth {
			ruleWidth = w
		}
	}

	for i, d := range diags {
		loc := locs[i]
		var locStr string
		if color {
			if d.Line > 0 {
				locStr = paint(d.Path, cyan) + paint(":", dim) + paint(fmt.Sprintf("%d", d.Line), yellow)
			} else {
				locStr = paint(d.Path, cyan)
			}
		} else {
			locStr = loc
		}
		pad := strings.Repeat(" ", locWidth-visibleLen(loc))
		ruleStr := paint(fmt.Sprintf("%-*s", ruleWidth, d.Rule), magenta)
		fmt.Printf("%s%s  %s  %s\n", locStr, pad, ruleStr, d.Message)
	}

	if len(diags) > 0 {
		summary := fmt.Sprintf("\n%d issue", len(diags))
		if len(diags) != 1 {
			summary += "s"
		}
		fmt.Println(paint(summary, bold) + paint(" found", red))
	} else {
		fmt.Println(paint("✓ no issues", green))
	}
}

func visibleLen(s string) int { return len(s) }

func stdoutIsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
