---
title: "Scaffold refresh: showcase v0.3 surface + rewritten CLAUDE.md"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: [01, 02, 03, 04]
---

## What to build

Update `fuego init`'s embedded scaffold so a new user's first thirty seconds demonstrate the v0.3 surface: nav partial driven by `.Site.Pages`, `theme/outputs/sitemap.xml` + `rss.xml` recipes, a small paginated collection example alongside the existing `.card` DSL, and a rewritten scaffold CLAUDE.md covering packs, the three hook points, template functions, and `fuego config` — so agents working in scaffolded projects build correctly from minute one.

## Acceptance criteria

- [ ] Scaffolded site builds clean and demonstrates: partial-based nav, sitemap/RSS outputs, one paginated collection, the `.card` DSL
- [ ] Scaffold CLAUDE.md rewritten: pack usage (`eng.Use`), AfterParse/Index/BeforeRender hooks, template func reference pointer, error-DX expectations
- [ ] `fuego serve` on a fresh scaffold works end-to-end (manual smoke + scaffold integration test updated)
- [ ] Scaffold golden test (`internal/scaffold`) updated; output deterministic
- [ ] README quick-start section updated to match the new scaffold tree

## Blocked by

- 01, 02, 03, 04 (features the scaffold showcases)
