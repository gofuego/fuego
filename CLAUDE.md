# CLAUDE.md — Fuego Contributor Guide

## What is Fuego?

Fuego is a **meta-engine for static site generation** in Go. Unlike Hugo or Eleventy, Fuego is format-agnostic — it does not bake in Markdown as the primary content format. Users define arbitrary DSLs (`.trivia`, `.card`, `.pitch`, anything) and map them to HTML through a configurable parsing and rendering pipeline.

The core value proposition: **you define the format, Fuego handles the infrastructure** (discovery, parsing, routing, taxonomy indexing, collections, rendering, and serving).

## Architecture Decisions

### AD-1: Universal AST with free-form node types

**Decision:** All content parsers produce `[]core.Node` where `Type` is a free-form string. The engine never interprets node types — templates decide rendering.

**Why:** Fuego supports arbitrary DSLs. A trivia parser emits `question`/`answer` nodes, a flashcard parser emits `front`/`back` nodes. If the engine imposed a fixed AST schema (like "heading", "paragraph"), it would become Markdown-centric and defeat its purpose as a meta-engine. The tradeoff is that the default renderer produces generic `<div data-type="...">` wrappers, which are functional but ugly. Users are expected to provide per-type renderer templates (`theme/renderers/{type}.html`) for production sites.

**Exception:** The built-in Markdown parser produces `Node{Type: "html"}` nodes, which the default renderer passes through as raw HTML without wrapping. This is the only node type the renderer treats specially.

### AD-2: YAML frontmatter envelope — not pluggable

**Decision:** Every content file uses `---` delimited YAML frontmatter. The envelope format is not configurable.

**Why:** The envelope is the universal metadata carrier across all content types. Making it pluggable (TOML, JSON) would add complexity with no user-facing benefit — YAML covers all use cases, and standardizing on one format keeps the parsing code simple. The payload below `---` is what's custom per DSL.

### AD-3: Three-tier parser precedence

**Decision:** Parser priority is: compiled Go parsers > declarative regex parsers > built-in parsers.

**Why:** Compiled parsers are the most powerful (full Go code), declarative parsers are the easiest (config-only regex rules), and built-ins are the fallback (Markdown). A user who registers a compiled `md` parser wants it to replace the built-in one. A user who defines a declarative `md` parser in config wants it to replace the built-in but not a compiled one. The merge happens once during INIT in `pipeline.RunUntil()`.

### AD-4: `core.Page` as the pipeline spine

**Decision:** `core.Page` is a mutable pointer struct that flows through every pipeline phase. Each phase enriches it: DISCOVER sets paths, PARSE sets envelope/nodes, ROUTE sets URL, INDEX may generate virtual pages.

**Why:** A single mutable struct avoids copying data between phases and keeps the pipeline linear. The struct lives in `core/` (not `internal/parse`) so that the public hook API can reference it. It was originally `parse.PageData` and was renamed when hooks were introduced.

### AD-5: Hooks are Go-only, not config-driven

**Decision:** `AfterParse` and `BeforeRender` hooks are Go functions registered via `eng.AfterParse(fn)` / `eng.BeforeRender(fn)`. There is no config-based hook mechanism.

**Why:** Hooks transform typed Go structs (`[]*core.Page`). Shell-based hooks would require JSON serialization round-trips, lose type safety, and add latency. The existing `prebuild` config field is shell-based because it runs before any pipeline data exists — a fundamentally different concern.

### AD-6: Error accumulation, not fail-fast

**Decision:** The pipeline accumulates errors via `core.ErrorAccumulator` with three severity levels: Warning (logged, build continues), LocalFatal (page skipped, build continues), GlobalFatal (build halts).

**Why:** A site with 500 pages shouldn't fail completely because one file has a parse error. LocalFatal lets the build produce partial output. GlobalFatal is reserved for structural problems (URL collisions, config errors) where continuing would produce corrupt output.

### AD-7: Golden-file integration testing

**Decision:** Integration tests use a golden-file pattern: each fixture has `input/` and `golden/` directories. `go test -update` regenerates expected output.

**Why:** Golden files make it easy to see exactly what the pipeline produces and detect regressions through byte-for-byte comparison. They also serve as documentation of expected behavior. All fixtures run in parallel with `t.Parallel()`.

### AD-8: Virtual pages for taxonomies and collections

**Decision:** Taxonomy term pages, taxonomy index pages, and collection pages are generated as virtual `core.Page` structs appended to the page list during INDEX. They use special types (`taxonomy-term`, `taxonomy-index`, `collection`) and are excluded from taxonomy term scanning in the manifest.

**Why:** Virtual pages go through the same RENDER phase as real pages — no special rendering path. Their nodes (`page-ref`, `term-ref`) carry metadata in attributes, and templates render them however they want. This keeps the pipeline uniform.

## Project Structure

```
fuego/
  core/                    Shared types (Page, Node, Parser, Hooks, Errors)
  engine/                  Public API (Engine, Register, AfterParse, BeforeRender, Run)
  cmd/fuego/               CLI binary entry point
  internal/
    cli/                   Cobra commands (build, serve, validate, list, init)
    config/                YAML config loading and validation
    discover/              File discovery, ignore patterns, content/asset classification
    parse/                 Frontmatter splitting, declarative parser, Markdown parser
    route/                 URL resolution (3-tier), collision detection
    index/                 Taxonomy and collection virtual page generation
    render/                Template loading, node rendering, static file copying
    manifest/              site-manifest.json generation
    pipeline/              Build orchestration (phase sequencing, hook execution)
    serve/                 Dev server (HTTP handler, file watcher, subprocess manager)
    scaffold/              Project scaffolding (embedded templates)
  testdata/                Integration test fixtures (input/ + golden/ per fixture)
  docs/                    PRD and design documents
```

## Package Dependency Rules

- `core/` has zero internal dependencies — it is the shared vocabulary
- `engine/` imports `core/` and `internal/cli/` only
- All `internal/` packages import `core/` freely
- `internal/pipeline/` is the only package that imports most other internal packages
- `internal/cli/` imports `pipeline/`, `config/`, and `serve/`
- No `internal/` package imports `engine/` (would create a cycle)

## Build Pipeline Phases

```
PREBUILD       →  Shell command from config (npm, tailwind, etc.)
INIT           →  Merge built-in + declarative + compiled parsers
DISCOVER       →  Walk content dir, apply ignore patterns, classify files
PARSE          →  Split frontmatter, dispatch to parsers (concurrent via errgroup)
AFTER-PARSE    →  User hooks: enrich/filter pages before routing
ROUTE          →  Resolve URLs (slug > pattern > filesystem), detect collisions
INDEX          →  Generate taxonomy + collection virtual pages, re-check collisions
BEFORE-RENDER  →  User hooks: final transforms before HTML generation
RENDER         →  Execute templates (concurrent via errgroup)
MANIFEST       →  Write site-manifest.json
STATIC         →  Copy public/ dir + colocated binary assets
```

`pipeline.RunUntil(phase)` allows partial execution. `validate` and `list` commands run through INDEX without rendering.

## Key Conventions

### Concurrency
- PARSE and RENDER phases use `errgroup` with `SetLimit(runtime.NumCPU())`
- Errors are collected in pre-allocated slices by index, not via channels
- The `core.ErrorAccumulator` is mutex-protected

### Determinism
- All map iterations use `sortedKeys()` helpers for deterministic output
- Pages in manifest are sorted by URL
- Integration tests verify determinism with `go test -count=3`

### Testing
- Unit tests live next to their source files (`*_test.go`)
- Integration tests: `integration_test.go` at the root, fixtures in `testdata/`
- Golden files: regenerate with `go test -run TestIntegrationFixtures -update`
- `fixtureParserRegistry()` in `integration_test.go` maps fixture names to compiled parsers
- Race detector: always run `go test ./... -race` before merging

### Content Files
- YAML frontmatter between `---` delimiters
- `type` field in frontmatter overrides file extension for parser dispatch
- `layout` field in frontmatter selects a layout template
- `slug` field in frontmatter overrides the URL slug segment

### Templates
- `theme/base.html` is required — it's the HTML shell
- `theme/layouts/{name}.html` override the `"content"` block defined in base
- `theme/renderers/{type}.html` override per-node-type rendering
- Template data: `.Page` (Envelope, Content, URL, Layout, Type), `.Site` (Name, BaseURL), `.JSON`
- Built-in template functions: `render` (recursive node rendering), `safeHTML`

## Common Tasks

### Adding a new pipeline phase
1. Add the phase constant to the `Phase` enum in `pipeline.go`
2. Implement the phase logic in `RunUntil()`
3. Add any new hook points if needed (update `core/hooks.go`, `engine.go`, thread through CLI)
4. Update golden files: `go test -run TestIntegrationFixtures -update`

### Adding a new built-in parser
1. Create `internal/parse/{name}.go` implementing `core.Parser`
2. Add it to `BuiltinParsers()` in `internal/parse/markdown.go` (or a new builtins file)
3. Add unit tests in `internal/parse/{name}_test.go`
4. The three-tier merge in `RunUntil()` handles the rest automatically

### Adding a new CLI command
1. Create `internal/cli/{name}.go` with `newXxxCmd(parsers, hooks, configPath)`
2. Register it in `newRootCmd()` in `root.go`
3. The command receives parsers, hooks, and config path — call `pipeline.Build()` or `RunUntil()` as needed

### Adding a new config field
1. Add the field to the appropriate struct in `internal/config/config.go`
2. Add defaults in `applyDefaults()` if needed
3. Add validation in the appropriate validate function if needed
4. Update `internal/config/config_test.go`

### Adding a new integration test fixture
1. Create `testdata/{name}/input/` with config.yaml, content files, and theme
2. If the fixture needs compiled parsers, add it to `fixtureParserRegistry()` in `integration_test.go`
3. Run `go test -run TestIntegrationFixtures/{name} -update` to generate golden files
4. Inspect golden output for correctness

## External Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/yuin/goldmark` | Built-in Markdown parser (GFM) |
| `github.com/bmatcuk/doublestar/v4` | Globstar pattern matching (ignore, collections) |
| `gopkg.in/yaml.v3` | YAML parsing (config, frontmatter) |
| `golang.org/x/sync` | errgroup for concurrent PARSE/RENDER |
| `github.com/fsnotify/fsnotify` | File watching for dev server |

## What NOT to Do

- **Don't add node-type-specific logic to the engine.** The engine is format-agnostic. If you need to handle a specific node type, do it in a renderer template or a hook.
- **Don't break the `core/` zero-dependency rule.** `core/` must not import any internal package.
- **Don't use channels for error collection.** The pipeline uses pre-allocated slices indexed by position for lock-free parallel error collection.
- **Don't skip collision re-checking after INDEX.** Virtual pages can collide with real pages or each other.
- **Don't add config knobs for things that work fine as code.** Hooks are Go code, not config. Parser extensions are Go code, not config. Keep config for declarative concerns (routes, patterns, site metadata).
