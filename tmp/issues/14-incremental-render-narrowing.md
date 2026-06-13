---
title: "Incremental render narrowing: affected-set re-render"
type: AFK
milestone: 0.3.1
labels: [needs-triage]
blocked_by: [13]
---

## What to build

Replace #13's conservative re-render-everything with an affected set: (changed pages) ∪ (all virtual pages — listings/paginators/term pages can reflect any content change) ∪ (pages whose layout's parse tree references `.Site` — detected once at template load, same mechanism as the `.JSON` walk in #06). Unchanged pages whose layouts are site-blind are not re-rendered; their outputs are left in place.

The #13 byte-equivalence CI suite re-runs unchanged against the narrowed path — it is the proof this optimization is safe.

## Acceptance criteria

- [ ] `.Site` reference detection per layout cached at LoadTemplates (shared with #06 infrastructure)
- [ ] Affected-set computation covers: changed/added pages, all virtual pages, `.Site`-referencing layouts' pages
- [ ] Full equivalence suite from #13 passes byte-identical on the narrowed path for every mutation class
- [ ] Benchmark (#12): single-file edit on the 10k-page site re-renders a measured small fraction; number recorded in README's incremental section
- [ ] Envelope-only edit (no content change) still propagates to nav menus everywhere (covered by the `.Site`-layout rule; explicit test)

## Blocked by

- 13-incremental-builds-core
