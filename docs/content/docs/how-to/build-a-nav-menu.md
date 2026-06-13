---
title: Build a Navigation Menu
layout: doc
nav_section: "Guides"
nav_weight: 2
tags:
  - how-to
  - templates
---

Templates can see every page on the site through `.Site.Pages` — no Go code needed for navs, listings, or related-content blocks.

## What `.Site.Pages` contains

A URL-sorted slice of slim page references, one per page (including taxonomy and collection virtual pages, excluding pages marked `Skip`):

| Field | Description |
|---|---|
| `.URL` | Resolved URL |
| `.Type` | Content type |
| `.Layout` | Layout name |
| `.Envelope` | Frontmatter map |

Refs deliberately omit page bodies: they exist for navigation and querying, not for rendering other pages' content.

## A nav partial

Create `theme/partials/nav.html`:

```html
<nav>
    <ul>
    {{- range sortBy (where .Site.Pages "type" "doc") "weight"}}
        <li><a href="{{.URL}}">{{.Envelope.title}}</a></li>
    {{- end}}
    </ul>
</nav>
```

Call it from `base.html`:

```html
{{partial "nav" .}}
```

Pages opt into the menu with frontmatter:

```yaml
title: Install
type: doc
weight: 1
```

`where` filters by map key, struct field (case-insensitive), or envelope key — `"type"` matches the ref's `Type`, `"weight"` matches `Envelope["weight"]`. `sortBy` sorts numbers numerically and copies before sorting, so the shared ref slice is never mutated.

## Listing taxonomy terms

Virtual pages appear in the refs like any other page, so a tag cloud is a filter away:

```html
{{range .Site.Pages}}{{if eq .Type "taxonomy-term"}}
    <a href="{{.URL}}">{{.Envelope.title}}</a>
{{end}}{{end}}
```

## Related pages

Inside a page template, query against the current page's envelope:

```html
{{$current := .Page}}
{{range where .Site.Pages "category" $current.Envelope.category | limit 5}}
    {{if ne .URL $current.URL}}<a href="{{.URL}}">{{.Envelope.title}}</a>{{end}}
{{end}}
```
