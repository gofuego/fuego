---
title: core.Page is the mutable pipeline spine
status: accepted
date_proposed: 2026-05-20
date_accepted: 2026-05-20
author: fabio
approvers: [fabio]
tags: [architecture, core, pipeline]
affects:
  - core/**
  - internal/pipeline/**
---

## Context

The build is a sequence of phases (discover, parse, route, index, render). Each
needs the prior phase's output plus a little more. The choice is between threading
an immutable value that's copied and re-shaped at every phase, or a single mutable
struct that each phase enriches in place.

## Decision

`core.Page` is a **mutable pointer struct** that flows through every phase. Each
phase only adds to it: DISCOVER sets paths, PARSE sets envelope/nodes, ROUTE sets
the URL, INDEX may append whole virtual pages. The struct lives in `core/` (not
`internal/parse`) so the public hook API can reference it — it was originally
`parse.PageData` and was renamed when hooks were introduced.

## Consequences

- **+** No copying between phases; the pipeline is strictly linear and each phase is
  "page so far → page slightly more complete."
- **+** Placing `Page` in `core/` lets the public hook API hand user code a
  `*core.Page` while the internals mutate it, with no import cycle.
- **−** Mutability means phase order matters and a phase can observe partial state;
  the linear pipeline and phase enum keep this disciplined.
- Enables the hook model of [007](007-hooks-are-go-not-config.adr.md) and the host
  contract of [014](014-manifest-as-host-integration-contract.adr.md).
