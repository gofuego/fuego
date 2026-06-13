---
title: "theme/outputs/: site-level non-HTML outputs (RSS, sitemap, feeds)"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: [02]
---

## What to build

Every file under `theme/outputs/` (e.g. `sitemap.xml`, `feed.xml`, `search.json`, `robots.txt`) is executed as a **text/template** (correct escaping for XML/JSON, not html/template) with `.Site` (including `.Site.Pages` refs) and written to the same relative path in the output dir, after RENDER. Nested directories preserved. The engine stays format-blind: RSS/sitemap are not features, they are recipe templates shipped in the scaffold (#18) and docs.

## Acceptance criteria

- [ ] Files in `theme/outputs/**` rendered with text/template + the curated func map from #01, written to matching output paths
- [ ] Output paths participate in collision detection against page URLs (GlobalFatal on conflict)
- [ ] Template errors report the output file name (error DX conventions from #05)
- [ ] Working `sitemap.xml` and `rss.xml` recipe templates exist and are exercised in an integration fixture with golden output
- [ ] Outputs listed in `site-manifest.json` is NOT required (manifest stays page-focused) — documented
- [ ] Docs: how-to "Add an RSS feed and sitemap"

## Blocked by

- 02-site-pages-slim-refs (outputs iterate `.Site.Pages`)
