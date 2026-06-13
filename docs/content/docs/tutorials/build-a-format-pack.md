---
title: "Tutorial: Build a Format Pack"
layout: doc
nav_section: "Tutorials"
nav_weight: 3
order: 3
tags:
  - tutorial
  - packs
  - go
---

A **format pack** bundles parsers, hooks, theme templates, and config defaults into one installable Go module. In this tutorial you'll build a small `note` pack that turns `.note` files into styled callouts, and wire it into a site with one line.

By the end you'll have a pack a stranger could `go get` and `eng.Use()`.

## 1. The parser

A note file is one callout per paragraph, tagged by kind:

```
info: Fuego ships zero built-in formats.
warning: Parsers are opt-in — register them in main.go.
```

Create `note.go` in your pack module:

```go
package notepack

import (
	"fmt"
	"strings"

	"github.com/FabioSol/fuego/core"
)

type noteParser struct{}

func (noteParser) Type() string { return "note" }

func (noteParser) Parse(raw []byte) (core.Envelope, []core.Node, error) {
	env, body, err := core.SplitFrontmatter(raw)
	if err != nil {
		return nil, nil, err
	}
	var nodes []core.Node
	for i, line := range strings.Split(strings.TrimSpace(string(body)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kind, text, ok := strings.Cut(line, ":")
		if !ok {
			return nil, nil, &core.ParseError{
				Line: i + 1,
				Err:  fmt.Errorf("note line missing \"kind:\" prefix"),
			}
		}
		nodes = append(nodes, core.Node{
			Type:       "note",
			Content:    strings.TrimSpace(text),
			Attributes: map[string]any{"kind": strings.TrimSpace(kind)},
		})
	}
	return env, nodes, nil
}
```

The `core.ParseError` makes a malformed line report `file:line` in the build output — a small DX win that costs nothing.

## 2. The theme

Pack templates live in an embedded filesystem mirroring a theme directory. Add `theme/renderers/note.html`:

```html
<div class="note note-{{.Attributes.kind}}">{{.Content}}</div>
```

Embed it in `notepack.go`:

```go
package notepack

import (
	"embed"
	"io/fs"

	"github.com/FabioSol/fuego/core"
)

//go:embed theme
var themeRoot embed.FS

func Pack() core.Pack {
	theme, _ := fs.Sub(themeRoot, "theme")
	return core.Pack{
		Name:    "note",
		Parsers: []core.Parser{noteParser{}},
		Theme:   theme,
	}
}
```

A user's own `theme/renderers/note.html` would override the pack's — packs lose to the user, always.

## 3. Config defaults

Let the pack route `.note` files without the user configuring anything. Add a `ConfigDefaults` YAML fragment:

```go
//go:embed config-defaults.yaml
var configDefaults []byte
```

```yaml
# config-defaults.yaml
routes:
  note: "/notes/{slug}"
```

Set `ConfigDefaults: configDefaults` on the returned `core.Pack`. The fragment is deep-merged beneath the user's `config.yaml`; the user can override `routes.note` and keep everything else. `fuego config` shows where each value came from.

## 4. Use it

In any Fuego site's `main.go`:

```go
eng := engine.New()
eng.Use(notepack.Pack())
```

Drop a `.note` file in `content/`, run `fuego build`, and the callouts render through the pack's theme — no parser code, no template, no route config in the consuming site.

## What you learned

- A pack is a plain Go module exporting `Pack() core.Pack`.
- It can carry parsers, hooks (`AfterParse`, `Index`, `BeforeRender`), an embedded theme, and config defaults.
- Precedence always favors the user: user parsers, user theme files, and user config win over the pack.

See [Format Packs](/docs/concepts/format-packs/) for the full reference and [Config Merging](/docs/config-merging/) for the merge rules.
