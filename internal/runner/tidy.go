package runner

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/joodaloop/joodalint/internal/rules"
)

// tidyDiagnostics shells out to tidy-html5 for every non-skipped .html
// file in the build. Returns nil diagnostics (and no error) when tidy is
// missing or is Apple's HTML4-era build, so `lint build` stays usable
// without the dependency.
func tidyDiagnostics(files []rules.BuiltFile) ([]rules.Diagnostic, error) {
	if _, err := exec.LookPath("tidy"); err != nil {
		fmt.Println("joodalint: tidy not installed — skipping HTML validity check. (brew install tidy-html5)")
		return nil, nil
	}
	if out, err := exec.Command("tidy", "--version").CombinedOutput(); err == nil {
		if bytes.Contains(out, []byte("Apple Inc.")) {
			fmt.Println("joodalint: Apple's /usr/bin/tidy is HTML4-era — skipping. (brew install tidy-html5)")
			return nil, nil
		}
	}

	var paths []string
	for _, f := range files {
		if f.Skipped || !strings.HasSuffix(f.Path, ".html") {
			continue
		}
		paths = append(paths, f.Path)
	}

	return runFiles(paths, func(p string) []rules.Diagnostic {
		cmd := exec.Command("tidy", "-quiet", "-errors", "--gnu-emacs", "yes",
			"--new-blocklevel-tags", "command-menu,fulltext-search,chat-form", p)
		out, _ := cmd.CombinedOutput()
		var ds []rules.Diagnostic
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.Contains(line, "Warning:") {
				continue
			}
			ds = append(ds, parseTidyLine(p, line))
		}
		return ds
	}), nil
}

// parseTidyLine parses gnu-emacs-format tidy output: "path:line:col: Error: msg".
// Falls back to a single-message diagnostic if the format doesn't match.
func parseTidyLine(path, line string) rules.Diagnostic {
	parts := strings.SplitN(line, ":", 5)
	if len(parts) == 5 {
		if n, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			return rules.Diagnostic{
				Path:    path,
				Line:    n,
				Rule:    "tidy",
				Message: strings.TrimSpace(parts[3]) + ":" + parts[4],
			}
		}
	}
	return rules.Diagnostic{Path: path, Rule: "tidy", Message: line}
}
