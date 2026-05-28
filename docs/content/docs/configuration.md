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
```

Pages with a `tags` field in frontmatter are automatically indexed. Virtual pages for each term and the index are generated during the INDEX phase.

## Collections

Glob-matched, sorted listing pages:

```yaml
collections:
  history-quiz:
    match: "trivia/history/**"
    sort_by: "points"
    layout: "listing"
    path: "/history-quiz"
```

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
