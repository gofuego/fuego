---
title: Embedding Fuego
layout: doc
nav_section: "Concepts"
nav_weight: 4
tags:
  - concepts
  - go
  - packs
---

Fuego isn't only a CLI — it's a library for **building your own static site
generator**. A domain-specific tool (an ADR site builder, an infra-diagram
generator, a changelog publisher) is typically a [format pack](/docs/concepts/format-packs/)
plus a thin program that drives the engine in-process. [fuego-adr](https://github.com/FabioSol/fuego-adr)
is exactly this shape.

## Two ways to drive the engine

```go
eng := engine.New()
eng.Register(markdown.Parser())
eng.Use(mypack.Pack())
```

From there you either:

- **`eng.Run(os.Args)`** — hand control to Fuego's CLI (`build`, `serve`,
  `config`, …), which reads a `config.yaml` from disk. This is what a scaffolded
  `main.go` does.
- **The programmatic API** — `eng.Build`, `eng.Serve`, `eng.Validate` — build
  in-process with configuration supplied in Go. This is what a domain-specific
  tool with its own CLI uses, so it never has to synthesize a temp config file.

## Programmatic build

```go
import (
	"context"
	"github.com/FabioSol/fuego/engine"
)

func main() {
	eng := engine.New()
	eng.Use(mypack.Pack())

	err := eng.Build(context.Background(), engine.BuildOptions{
		ContentDir: "docs/decisions",
		OutputDir:  "build",
		SiteName:   "Engineering Decisions",
		BaseURL:    "/decisions",
	})
	if err != nil {
		// handle
	}
}
```

`engine.Serve(ctx, opts)` runs the dev server (initial build, watch, live
reload, HTTP); `engine.Validate(ctx, opts)` runs the pipeline through INDEX
without rendering and returns the page count — a CI gate.

### BuildOptions

| Field | Purpose |
|---|---|
| `ConfigPath` | Optional base `config.yaml` to start from |
| `ContentDir` / `ThemeDir` / `OutputDir` / `StaticDir` | Directory overrides (packs may supply the theme) |
| `SiteName` / `BaseURL` | Site metadata |
| `DevPort` / `DevCommand` / `ProxyPort` | Dev-server settings (`Serve`) |
| `Incremental` / `CacheDir` | Reuse cached parses for unchanged content |

### Configuration resolution

The effective config is layered, lowest precedence first:

1. **Registered packs' config defaults** (`Pack.ConfigDefaults`)
2. **`ConfigPath`** file, if given
3. **`BuildOptions`** fields (the non-zero overrides above)

So a pack contributes routes and taxonomies, an optional file adds project
config, and your tool sets the dirs and site metadata on top — all deep-merged
([Config Merging](/docs/config-merging/)).

## Building a tool

The recommended shape, as used by fuego-adr:

1. Put all domain logic in a **pack** — parser, theme (with `static/` assets),
   `ConfigDefaults`, and hooks. Export `Pack() core.Pack`.
2. Write a small CLI that builds `BuildOptions` from its own flags and calls
   `eng.Use(yourpack.Pack())` + `eng.Build/Serve/Validate`.

Users then get both: your branded CLI, and the ability to drop your pack into
any Fuego project with one line. See the
[Build a Format Pack](/docs/tutorials/build-a-format-pack/) tutorial.
