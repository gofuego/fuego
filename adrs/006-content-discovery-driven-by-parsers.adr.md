---
title: Content discovery is driven by registered parsers
status: accepted
date_proposed: 2026-06-02
date_accepted: 2026-06-02
author: fabio
approvers: [fabio]
tags: [discovery, parsing, architecture]
affects:
  - internal/discover/**
---

## Context

DISCOVER must classify each file as content (to be parsed) or a static asset (to be
copied). A hardcoded list of content extensions (`.md`, `.html`, …) is the obvious
approach, but in an engine with no built-in parsers it creates a contradiction: a
`.md` file would be "content" with no parser able to handle it.

## Decision

A file is classified as content **if and only if** a registered parser matches its
extension or filename. There are no hardcoded content extensions. Parsers are the
single source of truth for what is content; everything else is an asset.

## Consequences

- **+** No contradiction is possible — the thing that says "this is content" is the
  same thing that can parse it.
- **+** Registering a Markdown parser is what makes `.md` files appear as content;
  remove it and they become plain assets.
- **−** Forgetting to register a parser silently turns its files into copied assets
  rather than erroring. `fuego list`/`validate` make this visible.
- Completes the parser-as-authority trio with
  [002](002-parser-owned-envelope-extraction.adr.md) and
  [005](005-extension-filename-parser-dispatch.adr.md).
