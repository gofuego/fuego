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
