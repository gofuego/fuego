---
title: The Build Pipeline
layout: doc
tags:
  - concepts
  - architecture
---

Every Fuego command runs the same pipeline. Understanding the phases helps you predict behavior, debug issues, and write effective hooks.

## Phase Diagram

```
PREBUILD       →  shell hook (npm, tailwind, etc.)
INIT           →  merge built-in + declarative + compiled parsers
DISCOVER       →  walk content dir, apply ignore patterns
PARSE          →  split frontmatter, run parsers (concurrent)
  ↓ AfterParse hooks
ROUTE          →  resolve URLs, detect collisions
INDEX          →  build taxonomies + collections, re-check collisions
  ↓ BeforeRender hooks
RENDER         →  execute templates (concurrent)
MANIFEST       →  write site-manifest.json
STATIC         →  copy public/ and colocated assets
```

## Phase Details

### PREBUILD

Runs the shell command from `config.yaml`'s `prebuild` field. This is for external tooling — Tailwind CSS compilation, npm scripts, asset preprocessing. Runs before any Fuego logic.

### INIT

Merges three parser sources in priority order:

1. **Built-in** (Markdown with GFM) — lowest priority
2. **Declarative** (regex rules from config) — overrides built-in
3. **Compiled** (Go code via `eng.Register()`) — highest priority

If two parsers target the same file extension, the higher-priority one wins.

### DISCOVER

Walks the `content/` directory and classifies each file:

- **Content files** — matched to a parser by extension (`.md`, `.trivia`, `.card`, etc.)
- **Binary assets** — images, fonts, etc. — copied to output alongside their routed content
- **Ignored files** — matched by `ignore` glob patterns in config

### PARSE

For each content file, in parallel:

1. Split the file at `---` delimiters to extract YAML frontmatter (the envelope) and the raw payload
2. Dispatch the payload to the matching parser
3. The parser returns `[]Node` — the universal AST

Concurrency is bounded by `runtime.NumCPU()` via `errgroup`.

### ROUTE

Assigns a URL to each page using three-tier priority:

1. Frontmatter `slug` field — overrides the slug segment
2. Config `routes[type]` pattern — pattern expansion with `{dir}`, `{slug}`, `{filename}`
3. Filesystem mirror — the default

After resolution, checks for URL collisions. A collision is a `GlobalFatal` error that stops the build.

### INDEX

Generates virtual pages for taxonomies and collections:

- **Taxonomy term pages** — one per unique term value (e.g., `/tags/go/`)
- **Taxonomy index pages** — list all terms (e.g., `/tags/`)
- **Collection pages** — glob-matched, sorted listing pages

Virtual pages are appended to the page list. Collision detection runs again to catch conflicts between virtual and real pages.

### RENDER

For each page, in parallel:

1. Pre-render nodes to HTML using the default renderer (or per-type renderer templates)
2. Build the template data (`.Page`, `.Site`, `.JSON`)
3. Execute the base template with the selected layout
4. Write `{url}/index.html` to the output directory

### MANIFEST

Writes `site-manifest.json` — a JSON index of all pages, taxonomy terms, and collection membership. Useful for client-side search, navigation, or API-like access to site data.

### STATIC

Copies `public/` contents to the output root and colocated binary assets to their routed paths.

## Partial Execution

`validate` runs through INDEX without RENDER — catching config errors, parse failures, and collisions without producing output. `list` runs through ROUTE and prints the page table. This is controlled by `pipeline.RunUntil(phase)`.
