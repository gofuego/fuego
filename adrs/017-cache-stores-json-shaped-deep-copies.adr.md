---
title: The build cache stores JSON-shaped deep copies, degrading per page
status: accepted
date_proposed: 2026-07-02
date_accepted: 2026-07-02
author: fabio
approvers: [fabio]
tags: [incremental, cache, packs]
affects:
  - internal/buildcache/**
  - internal/parse/**
---

## Context

The cache's `ParsedPage` used to hold the *same* envelope and node references as
the live `core.Page`, and the cache is saved at the end of the build — after
every hook has mutated those pages in place. So the cache actually serialized
post-hook state: hook-built values (`[]map[string]any` catalogs, backlink
lists) reached the gob encoder unregistered and failed the whole cache write,
and on a hit, restored pages carried the *previous* build's hook products, with
output correctness resting on hooks happening to be idempotent. Separately,
envelopes are `map[string]any` — parsers may store any Go type in them, but gob
can only encode dynamic types that were registered, and a pack's private type
can never be registered by the engine.

## Decision

The cache stores **post-PARSE state only, by deep copy**: both cache boundaries
(snapshot after parsing, restore on a hit) copy the envelope and node tree, so
hook mutation never reaches the cache and stale hook output never leaves it.

Cacheable envelope values are **JSON-shaped**: the registered set is
`map[string]any`, `[]any`, `[]map[string]any`, `map[string]string`,
`[]map[string]string`, `[]string`, `[]int`, `[]float64`, `[]bool`, scalars, and
`time.Time`. A page whose envelope holds any other concrete type is **dropped
from the cache individually** — a permanent per-page miss, reported as a
warning naming the page — never a build error, and never a failure of the whole
cache write.

## Consequences

- **+** The cache honors its post-PARSE contract structurally; non-idempotent
  hooks are safe (locked in by `TestIncrementalCacheIsHookIsolated`).
- **+** One exotic page costs itself cache reuse, not the whole site's.
- **−** Parsers that store custom structs or pointers in envelopes silently
  lose caching for their pages; conforming (as fuego-dotclaude did) means
  shaping display data as JSON-shaped maps rather than named types.
- **−** Snapshot and restore pay a deep-copy per page; envelope/node values
  outside the JSON-shaped set are shared by reference, so mutating *those* in a
  hook is still unsafe — the contract, not the copy, is the guarantee.
- Extends the miss-never-error posture of
  [012](012-opt-in-incremental-builds.adr.md) from whole-cache to per-page.
