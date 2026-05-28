---
title: "Fuego — Format-Agnostic Static Sites"
layout: home
---

Define custom DSLs. Build static sites. No Markdown required.

Fuego is a meta-engine for static site generation in Go. You define the content format — `.trivia`, `.card`, `.pitch`, anything — and Fuego handles discovery, parsing, routing, taxonomy indexing, and rendering.

```bash
go run github.com/FabioSol/fuego/cmd/fuego@latest init mysite
cd mysite && go run . serve
```

Markdown works out of the box with full GFM support. But the real power is custom formats defined entirely in YAML — or in Go when you need full control. This site is built with Fuego.
