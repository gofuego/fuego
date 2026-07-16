# CLAUDE.md — Fuego Contributor Guide

## What is Fuego?

Fuego is a **meta-engine for static site generation** in Go. Unlike Hugo or Eleventy, Fuego is format-agnostic — it does not bake in Markdown as the primary content format. Users define arbitrary DSLs (`.trivia`, `.card`, `.pitch`, anything) and map them to HTML through a configurable parsing and rendering pipeline.

The core value proposition: **you define the format, Fuego handles the infrastructure** (discovery, parsing, routing, taxonomy indexing, collections, rendering, and serving).

## Architecture Decisions

> These AD-N entries summarize the canonical Architecture Decision Records in
> [`adrs/`](adrs/); the ADR file is authoritative where they differ. The AD numbering
> is historical and no longer 1:1 with the ADR files — use this map:
> AD-1→ADR-001, AD-2→ADR-002, AD-3→ADR-003, AD-4→ADR-004, AD-4b→ADR-005,
> AD-4c→ADR-006, AD-5→ADR-007, AD-6→ADR-008, AD-7→ADR-009, AD-8→ADR-010,
> AD-9→ADR-011, AD-10→ADR-012, AD-11→ADR-013, AD-12→ADR-014, AD-13→ADR-015,
> AD-14→ADR-016, AD-15→ADR-017, AD-16→ADR-018, AD-17→ADR-019.

### AD-1: Universal AST with free-form node types

**Decision:** All content parsers produce `[]core.Node` where `Type` is a free-form string. The engine never interprets node types — templates decide rendering.

**Why:** Fuego supports arbitrary DSLs. A trivia parser emits `question`/`answer` nodes, a flashcard parser emits `front`/`back` nodes. If the engine imposed a fixed AST schema (like "heading", "paragraph"), it would become Markdown-centric and defeat its purpose as a meta-engine. The tradeoff is that the default renderer produces generic `<div data-type="...">` wrappers, which are functional but ugly. Users are expected to provide per-type renderer templates (`theme/renderers/{type}.html`) for production sites.

Nodes can be marked `Raw: true` to pass their content through the default renderer as raw HTML without wrapping or escaping. Any parser can use this flag — it is not tied to a specific node type.

### AD-2: Parser-owned envelope extraction

**Decision:** Parsers own envelope extraction. The `Parser` interface is `Parse(raw []byte) (Envelope, []Node, error)` — the parser receives the entire raw file and returns both metadata and parsed nodes. Convenience wrappers (`core.WithYAMLFrontmatter`, `core.WithNoEnvelope`) handle common patterns.

**Why:** Fuego is a meta-engine used to build domain-specific SSGs. Some formats use YAML frontmatter (Markdown blogs), some have metadata embedded in their content (Kubernetes manifests), and some have no metadata at all (Dockerfiles). The parser knows its format best. `core.SplitFrontmatter()` is a public helper for parsers that want YAML frontmatter.

### AD-3: Two-tier parser precedence

**Decision:** Parser priority is: compiled Go parsers > declarative regex parsers. There are no built-in parsers — the engine ships with zero format opinions.

**Why:** Compiled parsers are the most powerful (full Go code), declarative parsers are the easiest (config-only regex rules). Markdown support is opt-in via `parsers/markdown/`. A user who registers a compiled parser for the same extension as a declarative parser overrides it. The merge happens once during INIT in `pipeline.RunUntil()`.

### AD-4: `core.Page` as the pipeline spine

**Decision:** `core.Page` is a mutable pointer struct that flows through every pipeline phase. Each phase enriches it: DISCOVER sets paths, PARSE sets envelope/nodes, ROUTE sets URL, INDEX may generate virtual pages.

**Why:** A single mutable struct avoids copying data between phases and keeps the pipeline linear. The struct lives in `core/` (not `internal/parse`) so that the public hook API can reference it. It was originally `parse.PageData` and was renamed when hooks were introduced.

### AD-4b: Extension-only parser dispatch

**Decision:** Parser dispatch is determined by file extension (matching `Parser.Type()`) or filename (matching `FilenameParser.Filenames()`). The `type` frontmatter field does not drive parser dispatch.

**Why:** With parsers owning envelope extraction, the envelope isn't available before dispatch. Extension-based dispatch is explicit, visible in the filesystem, and needs no pre-parsing. Parsers implementing `FilenameParser` can handle extensionless files like `Dockerfile`.

### AD-4c: Content discovery driven by parsers

**Decision:** A file is classified as content if and only if a registered parser matches its extension or filename. There are no hardcoded content extensions.

**Why:** Without built-in parsers, hardcoded extension lists would create contradictions (discovering `.md` files as content with no parser to handle them). Parsers are the single source of truth for what's content.

### AD-5: Hooks are Go-only, not config-driven

**Decision:** `AfterParse`, `Index`, and `BeforeRender` hooks are Go functions registered via `eng.AfterParse(fn)` / `eng.Index(fn)` / `eng.BeforeRender(fn)`. There is no config-based hook mechanism.

**Why:** Hooks transform typed Go structs (`[]*core.Page`). Shell-based hooks would require JSON serialization round-trips, lose type safety, and add latency. The existing `prebuild` config field is shell-based because it runs before any pipeline data exists — a fundamentally different concern. The `Index` hook runs during INDEX (after ROUTE, before the collision re-check) and is the supported place to add virtual pages — pages added there are collision-checked, unlike pages injected in `BeforeRender`.

### AD-6: Error accumulation, not fail-fast

**Decision:** The pipeline accumulates errors via `core.ErrorAccumulator` with three severity levels: Warning (logged, build continues), LocalFatal (page skipped, build continues), GlobalFatal (build halts).

**Why:** A site with 500 pages shouldn't fail completely because one file has a parse error. LocalFatal lets the build produce partial output. GlobalFatal is reserved for structural problems (URL collisions, config errors) where continuing would produce corrupt output.

### AD-7: Golden-file integration testing

**Decision:** Integration tests use a golden-file pattern: each fixture has `input/` and `golden/` directories. `go test -update` regenerates expected output.

**Why:** Golden files make it easy to see exactly what the pipeline produces and detect regressions through byte-for-byte comparison. They also serve as documentation of expected behavior. All fixtures run in parallel with `t.Parallel()`.

### AD-8: Virtual pages for taxonomies and collections

**Decision:** Taxonomy term pages, taxonomy index pages, and collection pages are generated as virtual `core.Page` structs appended to the page list during INDEX. They use special types (`taxonomy-term`, `taxonomy-index`, `collection`) and are excluded from taxonomy term scanning in the manifest.

**Why:** Virtual pages go through the same RENDER phase as real pages — no special rendering path. Their nodes (`page-ref`, `term-ref`) carry metadata in attributes, and templates render them however they want. This keeps the pipeline uniform.

### AD-9: Format packs bundle a content format as one registerable unit

**Decision:** `core.Pack` bundles parsers, hooks, an embedded theme FS (templates + a `static/` subtree), and a `ConfigDefaults` YAML fragment. Registered via `eng.Use(pack)`. An optional `Init(ctx, *PackContext)` lifecycle reads the pack's `packs.{name}` config subtree. Precedence: user-registered parsers and user theme files always win over packs; among packs, later registration wins (with a warning).

**Why:** Packs are the ecosystem unit — they let a domain-specific format (ADRs, K8s manifests) ship as an installable Go module that a vanilla project consumes with one line. `core.Pack` stays in `core/` (stdlib-only, no internal deps). Pack `ConfigDefaults` are deep-merged under the user config (`config.ResolveLayers`); pack `static/` assets are copied during STATIC; the theme FS layers under the user theme dir in `render.LoadTemplates`.

### AD-10: Incremental builds are opt-in and provably equivalent

**Decision:** `Options.Incremental` (and the dev server, always) reuses parsed pages via `internal/buildcache` and narrows RENDER to the affected set. Default `build` is clean and cache-free. Any change to the engine binary, resolved config, or theme invalidates the cache and triggers a full rebuild.

**Why:** Big-site dev rebuilds are the one place users feel a real gap. The cache is safe by construction (the header pins the build environment) and the narrowing is guarded by a byte-equivalence test suite that builds every fixture clean and then incrementally under every mutation class. A corrupt or version-mismatched cache is a miss, never an error.

### AD-11: A programmatic build API alongside the CLI

**Decision:** `engine.Build/Serve/Validate` + `engine.BuildOptions` let a Go program build in-process. `engine.Run(args)` (the CLI) reads a `config.yaml` file; the programmatic API resolves config from in-memory layers (pack defaults → optional file → option overrides) via `config.ResolveLayers`.

**Why:** Tools built on Fuego (and pack-based wrappers) shouldn't have to synthesize a temp config file and shell into the CLI. The programmatic API is the supported way to embed Fuego; the serve loop lives in `internal/serve.Run` so the CLI and `engine.Serve` share it. See `docs/` "Embedding Fuego".

### AD-12: The site manifest is the host-integration contract (ADR-014)

**Decision:** `internal/manifest` writes `site-manifest.json` as the stable contract a host (e.g. fuego-studio) reads. The engine exposes **build facts only** — per page `url`, `type`, `layout`, `title`, `summary`, `output_path`, `source_path`, and the flattened `envelope`, plus a top-level `content_root` (the content dir relative to the enclosing git root). A host maps a page back to its repo source as `content_root` + `source_path`; editing policy (who may edit, which branch) lives in the host.

**Why:** Cleanly splits build facts (engine) from workflow policy (host), keeping the engine editing-agnostic. The manifest is a public contract — changing a field is a breaking change for hosts that read it.

### AD-13: Index files route to their directory root (ADR-015)

**Decision:** In the filesystem-mirror tier a file named `index` is the root of its directory — `content/index.md` → `/`, `content/blog/index.md` → `/blog/` — matching the universal SSG convention. The explicit `slug` and config-route tiers are unchanged, and the old `public/index.html` → `/index/` redirect hack is removed from the scaffold and theme packs.

**Why:** Home pages live at `/` with no redirect, one fewer file for every site to carry, and no base-URL breakage. Note: an existing `public/index.html` redirect is now *harmful* — copied during STATIC it would clobber the rendered root `index.html` — and must be removed.

### AD-14: Internal link checking resolves against the built output (ADR-016)

**Decision:** `fuego build --check-links` runs after output is written and resolves every `<a href>` exactly as a browser would — honoring each page's `<base href>` and the site base URL — then verifies the target exists in the output. It validates the shipped HTML, so it catches both dangling and base-escaping links from content or templates. Broken links are reported against the source page; warnings by default, `--strict-links` promotes them to a fatal error for CI. Anchors and external links are out of scope.

**Why:** Catches broken internal links at build time, attributed to the file to edit — far more traceable than a runtime 404. Cheap (hash lookups over in-memory HTML); most valuable run with the real `--base-url`, which surfaces the base-escape class a root build can't see.

### AD-15: The build cache stores JSON-shaped deep copies, degrading per page (ADR-017)

**Decision:** Both cache boundaries deep-copy envelopes and node trees, so the cache holds post-PARSE state only — hook mutation never reaches it, stale hook output never leaves it. Cacheable envelope values are JSON-shaped (the gob-registered composite set in `internal/buildcache`); a page holding any other concrete type is dropped from the cache individually, warned by name — never a build error.

**Why:** `ParsedPage` used to share references with live pages while the cache saved after hooks ran: hook-built values failed the whole gob encode, and cache hits restored the previous build's hook products, with correctness resting on hook idempotence. Deep copies make the contract structural; per-page degradation means one exotic envelope (a pack's private struct) costs that page its reuse, not the whole site's. Parsers wanting caching keep display data JSON-shaped.

### AD-16: Specificity-ordered parser dispatch across a shared resolver (ADR-018)

**Decision:** Filename-pattern claims are checked before bare-extension claims, in both discovery classification and parse dispatch; among multiple matching patterns the longest pattern string wins, ties resolving by existing parser precedence (user > later pack > earlier pack, declarative lowest). A parser claims by exactly one kind: declared patterns are the parser's **complete** claim set (its `Type()` is not implicitly claimed as an extension); a parser without patterns claims `Type()` as a bare extension. The claim logic lives in one resolver (`internal/dispatch`) consumed by both phases. `page.Type` remains the matched parser's `Type()`.

**Why:** With reusable format parsers, claims overlap: a markdown parser claims `md` while an ADR parser claims `*.adr.md`. Extension-first dispatch silently routed `guide.adr.md` to markdown — the more specific claim never got a look. Longest-pattern-wins is deterministic where registration order is not, and a single resolver keeps "is this content?" and "who parses it?" from drifting apart. Behavior is unchanged for sites without overlapping claims. This amends AD-4b/AD-4c (ADR-005/006 lineage).

### AD-17: TreeParser expands one artifact into a tree of real pages (ADR-019)

**Decision:** A new optional interface alongside `Parser`, `core.TreeParser`,
returns a `core.PageTree` — a root (envelope + nodes) plus `Children` keyed by
relative slug path, nested arbitrarily. The engine detects it by interface
assertion at PARSE (no registration change; plain `Parser`s are untouched) and
expands each tree node into a **real `core.Page`**: child `RelPath` =
source file's `RelPath` + "/" + slug path (child `SourcePath` is the artifact);
child `URL` = the root's routed URL + slug-path segments, composed in a second
ROUTE pass **after** the root goes through the normal three-tier routing (so the
index-file convention AD-13 on the root is honored). Children carry their own
envelopes, so taxonomies/collections/pagination see them natively through INDEX.
Sibling-slug collisions inside a tree and child-vs-page collisions surface
through the existing ROUTE/INDEX collision detection (GlobalFatal). This retires
"one file = one page" as a parse-contract invariant (amends AD-4/ADR-004).

**Cache (amends AD-15/ADR-017):** all pages of a tree are stored under that one
artifact's content-hash entry (`ParsedPage.Tree`), so an unchanged file restores
its whole tree from cache (skipped) and a changed file re-parses and re-renders
exactly its tree. Deep-copy isolation holds at both boundaries for every page.
Degradation is per ENTRY: an ordinary file degrades per page as before, but a
tree with any non-JSON-shaped child envelope drops the whole file's entry (a
missing child on a hit would silently change the output) — a warning, never an
error. **Manifest (amends AD-12/ADR-014):** every page of a tree lists the
shared root artifact's `RelPath` as its `source_path` — a deliberate
multiple-entries-per-source contract, so fuego-studio's `SourcePath != ""` guard
treats each child as editable-as-the-artifact; root and children stay
distinguishable by url/output_path.

**Why:** A rich artifact (an OpenAPI spec, a DBML schema) deserves to be a whole
*section* — an index plus a routed, taxonomy-visible, stably-URL'd page per
operation/table/suite. The prior virtual-page workaround produced pages
taxonomies couldn't see and incremental builds re-rendered constantly. Envelope
convention for library tree parsers: JSON-shaped values only; a missing
per-child layout falls back to the base template silently.

## Project Structure

```
fuego/
  core/                    Shared types (Page, Node, Parser, Pack, Hooks, Errors, Paginator, ParseError, SplitFrontmatter, Wrappers, PageTree, TreeParser)
  engine/                  Public API: CLI (Run) + programmatic build (Build/Serve/Validate, BuildOptions); Register, Use, AfterParse, Index, BeforeRender
  parsers/markdown/        First-party Markdown parser (opt-in, not built-in)
  cmd/fuego/               CLI binary entry point
  internal/
    cli/                   Cobra commands (build, serve, validate, list, config, init [--pack|--formats], formats add/sync)
    formats/               Format-module resolver, generated formats.go, docs materializer
    config/                YAML config loading, validation, layer merge + provenance (Resolve/ResolveLayers)
    discover/              File discovery, ignore patterns, content/asset classification
    parse/                 Parse orchestration, declarative parser, cache-aware ParseAllCached
    route/                 URL resolution (3-tier), collision detection
    index/                 Taxonomy + collection virtual pages, pagination
    render/                Template loading (theme + pack layers), node rendering, outputs, static copy, .JSON/.Site.Pages detection
    manifest/              site-manifest.json generation
    buildcache/            Incremental-build cache (gob; header + per-file content hashes; orphan removal)
    pipeline/              Build orchestration (phase sequencing, hook execution, Options)
    serve/                 Dev server (HTTP handler, file watcher, subprocess; reusable Run loop)
    scaffold/              Project scaffolding (embedded templates; --pack wiring)
  testdata/                Integration test fixtures (input/ + golden/ per fixture)
  docs/                    Self-hosted documentation site (a Fuego project, dogfooding)
```

## Package Dependency Rules

- `core/` has zero internal dependencies — it is the shared vocabulary
- `engine/` imports `core/` and `internal/` packages (`cli`, `config`, `pipeline`, `serve`) — it exposes both the CLI (`Run`) and a programmatic build API (`Build`/`Serve`/`Validate`)
- All `internal/` packages import `core/` freely
- `internal/pipeline/` is the only package that imports most other internal packages
- `internal/cli/` imports `pipeline/`, `config/`, and `serve/`
- No `internal/` package imports `engine/` (would create a cycle)

## Build Pipeline Phases

```
PREBUILD       →  Shell command from config (npm, tailwind, etc.)
INIT           →  Merge declarative + compiled parsers; run pack Init lifecycle
DISCOVER       →  Walk content dir, apply ignore patterns, classify by registered parsers
PARSE          →  Dispatch to parsers by extension/filename (concurrent; cache-aware)
AFTER-PARSE    →  User hooks: enrich/filter pages before routing
ROUTE          →  Resolve URLs (slug > pattern > filesystem), detect collisions
INDEX          →  Taxonomy + collection virtual pages, pagination, Index hooks, re-check collisions
BEFORE-RENDER  →  User hooks: final transforms before HTML generation
RENDER         →  Execute templates (concurrent; narrowed on incremental rebuilds)
OUTPUTS        →  Render theme/outputs/ (feeds, sitemaps) as text templates
MANIFEST       →  Write site-manifest.json
STATIC         →  Copy pack static/, then public/, then colocated binary assets
```

`pipeline.RunUntil(phase)` allows partial execution. `validate` and `list` commands run through INDEX without rendering.

**Incremental builds** (`Options.Incremental`, serve always): `internal/buildcache` keeps an on-disk gob cache of post-PARSE pages keyed by a header (engine binary hash + resolved config hash + theme tree hash) plus per-file content hashes. Unchanged content skips PARSE; a header mismatch falls back to a full, clean rebuild. RENDER is narrowed to the affected set (changed pages + virtual pages + pages whose template reads `.Site.Pages`, detected via `render/sitedetect.go`). Output stays byte-identical to a clean build — enforced by the equivalence suite in `incremental_test.go`.

## Key Conventions

### Concurrency
- PARSE and RENDER phases use `errgroup` with `SetLimit(runtime.NumCPU())`
- Errors are collected in pre-allocated slices by index, not via channels
- The `core.ErrorAccumulator` carries no lock — concurrent phases collect into index-keyed slices and the accumulator is filled serially after the errgroup completes, so it is lock-free by construction

### Determinism
- All map iterations use `sortedKeys()` helpers for deterministic output
- Pages in manifest are sorted by URL
- Integration tests verify determinism with `go test -count=3`

### Site manifest
- `internal/manifest` writes `site-manifest.json`. Each page entry carries `url`, `type`, `layout`, `title`, `summary`, `output_path` (`<url>/index.html`), `source_path` (content-dir-relative, forward slashes; the page's `RelPath`, or — for a tree-parsed file's pages — the shared root artifact's `RelPath`, so multiple entries map to one source; empty for virtual pages so they're non-editable), and the flattened `envelope`
- The top-level `content_root` is the content dir relative to the enclosing git root (empty outside a git repo). A host maps a page's repo-relative source to `content_root` + `source_path` — this is what fuego-studio uses to fetch/commit a page's source
- The `build`/`serve` `--base-url` flag overrides `site.base_url` for a single run (override only when the flag is set), so a deploy can target a subpath without a separate config file

### Testing
- Unit tests live next to their source files (`*_test.go`)
- Integration tests: `integration_test.go` at the root, fixtures in `testdata/`
- Golden files: regenerate with `go test -run TestIntegrationFixtures -update`
- `fixtureParserRegistry()` in `integration_test.go` maps fixture names to compiled parsers
- Race detector: always run `go test ./... -race` before merging

### Content Files
- Parsers own envelope extraction — YAML frontmatter is one option via `core.WithYAMLFrontmatter`, not a requirement
- Parser dispatch is by file extension (`Parser.Type()`) or filename (`FilenameParser.Filenames()`)
- `layout` and `slug` are envelope key conventions read by the pipeline after parsing
- `core.SplitFrontmatter()` is a public helper for parsers that use YAML frontmatter

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

### Adding a new first-party parser package
1. Create `parsers/{name}/{name}.go` exporting a `Parser() core.Parser` function
2. Use `core.WithYAMLFrontmatter`, `core.WithNoEnvelope`, or implement `core.Parser` directly
3. For extensionless files, implement `core.FilenameParser` with `Filenames() []string`
4. Add unit tests in `parsers/{name}/{name}_test.go`
5. Users register it via `eng.Register({name}.Parser())`

### Adding a new CLI command
1. Create `internal/cli/{name}.go` with `newXxxCmd(parsers, hooks, packs, configPath)`
2. Register it in `newRootCmd()` in `root.go`
3. The command receives parsers, hooks, packs, and the config path — call `pipeline.Build()` / `RunUntil()` (which take `packs` and `pipeline.Options`), or `loadConfig(path, packs)` for config-only commands

### Building a tool on Fuego (pack + programmatic API)
1. Put the format logic in a `core.Pack` returned by `Pack()` — parsers, hooks, an embedded theme FS (with `static/`), and a `ConfigDefaults` YAML fragment
2. Drive the engine with `engine.Build/Serve/Validate` and `engine.BuildOptions` (no temp config file)
3. See the self-hosted docs "Embedding Fuego" and "Format Packs", and `github.com/gofuego/fuego-adr` as the reference implementation

### Adding a new config field
1. Add the field to the appropriate struct in `internal/config/config.go`
2. Add defaults in `applyDefaults()` if needed
3. Add validation in the appropriate validate function if needed
4. Update `internal/config/config_test.go`

### Adding a new integration test fixture
1. Create `testdata/{name}/input/` with config.yaml, content files, and theme
2. Wire any compiled parsers / hooks / packs the fixture needs via `fixtureParserRegistry()`, `fixtureHooks()`, `fixturePacks()`, `fixturePackLayers()` in `integration_test.go`
3. Run `go test -run TestIntegrationFixtures/{name} -update` to generate golden files
4. Inspect golden output for correctness. Fixtures also flow through `incremental_test.go`'s parity sweep, so they must round-trip byte-identically through the build cache.

## External Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/yuin/goldmark` | First-party opt-in Markdown parser (GFM) — `parsers/markdown/` |
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
