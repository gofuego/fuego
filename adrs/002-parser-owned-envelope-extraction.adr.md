---
title: Parsers own envelope extraction
status: accepted
date_proposed: 2026-06-02
date_accepted: 2026-06-02
author: fabio
approvers: [fabio]
tags: [parsing, core, architecture]
affects:
  - core/**
  - internal/parse/**
---

## Context

Content metadata arrives in different shapes: Markdown blogs use YAML frontmatter,
Kubernetes manifests embed metadata inside the body, Dockerfiles have none at all.
An earlier design had the engine strip YAML frontmatter before handing the body to
a parser — which silently assumed every format is frontmatter-shaped and made
non-frontmatter formats second-class.

## Decision

The `Parser` interface is `Parse(raw []byte) (Envelope, []Node, error)`: the parser
receives the **entire raw file** and returns both the metadata envelope and the
parsed nodes. The engine does not pre-process the bytes. Convenience wrappers cover
common shapes — `core.WithYAMLFrontmatter`, `core.WithNoEnvelope` — and
`core.SplitFrontmatter()` is a public helper for parsers that want YAML
frontmatter. The parser knows its format best, so the parser decides.

## Consequences

- **+** Any metadata convention is supported; no format is privileged.
- **+** The engine's pre-parse stage disappears — one less place to encode a format
  assumption.
- **−** Every parser must handle (or explicitly opt out of) envelope extraction;
  the wrappers exist to keep that boilerplate-free.
- Because the envelope isn't available until after a parser runs, dispatch cannot
  depend on it — see [005](005-extension-filename-parser-dispatch.adr.md).
