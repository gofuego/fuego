---
title: Hooks
layout: doc
nav_section: "Reference"
nav_weight: 5
tags:
  - reference
  - go
---

Hooks let you transform pages between pipeline phases using Go functions.

## AfterParse

Runs after PARSE, before ROUTE. Use it to enrich or filter pages:

```go
eng := engine.New()

eng.AfterParse(func(pages []*core.Page) ([]*core.Page, error) {
    for _, p := range pages {
        words := len(strings.Fields(p.Nodes[0].Content))
        p.Envelope["reading_time"] = words / 200
    }
    return pages, nil
})
```

## Index

Runs during INDEX, after taxonomy and collection virtual pages are generated but **before the collision re-check**. This is the supported way to add virtual pages — set their URL and they are collision-checked exactly like engine-generated ones:

```go
eng.Index(func(pages []*core.Page) ([]*core.Page, error) {
    overview := &core.Page{
        RelPath:  "virtual:overview",
        URL:      "/overview",
        Type:     "diagram",
        Envelope: core.Envelope{"title": "Overview"},
        Nodes: []core.Node{{
            Type:       "graph-data",
            Attributes: map[string]any{"nodes": buildGraph(pages)},
        }},
    }
    return append(pages, overview), nil
})
```

Do not add pages in `BeforeRender` — pages added there bypass collision detection.

### Skipping pages

Set `page.Skip = true` in any hook to exclude a page from rendering and the manifest while keeping it visible to later hooks (drafts, internal pages):

```go
eng.Index(func(pages []*core.Page) ([]*core.Page, error) {
    for _, p := range pages {
        if p.Envelope["draft"] == true {
            p.Skip = true
        }
    }
    return pages, nil
})
```

## BeforeRender

Runs after INDEX, before RENDER. Pages have their final URLs and taxonomy assignments:

```go
eng.BeforeRender(func(pages []*core.Page) ([]*core.Page, error) {
    if os.Getenv("FUEGO_ENV") == "production" {
        var published []*core.Page
        for _, p := range pages {
            if p.Envelope["draft"] != true {
                published = append(published, p)
            }
        }
        return published, nil
    }
    return pages, nil
})
```

## Behavior

- Multiple hooks at the same point run in **FIFO** registration order
- Each hook receives the previous hook's output
- Hooks can **mutate** pages (add envelope fields), **filter** them (return a subset), or **skip** them (`page.Skip = true`)
- Only `Index` hooks should add pages — additions there go through collision detection
- Hooks run in all commands: `build`, `serve`, `validate`, `list`

## Why Go-Only?

Hooks transform typed Go structs. Shell-based hooks would require JSON serialization round-trips, lose type safety, and add latency. The `prebuild` config field handles the shell-command use case (npm, tailwind, etc.) since it runs before any pipeline data exists.
