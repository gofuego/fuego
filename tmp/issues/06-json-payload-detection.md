---
title: "Compute .JSON payload only for layouts that reference it"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: []
---

## What to build

At `LoadTemplates` time, walk each layout's (and base's) parse tree for a reference to `.JSON`. During RENDER, compute the JSON payload only for pages whose selected layout uses it. No config knob. Sites that hydrate (trivia, fuego-devops) see zero behavior change; docs-style sites silently drop the double serialization.

## Acceptance criteria

- [ ] Parse-tree walk detects `.JSON` in layouts, base, and templates they invoke (`template`/`block` nodes); detection result cached per layout
- [ ] Pages whose layout references `.JSON` render byte-identically to v0.2 output
- [ ] Pages whose layout does not reference it skip `JSONPayload` entirely (verified by benchmark delta in #12 fixture and absence in golden output)
- [ ] Existing integration fixtures regenerate with no diffs for `.JSON`-using fixtures
- [ ] Behavior documented on the templates reference page (explicitly, since it is implicit magic)

## Blocked by

None - can start immediately

## Measured (2026-06-13, M1 Pro, 10k-page bench)

Hydrated base (`{{.JSON}}` present): 4522 ms/build. Same site, no `.JSON` in theme: 4290 ms/build
(~5% faster — the JSON marshal cost, now skipped). Bench gained a `10000pages-nojson` variant.
