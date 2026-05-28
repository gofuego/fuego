---
title: CLI Reference
layout: doc
---

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
