---
title: "The meta-engine for static sites"
layout: home
---

Most static site generators bake in Markdown. **Fuego lets you define the format** — `.trivia`, `.card`, `.recipe`, anything — then use it to build a site, or package it as a **pack** the next person installs with one line.

```bash
go run github.com/gofuego/fuego/cmd/fuego@latest init mysite
cd mysite && go run . serve
```

Discovery, routing, taxonomies, pagination, feeds, a dev server — Fuego handles the infrastructure every generator needs. You bring the format. Markdown is a first-party parser you opt into; no format is privileged. This very site is built with Fuego.

