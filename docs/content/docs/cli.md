---
title: CLI Reference
layout: doc
nav_section: "Reference"
nav_weight: 2
tags:
  - reference
  - cli
---

## Install

```bash
go install github.com/FabioSol/fuego/cmd/fuego@latest
```

Requires Go 1.23+. The binary is placed in `$GOPATH/bin` (usually `~/go/bin`). Ensure this directory is in your `PATH`.

Alternatively, run any command without installing:

```bash
go run github.com/FabioSol/fuego/cmd/fuego@latest <command>
```

## Commands

### build

Build the static site. Runs the full pipeline and writes output to `build/`.

```bash
fuego build
```

### serve

Start a dev server with file watching and live rebuild.

```bash
fuego serve
```

Watches `content/` and `theme/` for changes. When a file changes, the site is rebuilt and served at `http://localhost:8080` (configurable via `dev.port` in config).

### validate

Check config and content for errors without producing output. Useful as a CI gate.

```bash
fuego validate
```

Runs the pipeline through INDEX (discovery, parsing, routing, collision detection) without rendering. Exit code 0 on success, 1 on any error.

### list

Print all pages as a table of TYPE, SOURCE, and URL.

```bash
fuego list
```

### config

Print the fully resolved configuration — your `config.yaml` deep-merged with every registered pack's defaults — annotated with per-key provenance (`# user` or `# pack: name`).

```bash
fuego config
```

Useful for answering "why is this value what it is?" when format packs contribute config defaults. Output is deterministic, so it is safe to diff. See [Config Merging](/docs/config-merging/).

### init

Scaffold a new Fuego project.

```bash
fuego init mysite
```

Creates a working project with a `.card` flashcard DSL, theme, and sample content.

## Global Flags

| Flag | Default | Description |
|---|---|---|
| `--config` | `config.yaml` | Path to configuration file |

## Error Handling

Three severity levels:

- **Warning** — logged, build continues
- **LocalFatal** — page skipped, build continues
- **GlobalFatal** — build fails immediately

`validate` catches config errors, parse failures, and URL collisions before you build.
