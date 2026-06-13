---
title: ".Site.Pages: slim cross-page refs in templates"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: [01]
---

## What to build

Expose `.Site.Pages` in `TemplateData` as a deterministic, URL-sorted slice of slim page refs — `URL`, `Type`, `Layout`, `Envelope` only (no Nodes, no Content). The slice is built once after BEFORE-RENDER (so hook mutations and virtual pages are included) and shared immutably across render workers. Pages with `Skip: true` (#10) are excluded once that lands.

End-to-end: a layout builds a nav menu and a "related pages" list using `where`/`sortBy` from #01 with zero Go code.

## Acceptance criteria

- [ ] `SiteTemplateData` gains `Pages []PageRef`; refs carry URL, Type, Layout, Envelope
- [ ] Built exactly once per build, after BEFORE-RENDER hooks, before RENDER fan-out; sorted by URL
- [ ] Render workers share the slice without copying; no data races under `go test -race`
- [ ] Integration fixture with a nav partial driven by `where .Site.Pages "type" "doc"` and golden output
- [ ] Virtual pages (taxonomy/collection) appear in the refs; envelope-only access verified in fixture
- [ ] Docs: how-to "Build a navigation menu" using only templates

## Blocked by

- 01-template-partials-and-funcs (fixture and docs use `where`/`sortBy`/`partial`)
