---
title: Why Format-Agnostic?
layout: doc
nav_section: "Concepts"
nav_weight: 1
tags:
  - concepts
  - architecture
---

Most static site generators treat Markdown as the default content format. Hugo, Jekyll, Eleventy — they all parse Markdown first and offer limited escape hatches for other formats. Fuego inverts this: **no format is privileged**.

## The Problem with Markdown-First

Markdown is great for prose. But not all content is prose:

- **Flashcards** have a front and a back. Markdown can't express that naturally.
- **Quiz questions** have a prompt, choices, and a correct answer. Markdown requires awkward conventions.
- **Product cards** have structured fields (price, rating, image). Markdown frontmatter can carry metadata, but the body is still unstructured.
- **Slide decks** need slide boundaries and speaker notes. Markdown uses `---` separators as a hack.

When your SSG is Markdown-first, you end up encoding structure in conventions: specific heading levels mean specific things, custom HTML blocks break the abstraction, and frontmatter becomes a dumping ground.

## Fuego's Approach

In Fuego, the content format is a first-class concept:

1. **You define the syntax** — via regex rules in config or a Go parser
2. **You define the semantics** — node types like `question`, `front`, `ingredient` carry meaning
3. **You define the rendering** — template per node type, or CSS on `data-type` attributes

The engine never interprets what a `question` node means. It just knows how to discover files, dispatch them to parsers, route the results, and render templates. Your DSL, your rules.

## The Universal AST

All parsers — Markdown, declarative regex, compiled Go — produce the same output:

```go
type Node struct {
    Type       string            // "question", "front", "html", anything
    Attributes map[string]any    // structured metadata
    Content    string            // text content
    Children   []Node            // nested nodes
    Raw        bool              // emit Content as raw HTML, unwrapped
}
```

This uniformity means the entire pipeline after PARSE is format-agnostic. Routing, taxonomy indexing, collections, rendering, manifest generation — all work the same regardless of whether the content started as `.md`, `.trivia`, or `.recipe`.

## Markdown is Still First-Class

Format-agnostic doesn't mean anti-Markdown. Fuego ships a first-party Markdown parser with full GFM support (tables, strikethrough, autolinks, task lists) — you opt in by registering it (`eng.Register(markdown.Parser())`), the same way you'd register any parser. It emits a single `Raw: true` node that the renderer passes through as HTML.

You can even mix formats in the same site: Markdown for blog posts, `.trivia` for quiz pages, `.card` for flashcards. Each file extension dispatches to its own parser, and the pipeline handles the rest.

## When This Matters

Format-agnostic design shines when:

- **Your content has inherent structure** that Markdown can't express
- **Non-developers author content** using a simple, domain-specific syntax
- **Client-side interactivity** needs typed data (the JSON embed carries the full AST)
- **Multiple content types coexist** in one site with different rendering needs

If all your content is prose, register the first-party Markdown parser and you have a normal Markdown site. The meta-engine architecture is there for when you need more.
