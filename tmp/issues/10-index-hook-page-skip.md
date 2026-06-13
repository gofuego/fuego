---
title: "Index hook + Page.Skip: first-class virtual pages through collision checks"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: []
---

## What to build

Add `core.IndexHook` (`func(pages []*Page) ([]*Page, error)`) registered via `eng.Index(fn)`, executed DURING INDEX alongside taxonomy/collection generation. Virtual pages returned by Index hooks flow through the normal URL resolution and the INDEX collision re-check — closing the hole where pages injected in BeforeRender (fuego-devops' diagram page) bypass collision detection entirely (AD-8 invariant). Also add `Page.Skip bool`: RENDER skips the page, it's excluded from `.Site.Pages` refs and the manifest.

## Acceptance criteria

- [ ] `IndexHook` type in `core/hooks.go`; `eng.Index()` threaded through engine → CLI → pipeline like existing hooks
- [ ] Pages added by Index hooks are collision-checked; a colliding virtual page is GlobalFatal with both claimants named
- [ ] `Page.Skip` honored by RENDER, manifest, and `.Site.Pages` (coordinate with #02)
- [ ] Integration fixture: Index hook injects a virtual page (graph-style, attribute-carrying nodes); second case proves a collision is caught
- [ ] BeforeRender continues to work unchanged (no breaking change); docs note steers virtual-page creation to the Index hook
- [ ] Docs: hooks page updated with the third hook point and Skip semantics

## Blocked by

None - can start immediately
