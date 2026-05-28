# CLAUDE.md — Fuego Site

This is a static site built with [Fuego](https://github.com/FabioSol/fuego), a format-agnostic meta-engine for static site generation in Go.

## Commands

```bash
go run . build          # Build the site to build/
go run . serve          # Dev server with live rebuild (http://localhost:8080)
go run . validate       # Check for errors without building
go run . list           # Print all pages: TYPE | SOURCE | URL
```

## Project Structure

```
config.yaml        Site config: parsers, routes, taxonomies, collections
main.go            Entry point — register compiled parsers and hooks here
content/           Content files (any extension matched by a parser)
theme/
  base.html        HTML shell (required) — defines the page wrapper
  layouts/         Named layouts — override the "content" block from base
  renderers/       Per-node-type renderers — override default <div> rendering
public/            Static assets — copied to output root as-is
build/             Generated output (gitignored)
```

## Adding Content

Create a file in `content/` with YAML frontmatter:

```
---
title: My Page
layout: card
---
front: The question
back: The answer
```

- **title** — used in templates via `{{.Page.Envelope.title}}`
- **layout** — selects `theme/layouts/{name}.html` (falls back to base.html)
- **slug** — overrides the URL slug segment (default: filename without extension)
- Everything below `---` is parsed by the parser matching the file extension

## Adding a New Content Format

### Declarative (config-only)

Add regex rules to `config.yaml`:

```yaml
parsers:
  trivia:
    rules:
      - match: '^\?\s*(.+)$'
        emit:
          type: question
          content: "$1"
```

This parses `.trivia` files line-by-line. First matching rule wins. `$1`, `$2` etc. are capture group substitutions.

### Compiled (Go code)

Implement `core.Parser` and register it in `main.go`:

```go
eng.Register(&MyParser{})
```

Compiled parsers override declarative ones with the same name.

## Templates

Templates use Go's `html/template`. Available data:

| Field | Description |
|-------|-------------|
| `.Page.Envelope` | Frontmatter map (access fields like `.Page.Envelope.title`) |
| `.Page.Content` | Pre-rendered HTML from the parser |
| `.Page.URL` | Resolved page URL |
| `.Page.Layout` | Layout name |
| `.Page.Type` | Parser/file type |
| `.Site.Name` | From `config.yaml` |
| `.Site.BaseURL` | From `config.yaml` — use as path prefix for links and assets |
| `.JSON` | Full page data as JSON (for client-side use) |

### Layouts

Create `theme/layouts/{name}.html` to override the `"content"` block:

```html
{{define "content"}}
<article>{{.Page.Content}}</article>
{{end}}
```

## Routing

URL resolution priority: frontmatter `slug` > config `routes` pattern > filesystem mirror.

```yaml
routes:
  card: "/cards/{slug}"
```

Placeholders: `{dir}`, `{slug}`, `{filename}`.

## Hooks

Register Go functions in `main.go` to transform pages between pipeline phases:

```go
eng.AfterParse(func(pages []*core.Page) ([]*core.Page, error) {
    // runs after parsing, before routing — enrich or filter pages
    return pages, nil
})

eng.BeforeRender(func(pages []*core.Page) ([]*core.Page, error) {
    // runs after indexing, before rendering — final transforms
    return pages, nil
})
```

## Deployment

Set `base_url` in `config.yaml` to your deploy path (e.g., `/my-repo` for GitHub Pages subpath, or empty for root). All template links should use `{{.Site.BaseURL}}` as a prefix.
