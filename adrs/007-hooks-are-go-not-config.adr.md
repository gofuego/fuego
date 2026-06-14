---
title: Hooks are Go functions, not config
status: accepted
date_proposed: 2026-05-20
date_accepted: 2026-05-20
author: fabio
approvers: [fabio]
tags: [hooks, architecture, extensibility]
affects:
  - core/**
  - engine/**
  - internal/pipeline/**
---

## Context

Users need to intervene in the build — enrich pages, drop drafts, generate index
pages. The options are a config/shell-based hook mechanism (declarative, but
serializes data across a boundary) or Go functions registered on the engine
(typed, in-process).

## Decision

`AfterParse`, `Index`, and `BeforeRender` hooks are **Go functions** registered via
`eng.AfterParse(fn)` / `eng.Index(fn)` / `eng.BeforeRender(fn)`. They transform
typed `[]*core.Page`. The `Index` hook runs during INDEX (after ROUTE, before the
collision re-check) and is the supported place to add virtual pages — pages added
there are collision-checked, unlike pages injected in `BeforeRender`. The existing
`prebuild` config field stays shell-based because it runs before any pipeline data
exists — a fundamentally different concern.

## Consequences

- **+** Full type safety and no JSON round-trip; hooks operate directly on the
  page structs.
- **+** A clear contract for where to safely add routable pages (the `Index` hook).
- **−** Hooks require a Go build step; they are not editable as config. This is the
  point — transformations are code, declarative concerns (routes, taxonomies) are
  config.
- Operates on the mutable spine of [004](004-page-as-mutable-pipeline-spine.adr.md);
  packs register hooks the same way ([011](011-format-packs-as-ecosystem-unit.adr.md)).
