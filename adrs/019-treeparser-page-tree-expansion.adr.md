---
title: TreeParser expands one artifact into a tree of real pages
status: proposed
date_proposed: 2026-07-15
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

**Scope of this slice, and its forward amendments.** Two consumers of the parse
contract are only partly updated here; the full design is stated now and lands
in the next slice:

- **Manifest ([014](014-manifest-as-host-integration-contract.adr.md)).** The
  end state is that every page of a tree lists the **shared source file** as its
  `source_path` — a deliberate, versioned move to *multiple manifest entries per
  source*, after which fuego-studio's editability guard treats each child as
  editable-as-the-artifact. This slice does **not** implement that: a child's
  `source_path` is still derived from its own composite `RelPath`. Amending
  ADR-014 to the multi-entry contract is the next slice's work.
- **Build cache ([017](017-cache-stores-json-shaped-deep-copies.adr.md)).** The
  end state is that the cache stores *all* pages of a file under that file's
  content-hash entry, so an unchanged artifact skips its whole tree and a changed
  one reparses exactly its tree. This slice does **not** implement multi-page
  cache storage; a multi-page file has no single-entry representation yet, so
  **tree-parsed files are excluded from the cache** — reparsed every build, warned
  per file — which is the safe intermediate that keeps clean/incremental
  byte-equivalence green ([012](012-opt-in-incremental-builds.adr.md)). RENDER
  narrowing re-renders a whole tree when its artifact is reparsed. This exclusion
  is **superseded by the next slice's cache amendment**.

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
  (and the cache/manifest contracts it fed) must be revisited — done here for
  ROUTE/RENDER, deferred (and documented above) for cache/manifest.
- **−** Until the next slice, tree-parsed files pay a full reparse every build and
  their children's manifest `source_path` does not yet point at the artifact.
- Amends [004](004-page-as-mutable-pipeline-spine.adr.md) (parse contract) and,
  forward, [014](014-manifest-as-host-integration-contract.adr.md) and
  [017](017-cache-stores-json-shaped-deep-copies.adr.md); builds on the parser
  contract of [002](002-parser-owned-envelope-extraction.adr.md).
