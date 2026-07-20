package runner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joodaloop/joodalint/internal/config"
	"github.com/joodaloop/joodalint/internal/rules"
)

var dateLayouts = []string{
	"2006-01-02",
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
}

func Help(cfg *config.Config) error {
	root := cfg.Paths.MarkdownRoot
	paths, err := walk(root, func(path string, d fs.DirEntry) bool {
		if cfg.SkipMarkdown(root, path) {
			return false
		}
		if d.IsDir() {
			return true
		}
		return config.IsMarkdownPath(path)
	})
	if err != nil {
		return err
	}
	draftPaths, err := walk(root, func(path string, d fs.DirEntry) bool {
		if d.IsDir() {
			if cfg.SkipMarkdown(root, path) {
				return strings.Contains(strings.ToLower(d.Name()), "draft")
			}
			return true
		}
		return config.IsMarkdownPath(path)
	})
	if err != nil {
		return err
	}
	printPostsChart(paths)
	printLongestDrafts(paths, draftPaths, root)
	return nil
}

func printLongestDrafts(paths, draftFolderPaths []string, root string) {
	type draft struct {
		path  string
		title string
		words int
	}
	var drafts []draft
	seen := make(map[string]bool)
	collect := func(p string, requireFlag bool) {
		if seen[p] {
			return
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return
		}
		fmRaw, body, _, _ := rules.SplitFrontmatter(b)
		words := len(strings.Fields(string(body)))
		var title string
		isDraft := false
		if len(fmRaw) > 0 {
			fm, err := rules.ParseFrontmatterYAML(fmRaw)
			if err == nil {
				if t, ok := fm["title"].(string); ok {
					title = t
				}
				if d, ok := fm["draft"]; ok && d == true {
					isDraft = true
				}
			}
		}
		if isDraft || !requireFlag {
			seen[p] = true
			drafts = append(drafts, draft{path: p, title: title, words: words})
		}
	}
	inRegular := make(map[string]bool, len(paths))
	for _, p := range paths {
		inRegular[p] = true
	}
	for _, p := range paths {
		collect(p, true)
	}
	for _, p := range draftFolderPaths {
		if inRegular[p] {
			continue
		}
		collect(p, false)
	}

	if len(drafts) == 0 {
		return
	}

	sort.Slice(drafts, func(i, j int) bool { return drafts[i].words > drafts[j].words })
	if len(drafts) > 10 {
		drafts = drafts[:10]
	}

	color := stdoutIsTTY()
	heading := "These drafts are closest to completion"
	if color {
		fmt.Printf("\n\x1b[1m%s\x1b[0m\n\n", heading)
	} else {
		fmt.Printf("\n%s\n\n", heading)
	}

	maxWords := drafts[0].words
	segWidth, titleWidth := 0, 0
	segs := make([]string, len(drafts))
	for i, d := range drafts {
		segs[i] = filepath.Base(d.path)
		if len(segs[i]) > segWidth {
			segWidth = len(segs[i])
		}
		if len(d.title) > titleWidth {
			titleWidth = len(d.title)
		}
	}

	const barWidth = 15
	for i, d := range drafts {
		var barLen int
		if maxWords > 0 {
			barLen = d.words * barWidth / maxWords
		}
		bar := strings.Repeat("█", barLen)
		segPad := strings.Repeat(" ", segWidth-len(segs[i]))
		titlePad := strings.Repeat(" ", titleWidth-len(d.title))
		count := formatCount(d.words)
		if color {
			fmt.Printf("  %s%s  \x1b[34m%s\x1b[0m%s  \x1b[33m%s\x1b[2m%s\x1b[0m  \x1b[33m%s\x1b[0m\n",
				segs[i], segPad, d.title, titlePad, bar, strings.Repeat(" ", barWidth-barLen), count)
		} else {
			fmt.Printf("  %s%s  %s%s  %s%s  %s\n",
				segs[i], segPad, d.title, titlePad, bar, strings.Repeat(" ", barWidth-barLen), count)
		}
	}
}

func formatCount(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	v := float64(n) / 1000
	if v >= 10 {
		return strconv.Itoa(int(v+0.5)) + "k"
	}
	return strconv.FormatFloat(float64(int(v*10+0.5))/10, 'f', 1, 64) + "k"
}

func printPostsChart(paths []string) {
	counts := make(map[string]int)
	var first, last time.Time

	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		fmRaw, _, _, _ := rules.SplitFrontmatter(b)
		if len(fmRaw) == 0 {
			continue
		}
		fm, err := rules.ParseFrontmatterYAML(fmRaw)
		if err != nil {
			continue
		}
		if d, ok := fm["draft"]; ok && d == true {
			continue
		}
		dateVal, ok := fm["date"]
		if !ok {
			continue
		}
		t, ok := parseDate(dateVal)
		if !ok {
			continue
		}
		key := quarterLabel(t)
		counts[key]++
		if first.IsZero() || t.Before(first) {
			first = t
		}
		if t.After(last) {
			last = t
		}
	}

	if len(counts) == 0 {
		return
	}

	quarters := quarterRange(first, last)
	var items []labeledCount
	for _, q := range quarters {
		items = append(items, labeledCount{label: q, count: counts[q]})
	}
	printBarChart("Posts per quarter ("+first.Format("2006-01-02")+" → "+last.Format("2006-01-02")+")", items)
}

type labeledCount struct {
	label string
	count int
}

func printBarChart(title string, items []labeledCount) {
	maxCount := 0
	total := 0
	for _, it := range items {
		if it.count > maxCount {
			maxCount = it.count
		}
		total += it.count
	}

	const barWidth = 40
	color := stdoutIsTTY()

	if color {
		fmt.Printf("\n\x1b[1m%s\x1b[0m\n\n", title)
	} else {
		fmt.Printf("\n%s\n\n", title)
	}

	for _, it := range items {
		c := it.count
		var barLen int
		if maxCount > 0 {
			barLen = c * barWidth / maxCount
		}

		if c > 0 {
			bar := strings.Repeat("█", barLen)
			if color {
				fmt.Printf("  %s  \x1b[33m%s\x1b[2m%s\x1b[0m  %d\n", it.label, bar, strings.Repeat(" ", barWidth-barLen), c)
			} else {
				fmt.Printf("  %s  %s  %d\n", it.label, bar, c)
			}
		} else {
			if color {
				fmt.Printf("  %s  \x1b[2m·\x1b[0m\n", it.label)
			} else {
				fmt.Printf("  %s  ·\n", it.label)
			}
		}
	}

	if total > 0 {
		if color {
			fmt.Printf("\n\x1b[2m%d posts total\x1b[0m\n", total)
		} else {
			fmt.Printf("\n%d posts total\n", total)
		}
	}
}

func quarterLabel(t time.Time) string {
	q := (t.Month()-1)/3 + 1
	return t.Format("2006") + " Q" + strconv.Itoa(int(q))
}

func quarterRange(start, end time.Time) []string {
	y, q := start.Year(), (start.Month()-1)/3+1
	endY, endQ := end.Year(), (end.Month()-1)/3+1
	var quarters []string
	for y < endY || (y == endY && q <= endQ) {
		quarters = append(quarters, strconv.Itoa(y)+" Q"+strconv.Itoa(int(q)))
		q++
		if q > 4 {
			q = 1
			y++
		}
	}
	return quarters
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
