---
title: Incremental builds are opt-in and provably equivalent
status: accepted
date_proposed: 2026-06-13
date_accepted: 2026-06-13
author: fabio
approvers: [fabio]
tags: [incremental, performance, reliability]
affects:
  - internal/buildcache/**
  - internal/pipeline/**
  - internal/render/**
---

## Context

Big-site dev rebuilds are the one place users feel a real performance gap. Caching
parsed pages and re-rendering only what changed would fix it — but an incremental
build that diverges from a clean build, even rarely, is worse than a slow one: it
erodes trust in every build.

## Decision

`Options.Incremental` (and the dev server, always) reuses parsed pages via
`internal/buildcache` and narrows RENDER to the affected set. Default `build` is
clean and cache-free. Safety is structural: the cache header pins the build
environment (engine binary hash + resolved config hash + theme tree hash), so any
change to those invalidates the cache and forces a full rebuild — a corrupt or
version-mismatched cache is a **miss, never an error**. RENDER narrowing (changed
pages + virtual pages + pages whose templates read `.Site.Pages`) is guarded by a
byte-equivalence suite that builds every fixture clean and then incrementally under
every mutation class.

## Consequences

- **+** Fast dev rebuilds without risking divergence; output stays byte-identical to
  a clean build.
- **+** Cache safety is by construction, not vigilance — the header makes stale
  reuse impossible.
- **−** A standing parity test suite must accompany every fixture, and new fixtures
  must round-trip through the cache cleanly.
- Relies on the determinism guarantees of
  [009](009-golden-file-determinism-testing.adr.md).
