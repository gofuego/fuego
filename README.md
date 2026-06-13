# Fuego

A meta-engine for static site generation in Go. Define custom DSLs (`.trivia`, `.card`, `.pitch`, anything), map them to HTML through a configurable parsing and rendering pipeline, and build static sites from arbitrary content formats.

**[Documentation](https://fabiosol.github.io/fuego/index/)**

## Why Fuego?

Most SSGs are Markdown-first. Fuego is format-agnostic. You define the content format, the parsing rules, and the templates. Fuego handles discovery, routing, taxonomy indexing, collections, and the build pipeline. No format is privileged — Markdown is a first-party parser you opt into (`eng.Register(markdown.Parser())`), the same as any other. The real power is custom formats, packaged and shared as **format packs**.

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

This scaffolds a working project with a `.card` flashcard DSL, a paginated collection, a Markdown homepage, a partial-driven nav, RSS/sitemap outputs, and a dev server:

```
mysite/
  CLAUDE.md          # agent-friendly project guide
  config.yaml        # site config, parsers, collections
  main.go            # engine entry point — registers the Markdown parser
  content/
    index.md         # Markdown homepage
    cards/           # sample .card DSL collection (paginated)
  theme/
    base.html        # HTML shell
    layouts/         # named layout overrides (home, card, listing)
    partials/        # nav.html, driven by .Site.Pages
    renderers/       # per-node-type rendering (front, back, page-ref)
    outputs/         # sitemap.xml + rss.xml (non-HTML outputs)
  public/
    style.css        # starter stylesheet
    index.html       # root redirect
```

```bash
fuego serve          # dev server at http://localhost:8080
```

## Key Features

- **Format-agnostic** — define content formats via config (declarative regex parsers) or Go code (compiled parsers)
- **Markdown** — opt-in GFM support via goldmark (tables, strikethrough, autolinks, task lists)
- **Format packs** — bundle parsers, hooks, and themes into installable modules via `eng.Use()`, with namespaced config and deep-merged defaults
- **Three-tier routing** — frontmatter slug > config pattern > filesystem mirror
- **Taxonomies & collections** — automatic term pages, index pages, and sorted listing pages, with optional pagination
- **Cross-page templates** — `.Site.Pages`, partials, and query funcs (`where`, `sortBy`, …) for navs and listings
- **Non-HTML outputs** — RSS, sitemaps, and search indexes from `theme/outputs/`
- **Hooks** — `AfterParse`, `Index`, and `BeforeRender` Go functions to enrich, filter, or transform pages
- **Incremental builds** — opt-in parse cache + render narrowing, byte-identical to a clean build
- **Embeddable** — a programmatic API (`engine.Build/Serve/Validate`) for building domain-specific generators on top of Fuego
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
fuego build [--incremental]   Build the static site
fuego serve                   Dev server with live rebuild
fuego validate                Check for errors (no output)
fuego list                    Print all pages as TYPE | SOURCE | URL
fuego config                  Print the resolved config with per-key provenance
fuego init <dir> [--pack M]   Scaffold a new project (optionally with a format pack)
```

## Documentation

Full documentation is available at **[fabiosol.github.io/fuego](https://fabiosol.github.io/fuego/index/)** — built with Fuego itself.

- [Getting Started](https://fabiosol.github.io/fuego/docs/getting-started/)
- [Configuration](https://fabiosol.github.io/fuego/docs/configuration/)
- [Custom Parsers](https://fabiosol.github.io/fuego/docs/custom-parsers/)
- [Templates](https://fabiosol.github.io/fuego/docs/templates/)
- [Hooks](https://fabiosol.github.io/fuego/docs/hooks/)
- [Format Packs](https://fabiosol.github.io/fuego/docs/concepts/format-packs/)
- [Embedding Fuego](https://fabiosol.github.io/fuego/docs/concepts/embedding/)
- [CLI Reference](https://fabiosol.github.io/fuego/docs/cli/)

## License

MIT
