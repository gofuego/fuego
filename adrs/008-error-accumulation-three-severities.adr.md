---
title: Error accumulation with three severities, not fail-fast
status: accepted
date_proposed: 2026-05-20
date_accepted: 2026-05-20
author: fabio
approvers: [fabio]
tags: [errors, architecture, reliability]
affects:
  - core/**
  - internal/pipeline/**
---

## Context

A site can have hundreds of pages. If the build failed completely on the first bad
file, one typo would block the whole site — painful during authoring. But some
errors (a URL collision, a broken config) genuinely do compromise the entire
output and must stop the build.

## Decision

The pipeline accumulates errors in a mutex-protected `core.ErrorAccumulator` with
three severities:

- **Warning** — logged; build continues.
- **LocalFatal** — the offending page is skipped; build continues with partial
  output.
- **GlobalFatal** — build halts (URL collisions, config errors), because continuing
  would produce corrupt output.

## Consequences

- **+** One malformed file can't sink a large site; the build produces partial,
  useful output.
- **+** Structural problems still halt, so corrupt output is never emitted.
- **−** Authors must read accumulated warnings/errors rather than getting a single
  hard stop; `fuego validate` surfaces them without rendering.
- The severity choice is about blast radius, and the collision re-check of
  [010](010-virtual-pages-for-taxonomies-collections.adr.md) raises GlobalFatal.
