---
title: Getting Started
layout: doc
---

## Install

Fuego requires Go 1.23 or later.

```bash
go run github.com/FabioSol/fuego/cmd/fuego@latest init mysite
```

This scaffolds a working project with a `.card` flashcard DSL defined in `config.yaml`:

```
mysite/
  config.yaml        # site config, parsers, routes
  main.go            # engine entry point
  content/
    hello.card       # sample content
  theme/
    base.html        # HTML shell
  public/            # static assets
```

## Build

```bash
cd mysite
go run . build
```

Output is written to `build/` by default. Open `build/index.html` in a browser.

## Dev Server

```bash
go run . serve
```

Watches content and theme files for changes. Rebuilds automatically on save.

## Project Structure

Every Fuego site has the same layout:

- **config.yaml** — site metadata, parser definitions, routes, taxonomies, collections
- **main.go** — Go entry point. Register compiled parsers and hooks here
- **content/** — your content files (any extension)
- **theme/** — HTML templates (base, layouts, renderers)
- **public/** — static assets copied to the output root

## Content Files

Every content file uses YAML frontmatter:

```
---
title: My Page
tags:
  - example
---
Your content here, in whatever format the parser expects.
```

The frontmatter becomes the page envelope. Everything below `---` is the raw payload passed to the parser.
