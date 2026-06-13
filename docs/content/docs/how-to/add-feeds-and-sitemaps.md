---
title: Add an RSS Feed and Sitemap
layout: doc
nav_section: "Guides"
nav_weight: 4
tags:
  - how-to
  - templates
---

Fuego has no built-in feed or sitemap generator — and doesn't need one. Any file you place in `theme/outputs/` is rendered as a template and written to the build root, so RSS, sitemaps, `robots.txt`, search indexes, and redirect maps are all the same one mechanism.

## How outputs work

Each file under `theme/outputs/` is executed as a **text** template (not HTML — so XML and JSON aren't corrupted by entity escaping) and written to the matching path in the output directory. Nested directories are preserved: `theme/outputs/feeds/news.xml` becomes `build/feeds/news.xml`.

Output templates receive `.Site`, including `.Site.Pages` — the same URL-sorted page references available in HTML templates — plus the data-shaping functions `where`, `sortBy`, `limit`, `first`, `dict`, and `dateFormat`.

## Sitemap

`theme/outputs/sitemap.xml`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
{{- range .Site.Pages}}
  <url><loc>{{$.Site.BaseURL}}{{.URL}}</loc></url>
{{- end}}
</urlset>
```

## RSS feed

`theme/outputs/feed.xml` — filter to the page type you publish, and format dates from the envelope:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>{{.Site.Name}}</title>
    <link>{{.Site.BaseURL}}</link>
    {{- range where .Site.Pages "type" "post"}}
    <item>
      <title>{{.Envelope.title}}</title>
      <link>{{$.Site.BaseURL}}{{.URL}}</link>
      {{- if .Envelope.date}}
      <pubDate>{{dateFormat "Mon, 02 Jan 2006 15:04:05 -0700" .Envelope.date}}</pubDate>
      {{- end}}
    </item>
    {{- end}}
  </channel>
</rss>
```

## robots.txt

`theme/outputs/robots.txt`:

```
User-agent: *
Allow: /
Sitemap: {{.Site.BaseURL}}/sitemap.xml
```

## Notes

- An output file that would overwrite a page's HTML (e.g. `outputs/about/index.html` when an `/about/` page exists) fails the build with a collision error.
- Outputs are not listed in `site-manifest.json` — the manifest indexes pages, not generated assets.
- A pack can ship outputs too; your `theme/outputs/` files override a pack's by the same name.
