---
title: No built-in parsers; two-tier parser precedence
status: accepted
date_proposed: 2026-06-02
date_accepted: 2026-06-02
author: fabio
approvers: [fabio]
tags: [parsing, architecture]
affects:
  - internal/parse/**
  - parsers/**
---

## Context

A meta-engine that shipped Markdown as a built-in parser would, in practice, become
"the Markdown engine with extras" — the privileged format would attract features,
defaults, and assumptions. Yet users still need an easy on-ramp (config-only
formats) and a powerful one (full Go).

## Decision

The engine ships **zero built-in parsers**. Markdown is a first-party but **opt-in**
package (`parsers/markdown/`), registered like any other (`eng.Register(markdown.Parser())`).
Parser precedence is two-tier:

> compiled Go parsers **>** declarative regex parsers

Compiled parsers are the most powerful (arbitrary Go); declarative parsers are the
easiest (config-only regex rules). Registering a compiled parser for an extension
overrides a declarative one for the same extension. The merge happens once, during
INIT in `pipeline.RunUntil()`.

## Consequences

- **+** No format is privileged; Markdown is exactly as first-class as any custom
  format.
- **+** Two clear authoring tiers — config for simple line-based formats, Go for
  anything real.
- **−** Even a Markdown blog must register a parser explicitly; there is no
  zero-config default. We consider this honest rather than burdensome.
- Builds on the free-form AST of [001](001-universal-ast-free-form-nodes.adr.md).
