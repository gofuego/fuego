---
title: Index files route to their directory root
status: accepted
date_proposed: 2026-06-15
date_accepted: 2026-06-15
author: fabio
approvers: [fabio]
tags: [routing, pipeline]
affects:
  - internal/route/**
---

## Context

In the filesystem-mirror routing tier, every file mapped to `/<dir>/<slug>/`,
including `content/index.md`, which became `/index/`. To serve a site's home at
`/`, projects shipped a `public/index.html` redirect to `/index/` — a hack that
every Fuego site (and the scaffold and theme packs) had to carry, and that broke
in subtle ways under a deployment base URL.

## Decision

A file named `index` is the **root of its directory** in the filesystem-mirror
tier: `content/index.md` → `/`, `content/blog/index.md` → `/blog/`. This matches
the universal SSG convention (Hugo, Eleventy, Docusaurus). The explicit `slug` and
config-route tiers are unchanged. The `public/index.html` redirect is removed from
the scaffold and theme packs, and a home link is now just the base URL root.

## Consequences

- **+** Home pages live at `/` with no redirect hack; the convention matches every
  other SSG, so it's unsurprising.
- **+** One fewer file for every site, scaffold, and pack to carry.
- **−** A redirect `public/index.html` is now *harmful* — copied during STATIC, it
  would clobber the rendered root `index.html`. The scaffold and packs dropped it,
  but existing sites that carry one must remove it.
- **−** Output paths changed for any site with index files (`/index/` → `/`); this
  is a behavior change that shifted golden fixtures and required a minor release.
- Builds on the three-tier routing of [004](004-page-as-mutable-pipeline-spine.adr.md);
  pairs with [016](016-internal-link-checking.adr.md), which catches links left
  pointing at the old `/index/`.
