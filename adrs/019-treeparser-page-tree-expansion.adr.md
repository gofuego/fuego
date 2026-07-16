---
title: TreeParser expands one artifact into a tree of real pages
status: accepted
date_proposed: 2026-07-15
date_accepted: 2026-07-15
author: fabio
approvers: [fabio]
tags: [parsing, routing, core, architecture]
affects:
  - core/**
  - internal/parse/**
  - internal/route/**
  - internal/render/**
---

## Context

Fuego's parse contract is **one file → one page**: a `core.Parser` returns a
single `(Envelope, []Node)`, and [002](002-parser-owned-envelope-extraction.adr.md)
plus [004](004-page-as-mutable-pipeline-spine.adr.md) build the whole pipeline on
that shape — one page per discovered file, enriched in place through ROUTE and
INDEX. But a single rich artifact often *is* a whole section. An OpenAPI spec
deserves an index plus a page per operation, tag, and schema; a DBML schema, a
page per table; a Playwright suite, a page per test. Today the only way to make
those extra pages is the virtual-page workaround (append `*core.Page`s in an
INDEX hook, [010](010-virtual-pages-for-taxonomies-collections.adr.md)), and
those pages have three defects: taxonomies and collections can't see them (they
run over the real page set, and are re-scanned before the hook), incremental
builds re-render them constantly (they aggregate, so they're always "affected"),
and a host like fuego-studio can't map them back to an editable source. The
format packs that own this parsing logic (fuego-devops, fuego-adr) are also
locked to their own themes, so the parsing can't be reused without adopting a
site shape. The engine needs a first-class way for one artifact to become a
routed, taxonomy-visible, stably-URL'd tree of pages.

## Decision

A new **optional** interface alongside `Parser`, `core.TreeParser`, returns a
`core.PageTree`: a root (`Envelope` + `Nodes`) plus `Children` keyed by relative
slug path, nested arbitrarily. The engine detects it by a plain **interface
assertion at PARSE** — there is no registration change, and a parser that does
not implement it is completely untouched. When a parser implements `TreeParser`,
the engine calls `ParseTree` (not `Parse`) and expands the returned tree into
**real `core.Page`s**:

- **Root** — the tree's own `Envelope`/`Nodes` become the source file's routed
  page, exactly as an ordinary parser's output would.
- **Child identity** — a child's `RelPath` is the source file's `RelPath` joined
  with its slug path (e.g. `api.openapi.yaml` + `tags/billing/get-invoice`), so a
  child never collides with a plain page unless intended. The child's
  `SourcePath` is the artifact itself.
- **Child URL** — composed in a **second ROUTE pass**: the root is resolved by
  the normal three-tier routing first (frontmatter slug → route pattern →
  filesystem mirror, honoring the index-file convention of
  [015](015-index-files-route-to-directory-root.adr.md)), then each child's URL is
  the root's resolved URL joined with its slug-path segments. The whole tree
  therefore hangs beneath wherever the root routes.
- **Envelopes** — each child carries its own envelope and nodes, so children flow
  through INDEX as ordinary pages and are seen **natively** by taxonomies,
  collections, and pagination.

Sibling-slug collisions inside one tree (two children whose slug paths compose to
the same URL) and collisions between a tree page and any other page surface
through the **existing** ROUTE/INDEX collision detection as a `GlobalFatal`
([008](008-error-accumulation-three-severities.adr.md)) — there is no
tree-local collision check.

This **retires "one file = one page" as a parse-contract invariant**, amending
[004](004-page-as-mutable-pipeline-spine.adr.md): PARSE may now emit several
pages for one file, and the mutable-spine model extends to each of them.

**Envelope convention for library tree parsers** (the engine stays agnostic):
envelope values are JSON-shaped so tree pages stay cache-eligible under
[017](017-cache-stores-json-shaped-deep-copies.adr.md); a missing per-child
layout falls back to the base template silently; parsers never emit slugs or
routes.

**Host-facing surfaces, and the decisions they amend.** Two consumers of the
parse contract are extended to the multi-page shape:

- **Manifest ([014](014-manifest-as-host-integration-contract.adr.md)).** Every
  page of a tree lists the **shared root artifact's `RelPath`** as its
  `source_path` — a deliberate, versioned move to *multiple manifest entries per
  source*, after which fuego-studio's `SourcePath != ""` editability guard treats
  each child as editable-as-the-artifact (clicking edit on an operation page opens
  the spec that defines it). Root and children stay distinguishable by their
  differing `url`/`output_path`; sorted-by-URL determinism is unchanged. This
  amends ADR-014's single-entry-per-source assumption.
- **Build cache ([017](017-cache-stores-json-shaped-deep-copies.adr.md)).** The
  cache stores *all* pages of a file under that file's content-hash entry
  (`ParsedPage.Tree`), so an unchanged artifact restores its whole tree from cache
  and a changed one reparses and re-renders exactly its tree; RENDER narrowing
  re-renders a whole tree when (and only when) its artifact is reparsed, so an
  edit to an unrelated file leaves the tree untouched. Deep-copy isolation holds
  at both boundaries for every page of the tree. Degradation is per ENTRY: an
  ordinary file degrades per page as ADR-017 specifies, but a tree with any
  non-JSON-shaped child envelope drops the whole file's entry — a missing child on
  a hit would silently change the output, so per-page degradation is structurally
  impossible for a tree — a warning, never an error. The cache header version is
  bumped, so an older cache is a miss, never a decode error. Clean/incremental
  byte-equivalence stays green ([012](012-opt-in-incremental-builds.adr.md)).
  This amends ADR-017's per-page-only degradation and single-page-per-entry
  storage.

## Consequences

- **+** One artifact becomes a routed section: an index plus a tree of real pages
  with stable URLs, visible to taxonomies/collections/pagination like any
  hand-written page.
- **+** Purely additive: `TreeParser` is optional, detected by interface
  assertion; existing parsers, fixtures, and the registration API are unchanged.
- **+** Collisions reuse the one existing detector, so a duplicate operation slug
  or a child shadowing a real page fails the build the same way any URL clash
  does.
- **−** PARSE is no longer one-page-per-file; code that assumed that invariant
  (and the cache/manifest contracts it fed) had to be revisited — done here for
  ROUTE/RENDER, the build cache, and the manifest.
- **−** A tree is cached and restored as a unit under one content hash, so one
  child whose envelope holds a non-JSON-shaped value costs the whole artifact its
  cache reuse (not just that child) — the price of storing a multi-page file
  under a single entry.
- Amends [004](004-page-as-mutable-pipeline-spine.adr.md) (parse contract),
  [014](014-manifest-as-host-integration-contract.adr.md) (multiple manifest
  entries per source), and
  [017](017-cache-stores-json-shaped-deep-copies.adr.md) (multi-page cache
  entries, per-entry degradation for trees); builds on the parser contract of
  [002](002-parser-owned-envelope-extraction.adr.md).
