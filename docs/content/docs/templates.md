---
title: Templates
layout: doc
nav_section: "Reference"
nav_weight: 3
tags:
  - reference
  - templates
---

Templates use Go's `html/template`. The theme directory structure:

```
theme/
  base.html              # HTML shell (required)
  layouts/
    post.html            # named layouts
    tag.html
  renderers/
    question.html        # per-node-type renderers
  partials/
    nav.html             # reusable snippets, called via {{partial "nav" .}}
```

## Base Template

The base template is required. It defines the HTML shell:

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

## Layout Templates

Layouts override the `"content"` block defined in the base template:

```html
{{define "content"}}
<article class="post">{{.Page.Content}}</article>
{{end}}
```

Set a page's layout via frontmatter (`layout: post`) or config (taxonomies, collections).

## Renderer Templates

Per-node-type renderer templates override the default `<div data-type="...">` rendering. Place them in `theme/renderers/{type}.html`.

For example, `theme/renderers/question.html` controls how `question` nodes render.

## Partials

Partials are reusable template snippets in `theme/partials/`. They are available in the base template, layouts, and renderers via the `partial` function:

```html
{{partial "nav" .}}
{{partial "footer" (dict "year" "2026" "owner" .Site.Name)}}
```

The second argument becomes the partial's data (`.`); omit it to pass nothing. Partials can call other partials. Calling a partial that does not exist fails the page render with an error listing the available partials.

## Template Functions

| Function | Example | Description |
|---|---|---|
| `render` | `{{render .Page.Nodes}}` | Recursively render nodes through renderer templates |
| `safeHTML` | `{{safeHTML .Page.Envelope.snippet}}` | Mark a string as raw HTML (skips escaping) |
| `partial` | `{{partial "nav" .}}` | Execute a template from `theme/partials/` |
| `dict` | `{{dict "k1" "v1" "k2" "v2"}}` | Build a map from key/value pairs, for partial arguments |
| `where` | `{{where .Site.Pages "type" "post"}}` | Filter a slice by key = value |
| `sortBy` | `{{sortBy .Site.Pages "weight" "desc"}}` | Copy-and-sort a slice by key; optional `"asc"`/`"desc"` |
| `limit` | `{{where .Site.Pages "type" "post" \| limit 5}}` | At most n leading elements (collection last, pipeline-friendly) |
| `first` | `{{first .Site.Pages}}` | First element, or nil if empty |
| `dateFormat` | `{{dateFormat "Jan 2, 2006" .Page.Envelope.date}}` | Format a `time.Time` or date string with a Go layout |

Key resolution for `where` and `sortBy`: map keys for maps, exported struct fields (case-insensitive), then the element's `Envelope` map — so `"type"` matches a page ref's `Type` field and `"author"` matches `Envelope["author"]`. `where` compares loosely (values match if their string forms are equal), so YAML numbers and template string literals compare predictably. `sortBy` sorts numbers numerically and everything else by string form; elements missing the key sort first.

`dateFormat` accepts `time.Time` or strings in RFC3339, `2006-01-02T15:04:05`, `2006-01-02 15:04:05`, or `2006-01-02`.

## Template Data

| Field | Description |
|---|---|
| `.Page.Envelope` | Frontmatter map |
| `.Page.Content` | Pre-rendered HTML |
| `.Page.URL` | Resolved URL |
| `.Page.Layout` | Layout name |
| `.Page.Type` | Parser type |
| `.Site.Name` | From config |
| `.Site.BaseURL` | From config |
| `.Site.Pages` | All pages as slim refs (`.URL`, `.Type`, `.Layout`, `.Envelope`), URL-sorted — see [Build a Navigation Menu](/docs/how-to/build-a-nav-menu/) |
| `.Paginator` | On paginated listing pages: `.CurrentPage`, `.TotalPages`, `.PrevURL`, `.NextURL` (nil otherwise) — see [Paginate a Collection](/docs/how-to/paginate-a-collection/) |
| `.JSON` | Full page data as JSON (computed only if the template references it) |

## JSON Embed

Embedding `{{.JSON}}` in your base template or a layout gives the page its full data (envelope, parsed nodes, URL) as JSON, typically inside `<script type="application/json" id="fuego-data">`. This enables client-side interactivity without a separate API.

The payload is computed only for pages whose template actually references `.JSON` — Fuego inspects templates at load time, so sites that never hydrate pay no serialization cost. There is no configuration for this; using `.JSON` anywhere in a template (including inside `if`/`range` blocks or via `$.JSON`) enables it for pages rendered with that template.
