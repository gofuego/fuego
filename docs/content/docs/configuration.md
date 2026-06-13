---
title: Configuration
layout: doc
tags:
  - reference
  - config
---

All configuration lives in `config.yaml` at the project root.

## Site

```yaml
site:
  name: "My Site"
  base_url: "https://example.com"
```

## Routes

Three-tier URL resolution: frontmatter `slug` > config route pattern > filesystem mirror.

```yaml
routes:
  trivia: "/quiz/{dir}/{slug}"
  card: "/cards/{slug}"
```

**Placeholders:**

| Placeholder | Expands to |
|---|---|
| `{dir}` | Directory path relative to content root |
| `{slug}` | Filename without extension |
| `{filename}` | Full filename with extension |

## Ignore Patterns

Doublestar glob patterns to skip files during discovery:

```yaml
ignore:
  - "**/.DS_Store"
  - "**/drafts/*"
```

## Taxonomies

Two-tier taxonomy pages — a page per term, plus an index listing all terms:

```yaml
taxonomies:
  tags:
    path: "/tags/{term}"
    layout: "tag"
    index_path: "/tags"
    index_layout: "tag-index"
    page_size: 10          # optional: paginate term pages
```

Pages with a `tags` field in frontmatter are automatically indexed. Virtual pages for each term and the index are generated during the INDEX phase. With `page_size` set, term pages over that size split into `/tags/{term}/page/{n}/` — see [Paginate a Collection](/docs/how-to/paginate-a-collection/).

## Collections

Glob-matched, sorted listing pages:

```yaml
collections:
  history-quiz:
    match: "trivia/history/**"
    sort_by: "points"
    layout: "listing"
    path: "/history-quiz"
    page_size: 10          # optional: split into /history-quiz/page/{n}/
```

## Packs

Each registered [format pack](/docs/concepts/format-packs/) reads its own `packs.{name}:` subtree. The engine routes the subtree to the pack; the pack validates and interprets it in Go.

```yaml
packs:
  adr:
    status_workflow: [proposed, accepted, superseded]
    diagram: true
```

A subtree whose name matches no registered pack logs a warning.

## Static Files

- `public/` directory contents are copied to the output root
- Binary files colocated with content (images, etc.) are mirrored to their routed paths

## Prebuild Hook

Run a shell command before each build:

```yaml
prebuild: "npm run build:css"
```

## Dev Server

```yaml
dev:
  port: 8080
  command: "npx vite --port 5173"
  proxy_port: 5173
```

The dev server proxies asset requests to the specified port when `command` and `proxy_port` are set.
