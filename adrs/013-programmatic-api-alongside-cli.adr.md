---
title: A programmatic build API alongside the CLI
status: accepted
date_proposed: 2026-06-13
date_accepted: 2026-06-13
author: fabio
approvers: [fabio]
tags: [api, embedding, architecture]
affects:
  - engine/**
  - internal/serve/**
---

## Context

Tools built on Fuego — pack-based wrappers like `fuego-adr`, or a host that builds
on demand — shouldn't have to synthesize a temp `config.yaml` and shell into the
CLI. That path serializes config to disk, spawns a process, and parses stdout:
slow, fragile, and untestable.

## Decision

`engine.Build/Serve/Validate` plus `engine.BuildOptions` let a Go program build
**in-process**. `engine.Run(args)` (the CLI) reads a `config.yaml` file; the
programmatic API resolves config from in-memory layers (pack defaults → optional
file → option overrides) via `config.ResolveLayers`. The serve loop lives in
`internal/serve.Run`, so the CLI and `engine.Serve` share one watcher/rebuild/proxy
implementation and cannot drift.

## Consequences

- **+** Embedding Fuego is first-class — no temp files, no subprocess, fully typed.
- **+** One pipeline and one serve loop behind two front doors; the CLI and API
  can't diverge.
- **−** Two public entry points to keep coherent (`Run` vs `Build/Serve/Validate`),
  though they bottom out in the same pipeline.
- This is what makes pack-based tools ([011](011-format-packs-as-ecosystem-unit.adr.md))
  practical to build.
