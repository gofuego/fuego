# Fuego

A meta-engine for static site generation in Go. Define custom DSLs (`.trivia`, `.card`, `.pitch`, anything), map them to HTML through a configurable parsing and rendering pipeline, and build static sites from arbitrary content formats.

**[Documentation](https://fabiosol.github.io/fuego/index/)**

## Why Fuego?

Most SSGs are Markdown-first. Fuego is format-agnostic. You define the content format, the parsing rules, and the templates. Fuego handles discovery, routing, taxonomy indexing, collections, and the build pipeline. Markdown works out of the box with GFM support, but the real power is custom formats.

## Install

```bash
go install github.com/FabioSol/fuego/cmd/fuego@latest
```

Requires Go 1.23+. Adds `fuego` to `$GOPATH/bin` (usually `~/go/bin`).

## Quick Start

```bash
fuego init mysite
cd mysite
fuego build
```

This scaffolds a working project with a `.card` flashcard DSL, a Markdown homepage, styled templates, and a dev server:

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

```bash
fuego serve          # dev server at http://localhost:8080
```

## Key Features

- **Format-agnostic** — define content formats via config (declarative regex parsers) or Go code (compiled parsers)
- **Built-in Markdown** — GFM support via goldmark (tables, strikethrough, autolinks, task lists)
- **Three-tier routing** — frontmatter slug > config pattern > filesystem mirror
- **Taxonomies & collections** — automatic term pages, index pages, and sorted listing pages
- **Hooks** — `AfterParse` and `BeforeRender` Go functions to enrich, filter, or transform pages
- **Dev server** — file watching, live rebuild, optional Vite proxy
- **Site manifest** — `site-manifest.json` with page index, taxonomy terms, and collection membership
- **Deterministic output** — sorted keys, reproducible builds

## Example: Custom Parser in Config

```yaml
parsers:
  trivia:
    rules:
      - match: '^\?\s*(.+)$'
        emit:
          type: question
          content: "$1"
      - match: '^\[([A-Z])\]\s+(.+)$'
        emit:
          type: answer
          content: "$2"
          attributes:
            letter: "$1"
```

This lets you write `.trivia` files and have them parsed into a universal AST that your templates render however you want.

## CLI

```
fuego build              Build the static site
fuego serve              Dev server with live rebuild
fuego validate           Check for errors (no output)
fuego list               Print all pages as TYPE | SOURCE | URL
fuego init <dir>         Scaffold a new project
```

## Documentation

Full documentation is available at **[fabiosol.github.io/fuego](https://fabiosol.github.io/fuego/index/)** — built with Fuego itself.

- [Getting Started](https://fabiosol.github.io/fuego/docs/getting-started/)
- [Configuration](https://fabiosol.github.io/fuego/docs/configuration/)
- [Custom Parsers](https://fabiosol.github.io/fuego/docs/custom-parsers/)
- [Templates](https://fabiosol.github.io/fuego/docs/templates/)
- [Hooks](https://fabiosol.github.io/fuego/docs/hooks/)
- [CLI Reference](https://fabiosol.github.io/fuego/docs/cli/)

## License

MIT
