---
title: "Port fuego-adr to a published format pack (validation)"
type: HITL
milestone: 0.3.1
labels: [needs-triage]
blocked_by: [09, 10]
---

## What to build

Restructure fuego-adr (separate repo) so its parser, hooks, embedded Tailwind theme, and config defaults ship as a `core.Pack` consumable via `eng.Use(adr.Pack())`, while the `fuego-adr` CLI keeps working as today (the CLI becomes a thin wrapper over its own pack). Publish the pack so it is `go get`-able — this is the first real pack and the prerequisite for `fuego init --pack` (#11).

HITL: involves product decisions in the fuego-adr repo (public API of the pack, what stays CLI-only, versioning).

## Acceptance criteria

- [ ] `adr.Pack()` exposes parser + theme FS + hooks + config defaults through the v0.3 pack API; no fuego core forks/patches
- [ ] `fuego-adr` CLI behavior unchanged for existing users (golden output comparison against pre-port build)
- [ ] A vanilla fuego project with only `eng.Use(adr.Pack())` + content renders a working ADR site; `packs.adr:` config honored via deep merge
- [ ] Friction log filed: every awkward point in the pack API encountered during the port becomes an issue BEFORE v0.3.1 tags
- [ ] Pack module published/taggable so `go get` resolves it

## Blocked by

- 09-config-deep-merge-provenance
- 10-index-hook-page-skip
