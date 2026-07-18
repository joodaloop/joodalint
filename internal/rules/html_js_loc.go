package rules

import (
	"fmt"
	"sort"
	"strings"
)

type jsFile struct {
	path    string
	urlPath string
	size    int64
}

func collectJSFiles(files []BuiltFile, ctx *HTMLContext) []jsFile {
	var candidates []jsFile
	for _, f := range files {
		if f.Skipped || f.Category != catJS {
			continue
		}
		if !isLinked(f, ctx) {
			continue
		}
		candidates = append(candidates, jsFile{path: f.Path, urlPath: f.URLPath, size: f.Size})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].size > candidates[j].size
	})
	return candidates
}

func ReportJSMetrics(files []BuiltFile, ctx *HTMLContext, color bool) {
	candidates := collectJSFiles(files, ctx)
	if len(candidates) == 0 {
		return
	}

	var total int64
	for _, jf := range candidates {
		total += jf.size
	}
	totalKB := total / 1024

	const (
		reset  = "\x1b[0m"
		yellow = "\x1b[33m"
		cyan   = "\x1b[36m"
	)
	paint := func(s, code string) string {
		if !color {
			return s
		}
		return code + s + reset
	}

	fmt.Printf("\nTotal site JS: %s across %d file", paint(fmt.Sprintf("%dKB", totalKB), yellow), len(candidates))
	if len(candidates) != 1 {
		fmt.Print("s")
	}
	fmt.Println()

	for _, jf := range candidates {
		kb := jf.size / 1024
		name := strings.TrimPrefix(jf.urlPath, "/")
		fmt.Printf("  %s %s\n", paint(name, cyan), paint(fmt.Sprintf("(%dKB)", kb), yellow))
	}
}
