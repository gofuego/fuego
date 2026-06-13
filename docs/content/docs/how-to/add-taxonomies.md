---
title: Add Taxonomies
layout: doc
nav_section: "Guides"
nav_weight: 1
tags:
  - how-to
  - config
  - taxonomies
---

Taxonomies create automatic index pages from frontmatter fields. This guide shows how to add tag pages to your site.

## 1. Declare the Taxonomy

In `config.yaml`:

```yaml
taxonomies:
  tags:
    path: "/tags/{term}"
    layout: "tag"
    index_path: "/tags"
    index_layout: "tag-index"
```

## 2. Tag Your Content

Add a `tags` field to any content file's frontmatter:

```yaml
---
title: My Post
tags:
  - go
  - web
---
```

## 3. Create the Tag Layout

Create `theme/layouts/tag.html` to display pages tagged with a specific term:

```html
{{define "content"}}
<h1>{{.Page.Envelope.title}}</h1>
<ul>
{{.Page.Content}}
</ul>
{{end}}
```

The content contains `page-ref` nodes — each rendered as a `<div>` with `data-type="page-ref"` and `data-attrs` containing `url`, `title`, and `type`. You can style these with CSS:

```css
[data-type="page-ref"] { margin: .5rem 0; }
```

## 4. Create the Tag Index Layout

Create `theme/layouts/tag-index.html` to list all tags:

```html
{{define "content"}}
<h1>{{.Page.Envelope.title}}</h1>
<ul>
{{.Page.Content}}
</ul>
{{end}}
```

The content contains `term-ref` nodes with attributes `term`, `count`, and `url`.

## How It Works

During the INDEX phase, Fuego scans every page's frontmatter for the declared taxonomy fields. It builds an inverted index (term to list of pages) and generates two kinds of virtual pages:

- **Term pages** — one per unique term (e.g., `/tags/go/`), containing `page-ref` nodes for each tagged page
- **Index page** — lists all terms (e.g., `/tags/`), containing `term-ref` nodes with page counts

These virtual pages flow through RENDER like any other page, using the layouts you specify.
