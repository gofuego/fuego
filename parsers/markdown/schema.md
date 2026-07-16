# markdown — parser contract

The engine's co-versioned default format: Markdown with YAML frontmatter,
rendered to HTML by goldmark with GFM extensions (tables, strikethrough,
autolinks, task lists). It lives in the engine repo
(`github.com/gofuego/fuego/parsers/markdown`) rather than fuego-formats — the
most common case needs no second module — but this contract follows the
fuego-formats `schema.md` convention so themes and agents read every format the
same way.

## Claims

By default the parser claims the bare extension `md` (`Parser.Type()`), not a
filename pattern. Under specificity-ordered dispatch this is the least
specific claim tier: any registered filename pattern that matches wins first,
so a compound-suffix format like `*.adr.md` safely outranks it — plain `.md`
files still route here.

Override the claim with `markdown.WithPatterns(...)`:

```go
eng.Register(markdown.Parser(markdown.WithPatterns("*.markdown")))
eng.Register(markdown.Parser(markdown.WithPatterns("README.md")))
```

Patterns **replace** the claim entirely — with `WithPatterns("README.md")` the
parser claims only files named `README.md`, and unclaimed `.md` files become
assets. Claims match base names only — no path scoping, no content sniffing.

## Envelope keys

The parser writes **no keys of its own**. The envelope is the file's YAML
frontmatter (`---`-delimited), passed through verbatim as parsed by YAML; a
file without frontmatter gets an empty envelope. Malformed frontmatter YAML is
a parse error for the page.

Conventional keys the engine reads if present: `title`, `layout`, `slug`.
Frontmatter values pass through as YAML parses them — strings, numbers, bools,
dates, and nested maps/lists all stay cache-eligible for incremental builds.

## Node types

| Constant | Value | Content | Raw | Attributes |
|----------|-------|---------|-----|------------|
| `markdown.NodeHTML` | `html` | the rendered HTML of the whole payload | `true` | none |

One node per file; a file whose payload renders to nothing (empty, or
frontmatter only) emits **zero** nodes. The unprefixed `html` value predates
the fuego-formats slug-prefix convention and is kept as an engine-native
exemption — existing themes key `theme/renderers/html.html` on it.

## Tree shape

One page, zero or one node. Not a TreeParser — a markdown file never expands
into child pages.

## Slug derivation

The parser emits no slugs, routes, or titles: routing is the engine's job
(explicit `slug` frontmatter > config route patterns > filesystem mirror, with
`index` files routing to their directory root). `title` appears only if the
frontmatter sets it.

## Stability

Co-versioned with the engine module (`github.com/gofuego/fuego`), not
independently tagged: pre-1.0, node types and claim behavior may change
between engine minor versions, and any such change is called out in the engine
release notes. The machine-checked form of this contract is the golden dump
fixture pair under `testdata/` (`*.md` input → `*.golden.json` envelope+nodes
dump), regenerated with `go test ./parsers/markdown -update`.
