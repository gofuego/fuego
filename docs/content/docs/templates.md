---
title: Templates
layout: doc
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
| `.JSON` | Full page data as JSON |

## JSON Embed

Every page includes a `<script type="application/json" id="fuego-data">` element containing the full page data (envelope, parsed nodes, URL). This enables client-side interactivity without a separate API.
