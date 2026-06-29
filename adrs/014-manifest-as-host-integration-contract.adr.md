---
title: The site manifest is the host-integration contract
status: accepted
date_proposed: 2026-06-13
date_accepted: 2026-06-13
author: fabio
approvers: [fabio]
tags: [manifest, integration, architecture]
affects:
  - internal/manifest/**
---

## Context

A host like fuego-studio serves a built site and, for editing, needs to map a
rendered page back to its source file in the repo. The engine knows the build facts
(where content came from, where output went); the host knows policy (who may edit,
which branch). The question is what the engine exposes, and where the line between
"build fact" and "workflow policy" sits.

## Decision

`internal/manifest` writes `site-manifest.json` with, per page, `url`, `type`,
`layout`, `title`, `summary`, `output_path` (`<url>/index.html`), `source_path`
(a real page's content-dir-relative `RelPath`; **omitted** for virtual pages,
whose internal `_virtual/...` `RelPath` is not a real file — so a host treats
them as non-editable), and the flattened `envelope`. The top-level `content_root` is the
content dir relative to the enclosing git root, so a host maps a page's repo source
as `content_root` + `source_path`. The `build`/`serve` `--base-url` flag overrides
`site.base_url` for one run, so a deploy can target a subpath. The engine exposes
**build facts only** — editing policy lives in the host.

## Consequences

- **+** A host can fetch and commit any page's source unambiguously, including for
  sites built from a subdirectory.
- **+** Virtual pages are marked non-editable by an omitted `source_path`.
- **+** `--base-url` lets one repo serve different mount paths (e.g. GitHub Pages and
  a studio mount) without a separate config file.
- **−** The manifest is a public contract — changing a field is a breaking change for
  hosts that read it.
- Keeps the engine editing-agnostic; the manifest carries facts, the host owns
  policy.
