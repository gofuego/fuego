---
title: Format Packs
layout: doc
tags:
  - concepts
  - packs
---

A format pack bundles everything a content format needs — parsers, hooks, and theme templates — into one registerable unit. Packs are plain Go modules: installing one is a `go get` plus one line of code.

```go
import "github.com/example/fuego-pack-adr"

eng := engine.New()
eng.Use(adr.Pack())
```

## What a pack contains

```go
core.Pack{
    Name:    "adr",
    Parsers: []core.Parser{adrParser},
    Hooks:   core.Hooks{Index: []core.IndexHook{buildGraph}},
    Theme:   themeFS, // embed.FS with base.html, layouts/, renderers/, partials/
}
```

The `Theme` FS mirrors the user theme directory layout: an optional `base.html` at the root, plus `layouts/`, `renderers/`, and `partials/` subdirectories. Packs typically embed it:

```go
//go:embed theme
var themeRoot embed.FS

func Pack() core.Pack {
    theme, _ := fs.Sub(themeRoot, "theme")
    return core.Pack{Name: "adr", Theme: theme /* ... */}
}
```

## Precedence

Conflicts resolve in one direction — toward whoever is closest to the site:

| Conflict | Winner | Noise |
|---|---|---|
| User `theme/` file vs. pack template | User file | silent — that's the override gesture |
| Later pack vs. earlier pack template | Later pack | warning logged |
| User `Register()` parser vs. pack parser | User parser, regardless of call order | silent |
| Later pack vs. earlier pack parser | Later pack | warning logged |
| Pack parser vs. declarative config parser | Pack parser | — |

A site can run entirely on a pack's theme — `base.html` is required overall, but it may come from a pack instead of the user's theme directory.

## Hooks

Pack hooks append to the engine's hook lists in registration order and run FIFO alongside user hooks. A pack that generates virtual pages (diagrams, indexes) should do so in an `Index` hook so its pages flow through collision detection.

## Config and the Init lifecycle

A pack reads its settings from a namespaced `packs.{name}:` subtree of `config.yaml`:

```yaml
packs:
  adr:
    status_workflow: [proposed, accepted, superseded]
    diagram: true
```

Give the pack an `Init` function to receive that subtree and act on it. `Init` runs once during the INIT phase — before content discovery — so it can register parsers and hooks conditionally:

```go
core.Pack{
    Name: "adr",
    Init: func(ctx context.Context, pc *core.PackContext) error {
        cfg := pc.Config() // map[string]any, the packs.adr subtree (nil if absent)

        // Validate in Go — there is no schema language.
        if _, ok := cfg["status_workflow"]; !ok {
            return fmt.Errorf("adr pack requires status_workflow")
        }

        // Register conditionally based on config.
        if enabled, _ := cfg["diagram"].(bool); enabled {
            pc.Index(buildDiagramHook)
        }
        return nil
    },
}
```

Rules:

- `Init` is optional; packs without one are pure declarative bundles.
- An `Init` error halts the build as `pack "{name}": <your error>`.
- A `packs.{name}:` subtree with no matching registered pack logs a warning naming the known packs — typos don't pass silently.
- Parsers and hooks registered in `Init` follow the same precedence as those declared on the `Pack` struct.
