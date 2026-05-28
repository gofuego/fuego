---
title: Hooks
layout: doc
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
- Hooks can **mutate** pages (add envelope fields) or **filter** them (return a subset)
- Hooks run in all commands: `build`, `serve`, `validate`, `list`

## Why Go-Only?

Hooks transform typed Go structs. Shell-based hooks would require JSON serialization round-trips, lose type safety, and add latency. The `prebuild` config field handles the shell-command use case (npm, tailwind, etc.) since it runs before any pipeline data exists.
