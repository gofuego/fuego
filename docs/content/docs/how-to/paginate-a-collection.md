---
title: Paginate a Collection
layout: doc
nav_section: "Guides"
nav_weight: 3
tags:
  - how-to
  - collections
---

Add `page_size` to any collection or taxonomy to split its listing into multiple pages.

## Configuration

```yaml
collections:
  posts:
    match: "posts/**"
    sort_by: "date"
    layout: "listing"
    path: "/blog"
    page_size: 10
```

With more entries than `page_size`, the listing splits into numbered pages:

- Page 1 stays at the configured path: `/blog/`
- Subsequent pages live under `page/`: `/blog/page/2/`, `/blog/page/3/`, …

The same applies to taxonomy **term** pages — set `page_size` under a taxonomy to paginate `/tags/{term}/`.

## The paginator in templates

Each paginated page exposes `.Paginator`:

| Field | Description |
|---|---|
| `.Paginator.CurrentPage` | 1-based page number |
| `.Paginator.TotalPages` | Total number of pages |
| `.Paginator.PrevURL` | Previous page URL, empty on page 1 |
| `.Paginator.NextURL` | Next page URL, empty on the last page |

`.Paginator` is `nil` on unpaginated pages, so guard with `{{if .Paginator}}`:

```html
{{define "content"}}
<ol>{{.Page.Content}}</ol>

{{- if .Paginator}}
<nav class="pagination">
    {{- if .Paginator.PrevURL}}<a href="{{.Paginator.PrevURL}}">← prev</a>{{end}}
    <span>page {{.Paginator.CurrentPage}} of {{.Paginator.TotalPages}}</span>
    {{- if .Paginator.NextURL}}<a href="{{.Paginator.NextURL}}">next →</a>{{end}}
</nav>
{{- end}}
{{end}}
```

Entries are paginated in the collection's `sort_by` order, and the split is deterministic across builds.
