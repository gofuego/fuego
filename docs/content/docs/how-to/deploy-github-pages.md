---
title: Deploy to GitHub Pages
layout: doc
tags:
  - how-to
  - deployment
---

## Prerequisites

- A GitHub repository with your Fuego site
- GitHub Pages enabled in repo settings (source: "GitHub Actions")

## 1. Set the Base URL

In `config.yaml`, set `base_url` to your repository name:

```yaml
site:
  name: "My Site"
  base_url: "/my-repo"
```

GitHub Pages serves your site at `https://username.github.io/my-repo/`, so all links and asset paths need this prefix.

## 2. Use BaseURL in Templates

Make sure your `theme/base.html` uses `{{.Site.BaseURL}}` for all links:

```html
<link rel="stylesheet" href="{{.Site.BaseURL}}/style.css">
<a href="{{.Site.BaseURL}}/index/">Home</a>
```

## 3. Add a Root Redirect

The engine generates pages at `/{slug}/index.html`, so there's no `index.html` at the root. Add `public/index.html` to redirect:

```html
<!DOCTYPE html>
<html>
<head><meta http-equiv="refresh" content="0;url=index/"></head>
<body></body>
</html>
```

## 4. Create the Workflow

Create `.github/workflows/docs.yml`:

```yaml
name: Deploy Site

on:
  push:
    branches: [main]

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: true

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
      - run: go run . build
      - uses: actions/upload-pages-artifact@v3
        with:
          path: build

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v4
```

## 5. Enable GitHub Pages

Go to your repository's Settings > Pages and set the source to **"GitHub Actions"**.

Push to `main` and the workflow will build and deploy your site automatically.
