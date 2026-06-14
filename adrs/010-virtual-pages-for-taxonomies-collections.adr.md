---
title: Taxonomies and collections are virtual pages
status: accepted
date_proposed: 2026-05-20
date_accepted: 2026-05-20
author: fabio
approvers: [fabio]
tags: [taxonomies, collections, pipeline, architecture]
affects:
  - internal/index/**
  - internal/render/**
---

## Context

Taxonomy term pages, taxonomy indexes, and collection listings have no source file
— they're derived from the set of real pages. They could be rendered by a special
code path that emits HTML directly, or modelled as ordinary pages that flow through
the normal renderer.

## Decision

These are generated as **virtual `core.Page` structs** appended to the page list
during INDEX, with special types (`taxonomy-term`, `taxonomy-index`, `collection`)
and an empty `RelPath`. They go through the **same RENDER phase** as real pages —
no special rendering path. Their nodes (`page-ref`, `term-ref`) carry metadata in
attributes, and templates render them freely. They are excluded from taxonomy term
scanning so they don't index themselves, and INDEX re-runs collision detection
after adding them, because a generated `/tags` can collide with a real page.

## Consequences

- **+** The pipeline stays uniform — RENDER never asks "is this page real?"
- **+** Templates fully control how generated pages look, consistent with
  [001](001-universal-ast-free-form-nodes.adr.md).
- **+** The empty `RelPath` is how the manifest marks them non-editable
  ([014](014-manifest-as-host-integration-contract.adr.md)).
- **−** Virtual pages can collide with real pages, so the post-INDEX collision
  re-check is mandatory; skipping it silently produces two pages at one path.
