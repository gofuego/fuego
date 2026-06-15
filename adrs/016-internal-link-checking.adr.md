---
title: Internal link checking resolves against the built output
status: accepted
date_proposed: 2026-06-15
date_accepted: 2026-06-15
author: fabio
approvers: [fabio]
tags: [render, quality, reliability]
affects:
  - internal/linkcheck/**
  - internal/pipeline/**
---

## Context

Fuego does not rewrite links in content — a Markdown `[x](/foo/)` renders verbatim.
Under a deployment base URL with `<base href>`, that means an author can write a
link that resolves nowhere (dangling) or escapes the deployment base (a
root-absolute `/foo/` served under `/owner/repo/`). These failures are invisible
until a reader hits a 404, with no hint which file produced the link.

## Decision

`fuego build --check-links` runs a checker after the output is written. It resolves
every `<a href>` **exactly as a browser would** — honoring each page's `<base href>`
and the site base URL — and verifies the result lands on a path that actually
exists in the output. It validates the final HTML as shipped, so it catches
dangling *and* base-escaping links from content or templates alike, without
modelling the engine's rendering rules. Broken links are reported against the
source page (its `RelPath`) with the href and the resolved target. They are
warnings by default; `--strict-links` promotes them to a fatal error for CI.
Anchors and external links are out of scope (network-bound and false-positive
prone).

## Consequences

- **+** Broken internal links are caught at build time and reported against the
  file to edit — far more traceable than a runtime 404.
- **+** Cheap: it's hash lookups over HTML already in memory; the slow, flaky part
  (external links) is excluded by design.
- **+** Most valuable when run with the real `--base-url`, where it catches the
  base-escape class that a root build cannot see.
- **−** Template-generated links can only be attributed to the page they appear on,
  not the template line.
- **−** Regex href extraction is adequate for generated HTML but not a full HTML
  parser; pathological markup could be missed.
- Pairs with [015](015-index-files-route-to-directory-root.adr.md) and reinforces
  the determinism discipline of [009](009-golden-file-determinism-testing.adr.md).
