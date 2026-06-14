---
title: Parser dispatch by extension or filename, not frontmatter type
status: accepted
date_proposed: 2026-06-02
date_accepted: 2026-06-02
author: fabio
approvers: [fabio]
tags: [parsing, routing, architecture]
affects:
  - internal/parse/**
  - internal/discover/**
---

## Context

Something must decide which parser handles a file. A natural-seeming option is a
`type:` field in the file's frontmatter. But with parsers now owning envelope
extraction (see [002](002-parser-owned-envelope-extraction.adr.md)), the envelope
isn't available *before* a parser runs — so `type`-based dispatch would require the
engine to pre-parse, reintroducing exactly the format assumption we removed.

## Decision

Parser dispatch is determined by **file extension** (matching `Parser.Type()`) or
**filename** (matching `FilenameParser.Filenames()`, for extensionless files like
`Dockerfile`). The `type` frontmatter field does not drive dispatch. `layout` and
`slug` remain envelope conventions read *after* parsing.

## Consequences

- **+** Dispatch needs no pre-parsing and is visible in the filesystem — you can see
  which parser handles a file from its name.
- **+** Extensionless files are handled cleanly via `FilenameParser`.
- **−** Two files of the same logical format must share an extension; you can't mix
  formats under one extension via a `type` switch. This is a fair trade for not
  pre-parsing.
