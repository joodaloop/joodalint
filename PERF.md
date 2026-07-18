# Performance notes

- Three identical rule interfaces: MarkdownRule, MarkdownASTRule, and MarkdownTextRule have byte-for-byte identical signatures, and the runner iterates the three registries with identical loops. The
split looks organizational, but a single interface with a category tag (or just one registry) would remove ~40 lines of ceremony.
- ANSI color boilerplate is copied three times (report, ReportJSMetrics, and now my summary follows suit). A tiny shared paint helper package-side would consolidate it.
- visibleLen in runner.go:534 is just len(s) — either a placeholder for future ANSI-aware width handling or dead indirection.

All numbers below were measured on an Apple M4, single-threaded, on
2026-05-01 via:

`go test -run='^$' -bench=. -benchmem ./internal/rules ./internal/runner`

This file now favors measured numbers over structural guesses. Where a point is
still an inference, it is called out as such.

## Measured baselines

### HTML parse — fast, not a bottleneck
| Size | Throughput | Per-call | Allocs |
|---|---|---|---|
| Small  (~6 KB)   | 214 MB/s | 29.3 us  | 262 |
| Medium (~27 KB)  | 233 MB/s | 117.0 us | 913 |
| Large  (~135 KB) | 237 MB/s | 570.0 us | 4 125 |

`parseHTML` is comfortably sub-millisecond even on a large page. This is not
where build time is going.

### Goldmark AST parse — also fast
| Size | Throughput | Per-call | Allocs |
|---|---|---|---|
| Small  (~5 KB)   | 127 MB/s | 39.8 us  | 394 |
| Medium (~26 KB)  | 129 MB/s | 196.1 us | 1 914 |
| Large  (~104 KB) | 122 MB/s | 830.0 us | 7 617 |

Markdown parsing is not free, but it is still well below the heavier rule
costs.

### FlattenProse (AST → ProseBlocks transformation)
| Size | Throughput | Per-call | Allocs |
|---|---|---|---|
| Small  (~5 KB)   | 665 MB/s |  7.6 us  | 167 |
| Medium (~26 KB)  | 601 MB/s | 42.1 us  | 809 |
| Large  (~104 KB) | 525 MB/s | 192.8 us | 3 211 |

`FlattenProse` walks the AST to extract inline prose spans and is per-file
setup work consumed by the AST-side `prose-hygiene` rule and `balance`. It is
fast (sub-200 us even on large files), so the bulk of the ~2.6 ms AST-side
`prose-hygiene` cost is in the rule checks, not the transformation.

### FlattenProse (AST → ProseBlocks transformation)
| Size | Throughput | Per-call | Allocs |
|---|---|---|---|
| Small  (~5 KB)   | 665 MB/s |  7.6 us  | 167 |
| Medium (~26 KB)  | 601 MB/s | 42.1 us  | 809 |
| Large  (~104 KB) | 525 MB/s | 192.8 us | 3 211 |

`FlattenProse` walks the AST to extract inline prose spans and is per-file
setup work consumed by the AST-side `prose-hygiene` rule and `balance`. It is
fast (sub-200 us even on large files), so the bulk of the ~2.6 ms AST-side
`prose-hygiene` cost is in the rule checks, not the transformation.

### Markdown rules, ranked (medium ~26 KB input)
| Rule | Throughput | Per-call | Allocs |
|---|---|---|---|
| `image-alt`       | 4 286 MB/s | 5.9 us   | 0 |
| `formatting`      | 2 872 MB/s | 8.8 us   | 75 |
| `headings`        | 2 132 MB/s | 11.9 us  | 6 |
| `url`             | 743 MB/s   | 34.0 us  | 150 |
| `prose-hygiene` (line-by-line) | 84.7 MB/s | 0.30 ms | 1 077 |
| `balance`         | 859 MB/s   | 29.4 us  | 0 |
| `prose-hygiene` (AST-based) | 9.78 MB/s | **2.59 ms** | 6 383 |

Both `prose-hygiene` halves share the same rule ID and both execute per file,
so the effective combined cost is ~2.87 ms — still the #1 in-process bottleneck
by a factor of ~9x over the next-heaviest rule. The AST-based half is the
expensive one.

Scaling for `prose-hygiene` is linear and consistent:

| Size | Throughput | Per-call | Allocs |
|---|---|---|---|
| Small  (~5 KB)   | 88.4 MB/s | 0.057 ms | 217 |
| Medium (~26 KB)  | 91.9 MB/s | 0.28 ms  | 1 077 |
| Large  (~104 KB) | 92.6 MB/s | 1.09 ms  | 4 303 |

Note: this is the structural (line-by-line) `Markdown()` rule. The AST-based
half (`MarkdownAST`) that checks ProseBlocks is heavier at ~2.6 ms for a medium
file, but is largely redundant — most line content checks run in both.
Consolidating them is a high-value optimization target.

### HTML rules, ranked (synthetic page with 100 internal pages / assets)
| Rule | Per-call | Allocs |
|---|---|---|
| `rendered-artifacts`     | 45.1 us  | 832 |
| `head-metadata`          | 46.3 us  | 840 |
| `image-src-exists`       | 67.1 us  | 1 042 |
| `asset-src-exists`       | 88.0 us  | 1 244 |
| `fragment-link-exists`   | 89.9 us  | 1 332 |
| `relative-link-exists`   | 128.1 us | 1 752 |

The slowest registered HTML rule here is still only ~0.13 ms/page. That puts
the HTML rule layer far below `tidy`.

### CSS scan and orphan report
| Task | Per-call | Allocs |
|---|---|---|
| `ReportOrphans` (401 files) | 14.5 us | 700 |
| `ScanCSSLinks` (20 CSS files, 2000 refs) | 1.24 ms | 10 212 |

`ScanCSSLinks` is measurable, but still small: about 62 us per CSS file in this
fixture.

### Frontmatter
| Task | Throughput | Per-call | Allocs |
|---|---|---|---|
| `SplitFrontmatter` | 2 115 MB/s | 0.48 us  | 0 |
| `ParseFrontmatterYAML` | 14.1 MB/s | 70.5 us | 2 312 |
| `frontmatter.Check` | 359 MB/s | 2.77 us | 1 |

The expensive part is the YAML parse, not the field validation. That means the
"frontmatter is parsed twice" note is real, but it is a tens-of-microseconds
issue, not a milliseconds-per-file issue.

### Line lookup helpers
| Helper | Per-call | Allocs |
|---|---|---|
| `LineAt` | 1.19 us | 0 |
| `NodeLine` | 13.4 ns | 0 |

`LineAt` is much more expensive than `NodeLine` in this benchmark. `NodeLine`
looks cheap here because the benchmarked heading node can use block line
metadata quickly; text-heavy nodes that force `earliestTextStart` subtree walks
will cost more. Even so, line lookup is nowhere near `prose-hygiene`,
`aspell`, or `tidy`.

### aspell (external process)
| Mode | Throughput | Per-call | Allocs |
|---|---|---|---|
| Startup (tiny body) | n/a | 5.12 ms | 72 |
| Throughput (~53 KB body) | 6.26 MB/s | 8.44 ms | 74 |

Subtracting the tiny-body startup cost from the large-body run leaves about
3.32 ms of actual scan time for ~53 KB, or about 16.0 MB/s of spell-checking
work after process startup.

Interpretation:

- The fixed startup tax is about 5.1 ms per Markdown file.
- On small Markdown files, startup dominates the cost.
- On a 1000-file docs tree, that fixed tax alone is about 5.1 s serially.

### tidy (external process)
| Mode | Throughput | Per-call | Allocs |
|---|---|---|---|
| Startup (tiny page) | n/a | 1.79 ms | 65 |
| Throughput (~54 KB page) | 16.1 MB/s | 3.37 ms | 65 |

Subtracting startup from the larger-page run leaves about 1.59 ms of actual
validation work for ~54 KB, or about 34 MB/s after process startup.

Interpretation:

- `tidy` is materially cheaper per file than `aspell` in this environment.
- It still pays a real fork+exec tax, so batching or caching would help.
- The earlier claim that `tidy` was "probably the dominant build cost" was too
  strong without measurement.

## Bottleneck ranking

### 1. `prose-hygiene` is the biggest in-process cost by a wide margin
The structural (line-by-line) half clocks in at ~0.28 ms for a medium file, but
the AST-based half weighs in at ~2.6 ms. The two halves have overlapping
coverage (many per-line checks happen in both), so consolidating them into a
single pass is the highest-leverage pure-Go optimization target.

The 6.5x improvement over earlier baselines came from fused ReplaceAllString
passes, cheap byte/substr prechecks, and removing the `***` ambiguity check.

### 2. `aspell` is the biggest cost when spelling is enabled
A ~5.1 ms per-file startup tax is large enough that on typical Markdown sizes
it will dominate all in-process rules, including `prose-hygiene`.

Best next moves:

- Keep one long-lived aspell process instead of one process per file.
- Or batch multiple files through one invocation.

### 3. `tidy` is a real build cost, but smaller than the earlier guess
At ~1.8 ms startup and ~3.4 ms total on a ~54 KB page, `tidy` is still one of
the heavier build-time checks, but it is not obviously catastrophic on its own.

Best next moves:

- Cache by content hash.
- Batch pages where possible.
- Only replace it with an in-process validator if correctness and maintenance
  tradeoffs are acceptable.

### 4. HTML rule checks are not the problem
Registered HTML rules land in the ~45-128 us range on a fairly busy synthetic
page. Even `ScanCSSLinks`, which is more expensive than the rules themselves,
is still only ~1.2 ms for 20 CSS files.

The repeated URL parsing in `isRelative` + `resolve` is still worth cleaning
up, but this is polish, not a top-tier bottleneck.

### 5. Frontmatter reparsing is worth fixing, but only because it is easy
The redundant YAML parse costs about 70 us/file. That is real and entirely
avoidable, but it is nowhere near the cost of `prose-hygiene`, `aspell`, or
`tidy`.

### 6. Line lookup and orphan reporting are low priority
`LineAt` is microsecond-scale, `NodeLine` is usually much cheaper, and orphan
reporting is tiny. These are not the first places to spend optimization effort.

## Repeated work that is still worth addressing

- Frontmatter YAML is parsed once to build `FrontmatterFile` and the measured
  parse cost is ~70.5 us/file. Removing the second parse is straightforward.
- `LineAt` still does an O(N) newline count from the start of the file. The
  benchmark says the current cost is modest, but a precomputed line offset
  table would remove it entirely.
- Markdown AST rules still walk the tree independently. The rule timings say
  this is secondary next to `prose-hygiene`, but it is still duplicated work.
- HTML rules repeatedly parse and resolve URLs. Measured rule times say the
  impact is small right now, but the duplication is real.
- `MarkLinked` still takes a mutex for every discovered target. Nothing here
  shows that as a current bottleneck, but it could matter on larger parallel
  sites.

## Probably not bottlenecks

- HTML parsing itself
- Goldmark parsing itself
- `rendered-artifacts`
- `ReportOrphans`
- Frontmatter field validation after parse

## Bottom line

If the goal is wall-clock speed, the priority order is now:

1. Consolidate the two `prose-hygiene` halves (line-by-line + AST) into one pass.
   The AST half at 2.6 ms/medium-file is the single largest in-process cost.
2. Replace per-file `aspell` startup with batching or a long-lived process.
3. Decide whether `tidy` needs caching or batching for large sites.
4. Remove redundant frontmatter parsing.
5. Clean up duplicated URL parsing and, later, shared AST walking.
