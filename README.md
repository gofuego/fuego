# Fuego

A meta-engine for static site generation in Go. Define custom DSLs (`.trivia`, `.card`, `.pitch`, anything), map them to HTML through a configurable parsing and rendering pipeline, and build static sites from arbitrary content formats.

## Why Fuego?

Most SSGs are Markdown-first. Fuego is format-agnostic. You define the content format, the parsing rules, and the templates. Fuego handles discovery, routing, taxonomy indexing, collections, and the build pipeline. Markdown works out of the box with GFM support, but the real power is custom formats.

## Quick Start

```bash
# Scaffold a new project
go run github.com/FabioSol/fuego/cmd/fuego@latest init mysite

cd mysite
go run . build
```

This generates a project with a `.card` flashcard DSL defined entirely in `config.yaml`:

```
mysite/
  config.yaml        # site config, parsers, routes
  main.go            # engine entry point
  content/
    hello.card       # your content
  theme/
    base.html        # HTML shell
  public/            # static assets (copied to output root)
```

## Parsing

### Built-in Markdown

`.md` files are parsed automatically using [goldmark](https://github.com/yuin/goldmark) with GitHub-Flavored Markdown (tables, strikethrough, autolinks, task lists). The output is a single `Node{Type: "html"}` containing rendered HTML. No config needed.

To override the built-in Markdown parser, register your own compiled or declarative parser for the `md` type. Parser priority: compiled > declarative > built-in.

## Custom Parsers

### Declarative (config-only)

Define parsers with ordered regex rules in `config.yaml`. No Go code needed:

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

This parses `.trivia` files line-by-line. First matching rule wins. Capture groups (`$1`, `$2`) are substituted into content and attributes.

### Compiled (Go code)

Implement the `core.Parser` interface for full control:

```go
type Parser interface {
    Type() string
    Parse(rawPayload []byte, meta Envelope) ([]Node, error)
}
```

Register it in `main.go`:

```go
eng := engine.New()
eng.Register(&MyCustomParser{})
eng.Run(os.Args)
```

Compiled parsers take priority over declarative ones with the same name. Both override the built-in Markdown parser.

## Content Files

Every content file uses a YAML frontmatter envelope:

```
---
title: Napoleon
tags:
  - history
  - europe
points: 20
---
? When was Napoleon born?
[A] 1769
[B] 1789
```

The frontmatter becomes the envelope (accessible in templates as `.Page.Envelope`). Everything below `---` is the raw payload passed to the parser.

## Universal AST

All parsers produce `[]Node`:

```go
type Node struct {
    Type       string
    Attributes map[string]any
    Content    string
    Children   []Node
}
```

The engine never interprets node types. Your templates decide how `question`, `front`, `answer`, or any other type renders.

## Configuration

### Routes

Three-tier URL resolution: frontmatter `slug` > config route pattern > filesystem mirror.

```yaml
routes:
  trivia: "/quiz/{dir}/{slug}"
  card: "/cards/{slug}"
```

Placeholders: `{dir}` (directory path), `{slug}` (filename without extension), `{filename}` (full filename).

### Ignore Patterns

Doublestar glob patterns to skip files during discovery:

```yaml
ignore:
  - "**/.DS_Store"
  - "**/drafts/*"
```

### Taxonomies

Two-tier taxonomy pages (term pages + index page):

```yaml
taxonomies:
  tags:
    path: "/tags/{term}"
    layout: "tag"
    index_path: "/tags"
    index_layout: "tag-index"
```

Pages with a `tags` field in their frontmatter are automatically indexed. Virtual term pages and an index page are generated.

### Collections

Glob-matched, sorted listing pages:

```yaml
collections:
  history-quiz:
    match: "trivia/history/**"
    sort_by: "points"
    layout: "listing"
    path: "/history-quiz"
```

### Static Files

- `public/` directory contents are copied to the output root
- Binary files colocated with content (images, etc.) are mirrored to their routed paths

### Prebuild Hook

Run a shell command before each build:

```yaml
prebuild: "npm run build:css"
```

### Dev Server

```yaml
dev:
  port: 8080
  command: "npx vite --port 5173"
  proxy_port: 5173
```

## Templates

Templates use Go's `html/template`. The theme directory structure:

```
theme/
  base.html              # HTML shell (required)
  layouts/
    post.html            # named layouts (optional)
    tag.html
  renderers/
    question.html        # per-node-type renderers (optional)
```

### Base Template

```html
<!DOCTYPE html>
<html>
<head><title>{{.Page.Envelope.title}} | {{.Site.Name}}</title></head>
<body>
{{block "content" .}}<div id="root">{{.Page.Content}}</div>{{end}}
<script type="application/json" id="fuego-data">{{.JSON}}</script>
</body>
</html>
```

### Layout Templates

Override the `"content"` block:

```html
{{define "content"}}<article class="post">{{.Page.Content}}</article>{{end}}
```

Set a page's layout via frontmatter (`layout: post`) or config (taxonomies, collections).

### Template Data

| Field | Description |
|-------|-------------|
| `.Page.Envelope` | Frontmatter map |
| `.Page.Content` | Pre-rendered HTML |
| `.Page.URL` | Resolved URL |
| `.Page.Layout` | Layout name |
| `.Page.Type` | Parser type |
| `.Site.Name` | From config |
| `.Site.BaseURL` | From config |
| `.JSON` | Full page data as JSON |

## CLI

```
fuego build       Build the static site
fuego serve       Dev server with file watching and live rebuild
fuego validate    Check config and content for errors (no output)
fuego list        Print all pages as TYPE | SOURCE | URL
fuego init <dir>  Scaffold a new project
```

Global flag: `--config path/to/config.yaml` (default: `config.yaml`)

## Hooks

Register Go functions to transform pages between pipeline phases:

```go
eng := engine.New()

// Runs after PARSE, before ROUTE — enrich or filter pages
eng.AfterParse(func(pages []*core.Page) ([]*core.Page, error) {
    for _, p := range pages {
        // Add reading time estimate
        words := len(strings.Fields(p.Nodes[0].Content))
        p.Envelope["reading_time"] = words / 200
    }
    return pages, nil
})

// Runs after INDEX, before RENDER — see final URLs, taxonomy pages
eng.BeforeRender(func(pages []*core.Page) ([]*core.Page, error) {
    // Filter drafts based on environment
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

Multiple hooks at the same point run in FIFO registration order. Each receives the previous hook's output. Hooks run in all commands (`build`, `serve`, `validate`, `list`).

## Build Pipeline

```
PREBUILD       →  shell hook
INIT           →  merge built-in + declarative + compiled parsers
DISCOVER       →  walk content dir, apply ignore patterns, classify files
PARSE          →  extract frontmatter, run parsers (concurrent)
AFTER-PARSE    →  user hooks (enrich, filter)
ROUTE          →  resolve URLs, detect collisions
INDEX          →  build taxonomies + collections, re-check collisions
BEFORE-RENDER  →  user hooks (final transforms)
RENDER         →  execute templates (concurrent)
MANIFEST       →  write site-manifest.json
STATIC         →  copy public/ and colocated assets
```

## Site Manifest

Every build produces `site-manifest.json` in the output directory:

```json
{
  "pages": [
    {"url": "/blog/welcome/", "type": "md", "title": "Welcome", "envelope": {...}}
  ],
  "taxonomies": {
    "tags": {
      "terms": {
        "history": [0, 2],
        "europe": [2]
      }
    }
  },
  "collections": {
    "history-quiz": {
      "pages": [1, 2]
    }
  }
}
```

Taxonomy and collection entries reference pages by integer index into the sorted `pages` array.

## Error Handling

Three severity levels:

- **Warning** -- logged, build continues
- **LocalFatal** -- page skipped, build continues
- **GlobalFatal** -- build fails immediately

`fuego validate` runs the pipeline through INDEX without rendering, catching config errors, parse failures, and URL collisions before you build.

## License

MIT
