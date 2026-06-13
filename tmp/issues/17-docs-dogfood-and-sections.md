---
title: "Docs site: dogfood v0.3 + new Diátaxis sections"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: [01, 02, 03, 04, 09]
---

## What to build

Convert the docs site (docs/) into the living proof of v0.3: nav rebuilt on `.Site.Pages` + partials (replacing any hardcoded nav), `sitemap.xml` + RSS via `theme/outputs/`, the tutorials collection paginated, and one real pack consumed end-to-end. Add the new content: concepts page for format packs, how-to per Tier-1 feature (nav menu, pagination, RSS/sitemap), "Build a format pack" tutorial, and reference pages for template functions, config merge rules, and `fuego config`. If a feature is awkward to use in our own docs site, that's a release blocker finding — file it.

## Acceptance criteria

- [ ] Docs nav driven by `.Site.Pages` + a partial; zero hardcoded page lists in theme
- [ ] sitemap.xml and RSS generated via theme/outputs/ and live on the published site
- [ ] Tutorials collection paginated (page_size exercised in production, even if small)
- [ ] One pack consumed by the docs site build (candidate: the #07/#08 example pack until #15 publishes)
- [ ] New pages: packs concept, 3+ Tier-1 how-tos, format-pack tutorial, references for funcs/merge/`fuego config`; existing pages updated where v0.3 changes behavior (.JSON note, hooks page)
- [ ] Docs build passes on v0.3 HEAD in CI (replace directive already points at parent)

## Blocked by

- 01, 02, 03, 04 (features it dogfoods), 09 (merge rules it documents)
