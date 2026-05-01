# Performance notes

All numbers below were measured on an Apple M4, single-threaded, on
2026-05-01 via:

`GOCACHE=/Users/judah/Documents/hugolint/.gocache go test -bench=. -benchmem ./internal/rules ./internal/runner`

This file now favors measured numbers over structural guesses. Where a point is
still an inference, it is called out as such.

## Measured baselines

### HTML parse — fast, not a bottleneck
| Size | Throughput | Per-call | Allocs |
|---|---|---|---|
| Small  (~6 KB)   | 184 MB/s | 34.0 us  | 262 |
| Medium (~27 KB)  | 225 MB/s | 121.6 us | 913 |
| Large  (~135 KB) | 235 MB/s | 576.2 us | 4 125 |

`parseHTML` is comfortably sub-millisecond even on a large page. This is not
where build time is going.

### Goldmark AST parse — also fast
| Size | Throughput | Per-call | Allocs |
|---|---|---|---|
| Small  (~5 KB)   | 126 MB/s | 40.9 us  | 392 |
| Medium (~26 KB)  | 129 MB/s | 200.0 us | 1 857 |
| Large  (~104 KB) | 127 MB/s | 816.3 us | 7 363 |

Markdown parsing is not free, but it is still well below the heavier rule
costs.

### Markdown rules, ranked (medium ~26 KB input)
| Rule | Throughput | Per-call | Allocs |
|---|---|---|---|
| `image-alt`       | 4 341 MB/s | 6.0 us   | 0 |
| `formatting`      | 2 971 MB/s | 8.7 us   | 72 |
| `headings`        | 2 232 MB/s | 11.6 us  | 6 |
| `url`             | 798 MB/s   | 32.4 us  | 144 |
| `balance`         | 370 MB/s   | 69.8 us  | 359 |
| `prose-hygiene`   | 13.2 MB/s  | 1.95 ms  | 5 931 |

`prose-hygiene` is still the clear in-process Markdown bottleneck, but the
optimized version is much cheaper. It is now roughly:

- 28x slower than `balance`
- 60x slower than `url`
- 224x slower than `formatting`
- 325x slower than `image-alt`

Scaling for `prose-hygiene` is linear and consistent:

| Size | Throughput | Per-call | Allocs |
|---|---|---|---|
| Small  (~5 KB)   | 13.28 MB/s | 0.389 ms | 1 184 |
| Medium (~26 KB)  | 13.20 MB/s | 1.96 ms  | 5 931 |
| Large  (~104 KB) | 13.28 MB/s | 7.80 ms  | 23 732 |

That is about a 3.2x throughput win, about 69% lower latency, and about 46%
fewer allocations than the earlier baseline. The allocator profile still
scales linearly, but the old `ReplaceAllString` pipeline is no longer the main
cost shape.

### HTML rules, ranked (synthetic page with 100 internal pages / assets)
| Rule | Per-call | Allocs |
|---|---|---|
| `rendered-artifacts`     | 43.6 us  | 832 |
| `head-metadata`          | 46.9 us  | 840 |
| `image-src-exists`       | 65.3 us  | 1 042 |
| `fragment-link-exists`   | 87.2 us  | 1 332 |
| `asset-src-exists`       | 90.8 us  | 1 244 |
| `relative-link-exists`   | 125.9 us | 1 752 |

The slowest registered HTML rule here is still only ~0.13 ms/page. That puts
the HTML rule layer far below `tidy`.

### CSS scan and orphan report
| Task | Per-call | Allocs |
|---|---|---|
| `ReportOrphans` (401 files) | 14.8 us | 700 |
| `ScanCSSLinks` (20 CSS files, 2000 refs) | 1.32 ms | 10 212 |

`ScanCSSLinks` is measurable, but still small: about 66 us per CSS file in this
fixture.

### Frontmatter
| Task | Throughput | Per-call | Allocs |
|---|---|---|---|
| `SplitFrontmatter` | 2 372 MB/s | 0.43 us  | 0 |
| `ParseFrontmatterYAML` | 14.5 MB/s | 68.6 us | 2 312 |
| `frontmatter.Check` | 371 MB/s | 2.68 us | 1 |

The expensive part is the YAML parse, not the field validation. That means the
"frontmatter is parsed twice" note is real, but it is a tens-of-microseconds
issue, not a milliseconds-per-file issue.

### Line lookup helpers
| Helper | Per-call | Allocs |
|---|---|---|
| `LineAt` | 1.21 us | 0 |
| `NodeLine` | 13.8 ns | 0 |

`LineAt` is much more expensive than `NodeLine` in this benchmark. `NodeLine`
looks cheap here because the benchmarked heading node can use block line
metadata quickly; text-heavy nodes that force `earliestTextStart` subtree walks
will cost more. Even so, line lookup is nowhere near `prose-hygiene`,
`aspell`, or `tidy`.

### aspell (external process)
| Mode | Throughput | Per-call | Allocs |
|---|---|---|---|
| Startup (tiny body) | n/a | 3.98 ms | 72 |
| Throughput (~53 KB body) | 7.14 MB/s | 7.40 ms | 73 |

Subtracting the tiny-body startup cost from the large-body run leaves about
3.42 ms of actual scan time for ~53 KB, or about 15.5 MB/s of spell-checking
work after process startup.

Interpretation:

- The fixed startup tax is about 4 ms per Markdown file.
- On small Markdown files, startup dominates the cost.
- On a 1000-file docs tree, that fixed tax alone is about 4 s serially.

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
At ~2.0 ms for a medium file in the last measured optimized run, it still
dominates every other Go-side Markdown rule.
This is the highest-leverage pure-Go optimization target.

Implemented:

- Implemented: fused the four `ReplaceAllString` passes into one scrubber.
- Implemented: added cheap byte/substr prechecks before most regex work.
- Implemented: cut allocations by roughly half in the measured synthetic
  benchmark.

Not kept:

- The literal lead-byte prefilter was not worth its complexity.
- The `***` ambiguity check was removed entirely.

Measurement note:

- The table above records the last measured optimized run. The final
  "worthwhile guards restored, `***` removed" cleanup was not rerun, so no
  post-cleanup benchmark number is asserted here.

### 2. `aspell` is the biggest cost when spelling is enabled
This is now measured, not inferred. A ~4 ms per-file startup tax is large
enough that on typical Markdown sizes it will dominate all in-process rules,
including `prose-hygiene`.

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
Registered HTML rules land in the ~44-126 us range on a fairly busy synthetic
page. Even `ScanCSSLinks`, which is more expensive than the rules themselves,
is still only ~1.3 ms for 20 CSS files.

The repeated URL parsing in `isRelative` + `resolve` is still worth cleaning
up, but this is polish, not a top-tier bottleneck.

### 5. Frontmatter reparsing is worth fixing, but only because it is easy
The redundant YAML parse costs about 69 us/file. That is real and entirely
avoidable, but it is nowhere near the cost of `prose-hygiene`, `aspell`, or
`tidy`.

### 6. Line lookup and orphan reporting are low priority
`LineAt` is microsecond-scale, `NodeLine` is usually much cheaper, and orphan
reporting is tiny. These are not the first places to spend optimization effort.

## Repeated work that is still worth addressing

- Frontmatter YAML is parsed once to build `FrontmatterFile` and the measured
  parse cost is ~68.6 us/file. Removing the second parse is straightforward.
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
- The `balance` rule

## Bottom line

If the goal is wall-clock speed, the priority order is now:

1. Speed up `prose-hygiene`.
2. Replace per-file `aspell` startup with batching or a long-lived process.
3. Decide whether `tidy` needs caching or batching for large sites.
4. Remove redundant frontmatter parsing.
5. Clean up duplicated URL parsing and, later, shared AST walking.

The biggest update from the new measurements is that the truly expensive work
is concentrated in `prose-hygiene` and external process startup. Most of the
other concerns in this codebase are real, but second-order.
