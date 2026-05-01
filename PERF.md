# Performance notes

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

### Markdown rules, ranked (medium ~26 KB input)
| Rule | Throughput | Per-call | Allocs |
|---|---|---|---|
| `image-alt`       | 4 286 MB/s | 5.9 us   | 0 |
| `formatting`      | 2 872 MB/s | 8.8 us   | 75 |
| `headings`        | 2 132 MB/s | 11.9 us  | 6 |
| `url`             | 743 MB/s   | 34.0 us  | 150 |
| `prose-hygiene`   | 84.7 MB/s  | 0.30 ms  | 1 077 |
| `balance`         | —          | —        | — |

The `balance` benchmark currently reports implausible numbers (2.25 ns, 0 allocs),
suggesting it is hitting a no-op path. Historical: ~370 MB/s, 69.8 us, 359 allocs.

`prose-hygiene` is the clear in-process Markdown bottleneck. It is now roughly:

- 2.5x slower than `url`
- 25x slower than `headings`
- 34x slower than `formatting`
- 50x slower than `image-alt`

Scaling for `prose-hygiene` is linear and consistent:

| Size | Throughput | Per-call | Allocs |
|---|---|---|---|
| Small  (~5 KB)   | 81.1 MB/s | 0.062 ms | 217 |
| Medium (~26 KB)  | 85.2 MB/s | 0.30 ms  | 1 077 |
| Large  (~104 KB) | 85.9 MB/s | 1.18 ms  | 4 303 |

Relative to the earlier baseline (~13.2 MB/s), this is about a 6.5x throughput
improvement with about 5.5x fewer allocations.

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
At ~0.30 ms for a medium file, it still dominates every other Go-side Markdown
rule. This is the highest-leverage pure-Go optimization target.

The optimizations implemented (fused ReplaceAllString passes, cheap byte/substr
prechecks) brought it down from ~1.96 ms to ~0.30 ms — about a 6.5x improvement.
The `***` ambiguity check was removed. The literal lead-byte prefilter was
tried but not kept.

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

1. Replace per-file `aspell` startup with batching or a long-lived process.
2. Decide whether `tidy` needs caching or batching for large sites.
3. Remove redundant frontmatter parsing.
4. Clean up duplicated URL parsing and, later, shared AST walking.

`prose-hygiene` at ~0.30 ms/medium-file is no longer the dominant bottleneck.
External process startup (`aspell` at 5.1 ms, `tidy` at 1.8 ms) now dwarfs
all in-process Go costs combined.
