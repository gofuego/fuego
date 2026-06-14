---
title: Format packs bundle a content format as one installable unit
status: accepted
date_proposed: 2026-06-13
date_accepted: 2026-06-13
author: fabio
approvers: [fabio]
tags: [packs, architecture, extensibility]
affects:
  - core/**
  - internal/render/**
  - internal/config/**
---

## Context

Parsers, hooks, themes, and config defaults for a domain (ADRs, Kubernetes
manifests) are useless scattered. For an ecosystem to exist, a format needs to ship
as **one thing** a vanilla project consumes with a single line — without dragging
the whole engine into the pack author's dependency graph.

## Decision

`core.Pack` bundles parsers, hooks, an embedded theme FS (`base.html`, `layouts/`,
`renderers/`, `partials/`, and a `static/` subtree), and a `ConfigDefaults` YAML
fragment, registered via `eng.Use(pack)`. An optional `Init(ctx, *PackContext)`
reads the pack's `packs.{name}` config subtree. `core.Pack` stays in `core/`
(stdlib-only) so a pack imports `core` without the engine. Precedence: user
parsers and user theme files always win over packs; among packs, later registration
wins (with a warning). Pack `ConfigDefaults` deep-merge under the user config;
pack `static/` is copied during STATIC; the theme FS layers under the user theme
dir in `render.LoadTemplates`.

## Consequences

- **+** A domain-specific SSG ships as an importable Go module (e.g. `fuego-adr`)
  consumed in one line.
- **+** Packs inject into the normal phases — no pack-specific branching in the
  engine.
- **+** Clear, debuggable precedence (user > later pack > earlier pack), with
  provenance visible via `fuego config`.
- **−** More resolution machinery (deep merge, theme layering, the Init lifecycle)
  than a single-project engine would need.
