---
title: Getting Started
layout: doc
tags:
  - getting-started
  - tutorial
---

## Install

Fuego requires Go 1.23 or later.

```bash
go install github.com/FabioSol/fuego/cmd/fuego@latest
```

This adds the `fuego` binary to your `$GOPATH/bin` (usually `~/go/bin`). Make sure it's in your PATH:

```bash
export PATH="$HOME/go/bin:$PATH"
```

You can also run without installing:

```bash
go run github.com/FabioSol/fuego/cmd/fuego@latest init mysite
```

## Scaffold a Project

```bash
fuego init mysite
```

This scaffolds a working project with a `.card` flashcard DSL, a Markdown homepage, styled templates, and a dev server ready to go:

```
mysite/
  CLAUDE.md          # agent-friendly project guide
  config.yaml        # site config, parsers, routes
  main.go            # engine entry point
  content/
    index.md         # Markdown homepage
    hello.card       # sample custom DSL content
  theme/
    base.html        # HTML shell with nav
    layouts/         # named layout overrides
  public/
    style.css        # starter stylesheet
    index.html       # root redirect
```

## Build

```bash
cd mysite
fuego build
```

Output is written to `build/` by default. If you didn't install the CLI globally, use `go run . build` instead.

## Dev Server

```bash
fuego serve
```

Starts a local server at `http://localhost:8080` with file watching. Edit any content or theme file and the site rebuilds automatically.

## Project Structure

Every Fuego site has the same layout:

- **config.yaml** — site metadata, parser definitions, routes, taxonomies, collections
- **main.go** — Go entry point. Register compiled parsers and hooks here
- **content/** — your content files (any extension matched by a parser)
- **theme/** — HTML templates (base, layouts, renderers)
- **public/** — static assets copied to the output root
- **build/** — generated output (gitignored)

## Content Files

Every content file uses YAML frontmatter:

```
---
title: My Page
layout: card
tags:
  - example
---
Your content here, in whatever format the parser expects.
```

The frontmatter becomes the page envelope (accessible in templates as `.Page.Envelope`). Everything below `---` is the raw payload passed to the parser matching the file extension.

## Deployment

Set `base_url` in `config.yaml` to your deploy path:

- **Root domain** — `base_url: ""`
- **GitHub Pages subpath** — `base_url: "/my-repo"`

Use `{{.Site.BaseURL}}` in your templates as a prefix for all links and asset references.
