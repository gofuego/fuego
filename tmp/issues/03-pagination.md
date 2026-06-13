---
title: "Pagination: page_size on collections and taxonomies"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: [02]
---

## What to build

Add `page_size` to collection and taxonomy config. During INDEX, listing virtual pages whose entry count exceeds `page_size` are split into N virtual pages: the base path serves page 1, subsequent pages at `{path}/page/{n}`. All generated pages flow through the existing collision re-check (AD-8 invariant). Templates receive a `.Paginator` with `CurrentPage`, `TotalPages`, `PrevURL`, `NextURL`, and the current page's refs.

End-to-end: a 25-entry collection with `page_size: 10` produces three listing pages with working prev/next links, byte-deterministic.

## Acceptance criteria

- [ ] `page_size` accepted on collections and taxonomies (term pages); validated (>0) in config
- [ ] Page 1 at the configured base path; pages 2..N at `{path}/page/{n}`; all collision-checked
- [ ] `.Paginator` available to listing layouts; absent (nil-safe) on non-paginated pages
- [ ] Entries within pages respect the existing `sort_by` ordering; pagination is deterministic across runs (`-count=3`)
- [ ] Integration fixture `testdata/pagination/` covering a paginated collection and a paginated taxonomy term, with golden output
- [ ] Docs: how-to "Paginate a collection" + config reference update

## Blocked by

- 02-site-pages-slim-refs (paginator pages carry refs in the same shape)
