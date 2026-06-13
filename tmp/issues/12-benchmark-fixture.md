---
title: "Benchmark fixture: generated 10k-page site + go test -bench + README numbers"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: []
---

## What to build

A benchmark harness that generates a synthetic site (deterministic seed): 10k pages mixing Markdown and a declarative DSL, taxonomies with realistic term distribution, a few collections, layouts + partials. `go test -bench` runs full builds (and later, incremental builds from #13/#14) against it and reports pages/sec, total wall time, allocations. Publish the numbers in the README with hardware context.

Do this EARLY: it establishes the v0.2-era baseline so every other slice's perf impact (especially #02, #06, #13) is measured, not guessed.

## Acceptance criteria

- [ ] Generator produces the fixture deterministically from a seed into a temp dir (not committed — only the generator is)
- [ ] `go test -bench=Build` builds the 10k-page site end-to-end; benchmark excluded from normal `go test ./...` runs (guarded by `-bench`/build tag)
- [ ] Reports total time, pages/sec, and B/op via standard benchmark output
- [ ] Baseline numbers captured before Tier-1 merges land (recorded in the issue/PR), final numbers in README
- [ ] CI job runs the benchmark on a fixed runner class for trend visibility (informational, not gating)

## Blocked by

None - can start immediately

## Baseline (recorded 2026-06-13, pre-Tier-1 perf changes)

Apple M1 Pro, darwin/arm64, `go test -bench=Build -benchtime=3x .`:

| Site size | ms/build | pages/sec | ms/page |
|-----------|----------|-----------|---------|
| 1,000 pages | 287 | 3,480 | 0.29 |
| 10,000 pages | 4,402 | 2,271 | 0.44 |

Notes: unconditional JSON payload still enabled (pre-#06); partials/funcs (#01) already merged but
do not affect per-page cost. These are the comparison numbers for #06 and #13/#14.
