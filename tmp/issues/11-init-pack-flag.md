---
title: "fuego init --pack <module>: wire-only pack scaffolding"
type: AFK
milestone: 0.3.1
labels: [needs-triage]
blocked_by: [09, 15]
---

## What to build

`fuego init mysite --pack github.com/gofuego/fuego-adr` scaffolds the generic skeleton, then writes a main.go importing the pack module and calling `eng.Use(pkg.Pack())`, runs `go mod init` + `go get <module>`. Convention: a pack module's root package exports `Pack() core.Pack`. **Init never executes third-party code** — pack config defaults flow at runtime via deep merge (#09), so no config writing is needed; `fuego config` shows the resolved result. Richer pack-provided sample content is explicitly out of scope (future `scaffold` subcommand on the user's own binary).

## Acceptance criteria

- [ ] `--pack` accepts a module path; generated main.go compiles with the import + `eng.Use` wiring (last path segment → package name, with `--pack-symbol` escape hatch for mismatches)
- [ ] `go mod init` + `go get` run with clear errors surfaced verbatim on failure (no network retry magic)
- [ ] No third-party code is compiled or executed by init itself (test asserts no `go run`/`go build` of the pack)
- [ ] End-to-end test against the published fuego-adr pack (#15): init → build produces a working site
- [ ] Generated CLAUDE.md and README mention the pack and point at `fuego config` for effective config
- [ ] Docs: getting-started gains the `--pack` path; `Pack()` convention documented in the pack tutorial

## Blocked by

- 09-config-deep-merge-provenance
- 15-port-fuego-adr-to-pack (needs a published pack to test against)
