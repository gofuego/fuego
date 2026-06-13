---
title: "Template layer: theme/partials/ + curated template funcs"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: []
---

## What to build

Add a `theme/partials/` directory loaded into the template cache and callable from base, layouts, and renderers via `{{partial "nav" .}}`. Add a curated, stdlib-only template function set: `partial`, `dict` (build args for partials), `where` (filter a slice of page refs by envelope key/value), `sortBy`, `limit`, `first`, `dateFormat` (parse + format envelope date strings). No sprig, no new dependencies.

This is the foundation slice: `.Site.Pages` (#02), outputs (#04), packs (#07), docs (#17), and scaffold (#18) all consume it.

## Acceptance criteria

- [ ] Files in `theme/partials/*.html` are parsed once at `LoadTemplates` and available to base, all layouts, and all renderer templates
- [ ] `partial` errors name the missing partial and the calling template (ties into error DX conventions from #05)
- [ ] All 7 funcs implemented with unit tests, including `where`/`sortBy` over `[]PageRef`-shaped data and `dateFormat` over common envelope date formats
- [ ] Integration fixture `testdata/partials-funcs/` with golden output exercising a partial called from base and a layout
- [ ] Docs: one reference page listing every template function with a working example
- [ ] `go test ./... -race` and `-count=3` determinism pass

## Blocked by

None - can start immediately
