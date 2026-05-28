---
title: "Tutorial: Your First Compiled Parser"
layout: doc
order: 2
tags:
  - tutorial
  - parsers
  - go
---

Declarative parsers handle most use cases, but sometimes you need the full power of Go. In this tutorial you'll write a compiled parser for a `.recipe` format.

## The Format

Recipe files look like this:

```
---
title: Pancakes
tags:
  - breakfast
---
@ 2 cups flour
@ 2 eggs
@ 1 cup milk

> Mix dry ingredients
> Add wet ingredients and stir
> Cook on griddle until golden
```

Lines starting with `@` are ingredients. Lines starting with `>` are steps.

## 1. Define the Parser

In your project, create `parser_recipe.go`:

```go
package main

import (
    "strings"
    "github.com/FabioSol/fuego/core"
)

type RecipeParser struct{}

func (p *RecipeParser) Type() string { return "recipe" }

func (p *RecipeParser) Parse(raw []byte, meta core.Envelope) ([]core.Node, error) {
    var nodes []core.Node
    for _, line := range strings.Split(string(raw), "\n") {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }
        switch {
        case strings.HasPrefix(line, "@ "):
            nodes = append(nodes, core.Node{
                Type:    "ingredient",
                Content: strings.TrimPrefix(line, "@ "),
            })
        case strings.HasPrefix(line, "> "):
            nodes = append(nodes, core.Node{
                Type:    "step",
                Content: strings.TrimPrefix(line, "> "),
            })
        }
    }
    return nodes, nil
}
```

## 2. Register It

In `main.go`, add one line:

```go
eng := engine.New()
eng.Register(&RecipeParser{})
eng.Run(os.Args)
```

That's it. Fuego now knows how to parse `.recipe` files.

## 3. When to Use Compiled vs Declarative

Use **declarative** (config-only) when:

- Your format is line-oriented
- Simple regex rules can capture the structure
- You want non-developers to be able to define formats

Use **compiled** (Go code) when:

- You need multi-line parsing (blocks, nesting)
- You need to call external libraries
- You need complex validation or transformation logic
- Performance matters (compiled parsers avoid regex overhead)

Both produce the same `[]Node` output. The rest of the pipeline (routing, rendering, taxonomy indexing) works identically.
