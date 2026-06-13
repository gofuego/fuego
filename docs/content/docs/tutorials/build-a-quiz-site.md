---
title: "Tutorial: Build a Quiz Site"
layout: doc
nav_section: "Tutorials"
nav_weight: 1
order: 1
tags:
  - tutorial
  - parsers
  - templates
---

In this tutorial you'll build a trivia quiz site from scratch using a custom `.trivia` format. No Go code needed — everything is defined in `config.yaml`.

## 1. Scaffold the Project

```bash
go run github.com/FabioSol/fuego/cmd/fuego@latest init quiz-site
cd quiz-site
```

## 2. Define the Trivia Format

Open `config.yaml` and replace the `parsers` section:

```yaml
parsers:
  trivia:
    rules:
      - match: '^\?\s*(.+)$'
        emit:
          type: question
          content: "$1"
      - match: '^\[([A-Z])\]\s+(.+)$'
        emit:
          type: answer
          content: "$2"
          attributes:
            letter: "$1"
      - match: '^>\s*(.+)$'
        emit:
          type: hint
          content: "$1"
```

This defines three rules. Each line of a `.trivia` file is tested against the rules in order — first match wins. Capture groups (`$1`, `$2`) fill in the content and attributes.

## 3. Write a Trivia File

Create `content/history.trivia`:

```
---
title: History Quiz
tags:
  - history
---
? When was the Declaration of Independence signed?
[A] 1776
[B] 1789
[C] 1804
> Think American Revolution

? Who was the first Roman Emperor?
[A] Julius Caesar
[B] Augustus
[C] Nero
> Not the one who was assassinated
```

## 4. Add a Renderer Template

The parser produces `question`, `answer`, and `hint` nodes. Without custom renderers, they render as generic `<div data-type="...">` wrappers. Let's style them.

Create `theme/renderers/question.html`:

```html
<div class="question">{{.Content}}</div>
```

Create `theme/renderers/answer.html`:

```html
<div class="answer"><span class="letter">{{index .Attributes "letter"}}</span> {{.Content}}</div>
```

Create `theme/renderers/hint.html`:

```html
<details class="hint"><summary>Hint</summary>{{.Content}}</details>
```

## 5. Add Some CSS

Add to `public/style.css`:

```css
.question { font-size: 1.2rem; font-weight: 600; margin: 1.5rem 0 .5rem; }
.answer { padding: .25rem 0; }
.letter { display: inline-block; width: 1.5rem; font-weight: 700; color: #e94560; }
.hint { margin: .5rem 0 1rem; color: #888; }
```

## 6. Build and Preview

```bash
go run . serve
```

Open `http://localhost:8080/history/` — you'll see your trivia questions rendered with styled answers and collapsible hints.

## What Just Happened?

You defined a content format entirely in YAML. No Go code, no plugins, no build scripts. The pipeline:

1. **DISCOVER** found `history.trivia` in `content/`
2. **PARSE** matched it to the `trivia` declarative parser, ran each line through the regex rules, and produced `question`, `answer`, and `hint` nodes
3. **ROUTE** used the filesystem mirror to assign the URL `/history/`
4. **RENDER** found your renderer templates and used them instead of the default `<div>` wrappers

## Next Steps

- Add route patterns: `routes: { trivia: "/quiz/{slug}" }` → changes the URL to `/quiz/history/`
- Add taxonomies on the `tags` field to auto-generate tag pages
- Add more trivia files and use a collection to build a quiz index page
