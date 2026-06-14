---
title: A universal AST with free-form node types
status: accepted
date_proposed: 2026-05-20
date_accepted: 2026-05-20
author: fabio
approvers: [fabio]
tags: [architecture, core, parsing]
affects:
  - core/**
  - internal/render/**
---

## Context

Fuego is a meta-engine: users define arbitrary content formats (`.trivia`, `.card`,
`.pitch`, anything) and map them to HTML. The central question is what data model
sits between a parser and a template. A fixed AST schema — `heading`, `paragraph`,
`list` — would be familiar and let the engine render sensible defaults. But it
would also bake in a Markdown-shaped worldview and quietly privilege one format,
defeating the entire premise.

## Decision

All parsers produce `[]core.Node` where `Type` is a **free-form string**. The
engine never interprets node types — templates decide rendering. A trivia parser
emits `question`/`answer` nodes; a flashcard parser emits `front`/`back`. The
default renderer wraps each node as `<div data-type="{Type}">`, and production
sites override per type via `theme/renderers/{type}.html`.

A node may set `Raw: true` to pass its `Content` through as raw HTML, unwrapped and
unescaped — any parser can use it (the Markdown parser emits one raw node rather
than modelling every heading as a typed node).

## Consequences

- **+** Any content format is expressible; the engine has zero format opinions.
- **+** Rendering is fully owned by templates — the engine never grows
  node-type-specific branches.
- **−** The default rendering is generic and ugly; real sites must supply renderer
  templates. We accept this — out-of-the-box polish is impossible for an engine
  that doesn't know what your content *is*.
- This is the invariant the rest of the architecture protects; see
  [003](003-no-built-in-parsers-two-tier-precedence.adr.md) and
  [006](006-content-discovery-driven-by-parsers.adr.md).
