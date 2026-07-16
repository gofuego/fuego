---
title: Specificity-ordered parser dispatch across a shared resolver
status: accepted
date_proposed: 2026-07-15
date_accepted: 2026-07-15
author: fabio
approvers: [fabio]
tags: [parsing, routing, architecture]
affects:
  - internal/dispatch/**
  - internal/discover/**
  - internal/parse/**
---

## Context

[005](005-extension-filename-parser-dispatch.adr.md) settled *what* drives
dispatch — a file's extension (matching `Parser.Type()`) or filename (matching
`FilenameParser.Filenames()`), never a frontmatter `type`. It left the ordering
between those two claim kinds unspecified, which was fine while every parser
owned a distinct extension.

The fuego-formats work breaks that assumption. A single site now registers many
parsers whose claims overlap: a markdown parser claims `md`, and an ADR parser
claims the compound suffix `*.adr.md`. Under the old dispatch a file's bare
extension was checked *first*, so `guide.adr.md` matched `md` and rendered as
plain markdown — the more specific `*.adr.md` claim never got a look. A platform
engineer must also be able to point a brownfield parser at `*.api.yaml` without
it fighting a generic `yaml` claim.

Compounding this, the claim rule lived in **two** places — discovery
classification (`internal/discover`, deciding content vs. asset) and parse
dispatch (`internal/parse`, choosing the parser) — each with its own extension
-then-filename logic. Two copies of a rule that must agree is a latent drift bug:
a file could classify as content under one and dispatch differently under the
other.

## Decision

The claim rule is extracted into one resolver module, `internal/dispatch`, that
both DISCOVER and PARSE consume. Given the registered parser set and a file's
base name it returns the winning parser type, or none (→ asset). DISCOVER records
that type on the file entry; PARSE dispatches by that exact value, so a file is
always parsed by the parser that classified it as content.

Resolution is **specificity-ordered**:

1. **Filename patterns are checked before bare extensions.** `*.adr.md` beats
   `md`.
2. **Among matching patterns, the longest pattern string wins** — `*.adr.md`
   (8) beats `*.md` (4) beats `*` (1). Length is the specificity proxy.
3. **Ties (equal-length patterns matching the same file) resolve by existing
   parser precedence** — user-registered > later pack > earlier pack >
   declarative — the same order the engine already uses to merge same-type
   parsers.
4. If no pattern matches, the bare extension is looked up; an extension maps to
   exactly one registered parser, so extension claims never tie.

A parser claims by exactly one kind. A parser that declares filename patterns
(`FilenameParser` with a non-empty `Filenames()`) claims **exactly those
patterns** — its `Type()` is not implicitly claimed as an extension. A parser
without patterns claims its `Type()` as a bare extension. This makes claim
overrides total: a markdown parser re-claimed as `README.md` claims only
`README.md`, instead of silently keeping every `.md` file through its `md`
type.

`page.Type` stays the matched parser's `Type()` (`adr` for a `*.adr.md` file);
`page.Ext` stays the literal extension (`md`). No content sniffing is
introduced — dispatch remains a pure function of the file name.

This **amends** [005](005-extension-filename-parser-dispatch.adr.md) rather than
superseding it: extension-or-filename dispatch, and the rejection of
frontmatter-`type` dispatch, still hold. Only the ordering between the two claim
kinds — previously extension-first, now pattern-first-by-specificity — changes.

## Consequences

- **+** Many formats coexist in one site without shadowing: `*.adr.md` never
  silently renders as markdown, and a parser can be adopted against non-default
  suffixes (`*.api.yaml`) without renaming files.
- **+** One resolver means discovery classification and parse dispatch cannot
  diverge — a file classified as content is dispatched by the identical rule.
- **+** The ordering is total and deterministic (length, then precedence, then
  the extension map), so dispatch is reproducible build-to-build.
- **−** A parser author must now understand specificity: a broad pattern (`*`)
  is silently outranked by any longer one, which is intended but can surprise.
- **−** Precedence tie-breaking relies on the engine's registration order,
  which is invisible in the filesystem; sites that need a specific winner
  between two equal-length patterns must control registration order.
- Behavior is unchanged for every existing site — none registers overlapping
  claims today — and is locked in by the `dispatch-specificity` golden fixture
  and the resolver unit matrix, in the discipline of
  [009](009-golden-file-determinism-testing.adr.md).
