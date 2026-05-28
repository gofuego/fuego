# PRD: Fuego Phases 3–10 — From Declarative Parsers to Production-Ready SSG

**Label:** `needs-triage`

---

## Problem Statement

Fuego currently has a working vertical slice (Phases 0–2): a Go binary that reads content files with YAML frontmatter, dispatches to compiled Go parsers, resolves filesystem-mirror URLs, and renders static HTML pages with embedded JSON for client-side hydration. However, Fuego is only usable by developers who write Go parser plugins and manually manage their content directories. It lacks:

- **No-code DSL definition**: Users who don't write Go cannot define custom content types. They must write a Go `Parser` implementation for every DSL.
- **No URL control**: Routes are locked to filesystem mirroring. There is no way to define URL patterns per content type or override routing conventions at scale.
- **No content organization**: There are no taxonomy pages (tag listings, category indexes), no collection pages (sorted/filtered groups), and no site-wide manifest for client-side navigation.
- **No developer experience tooling**: There is no `fuego init` scaffolding, no `fuego validate` for CI, no `fuego list` for debugging routes, and no `fuego serve` with live reload.
- **No static asset handling**: Ignore patterns, `public/` passthrough, and content-colocated binary assets (images, fonts) are not implemented.
- **No site manifest**: Client-side JavaScript has no structured index of pages, taxonomies, or collections to power navigation and search.

Without these capabilities, Fuego cannot fulfill its promise as a meta-engine — it is merely a bespoke Go template runner.

## Solution

Complete Fuego's feature set across eight vertical slices (Phases 3–10), each delivering a testable, runnable increment. When complete, Fuego will be a fully functional meta-engine for static site generation where:

- Users define custom DSLs entirely through YAML configuration (declarative regex parsers) or Go code (compiled parsers).
- URL patterns, taxonomy pages, and collection listings are configured declaratively.
- A single `fuego serve` command provides a complete local development experience with hot reload and reverse proxying to a JS bundler.
- `fuego init` scaffolds a working project that builds and serves out of the box.
- `fuego validate` and `fuego list` enable CI gating and route debugging.
- The build produces a deterministic, self-contained static site with a `site-manifest.json` for client-side navigation.

## User Stories

1. As a content author, I want to define a new DSL (e.g., `.trivia`) using only regex rules in `config.yaml`, so that I can create structured content files without writing any Go code.
2. As a content author, I want capture groups in my declarative parser rules to extract specific parts of a line into node fields, so that I can parse structured formats like `[X] Answer text` into typed data.
3. As a content author, I want the engine to report invalid regex patterns at config load time, so that I catch DSL definition errors immediately instead of at parse time.
4. As a content author, I want compiled Go parsers to take priority over declarative parsers when both are registered for the same type, so that I can override config-based parsing with custom logic when needed.
5. As a site builder, I want to define URL patterns per content type in `config.yaml` (e.g., `trivia: "/quiz/{dir}/{slug}"`), so that my site's URL structure is independent of the filesystem layout.
6. As a site builder, I want frontmatter `slug` fields to override the filename segment of the URL, so that I can rename files without breaking links.
7. As a site builder, I want route patterns to support `{dir}`, `{slug}`, and `{filename}` placeholders, so that I have flexible URL composition from file path components and envelope fields.
8. As a site builder, I want the engine to detect URL collisions across all pages and report them as fatal build errors with both file paths, so that I never deploy a site with duplicate routes.
9. As a site builder, I want to declare ignore patterns in `config.yaml` (e.g., `**/.DS_Store`, `content/**/drafts/*`), so that the engine skips draft files, system files, and scratch content.
10. As a site builder, I want the engine to support `**` (globstar) patterns in ignore rules, so that I can recursively match directories and file patterns.
11. As a site builder, I want files in the `public/` directory to be copied verbatim to the build output root, so that I can serve `favicon.ico`, `robots.txt`, and hosting provider files (`_redirects`) without engine processing.
12. As a site builder, I want binary files (images, PDFs, fonts) colocated with content files to be automatically copied to mirrored output paths, so that relative asset references in content work correctly.
13. As a site builder, I want to declare taxonomy fields in `config.yaml` (e.g., `tags`, `category`), so that the engine automatically generates listing pages for each taxonomy term.
14. As a site builder, I want two-tier taxonomy pages — a term page listing all content with that term (e.g., `/tags/go/`) and an index page listing all terms (e.g., `/tags/`), so that visitors can browse content by classification.
15. As a site builder, I want each taxonomy tier to have its own configurable URL pattern and layout template, so that term pages and index pages can be visually distinct.
16. As a site builder, I want to declare collections in `config.yaml` with glob matching and sort order, so that the engine generates curated listing pages (e.g., "all history trivia sorted by points").
17. As a site builder, I want the engine to generate a `site-manifest.json` containing all page URLs, flattened envelope metadata, and integer-indexed collection/taxonomy membership, so that client-side JavaScript can power navigation, search, and filtering without extra network requests.
18. As a site builder, I want the site manifest to include an optional `summary` field per page, so that client-side fuzzy search has enough text to be useful without indexing full page bodies.
19. As a developer, I want `fuego validate` to run the pipeline through the INDEX phase without rendering, so that I can gate deployments in CI without wasting CPU on HTML generation.
20. As a developer, I want `fuego validate` to exit with code 1 and a structured error report when any fatal error is detected, so that CI pipelines can block broken content merges.
21. As a developer, I want `fuego list` to print a table of all discovered pages with their type, source path, and resolved URL, so that I can debug routing rules without reading build output.
22. As a developer, I want `fuego init <name>` to scaffold a complete working project with `go.mod`, `main.go`, `config.yaml`, a sample `.card` content file, `base.html`, and `app.js`, so that I can start building immediately.
23. As a developer, I want the scaffolded project to build and serve successfully out of the box, so that my first experience with Fuego is seeing a working page, not reading documentation.
24. As a developer, I want `fuego init` to run `go mod tidy` automatically after scaffolding, so that the project is immediately compilable.
25. As a developer, I want `fuego serve` to start a local HTTP server with automatic content re-parsing when I save a file, so that I see changes reflected without manually re-running the build.
26. As a developer, I want `fuego serve` to reverse-proxy asset requests to a Vite/esbuild dev server, so that I have a single browser URL for both content and theme assets during development.
27. As a developer, I want `fuego serve` to spawn the Vite/esbuild subprocess automatically based on `dev.command` in `config.yaml`, so that I only need to run one command to start developing.
28. As a developer, I want `fuego serve` to show an error overlay in the browser when a content file has a parse error, so that I can see what's broken without checking the terminal.
29. As a developer, I want `fuego serve` to debounce rapid file changes (100ms window), so that saving multiple files quickly doesn't trigger redundant rebuilds.
30. As a developer, I want the `prebuild` hook in `config.yaml` to run before every build, so that I can compile theme assets or run preprocessing steps as part of the build pipeline.
31. As a developer, I want the engine to wipe the output directory at the start of every build, so that renamed or deleted content files don't leave ghost pages in the output.
32. As a developer, I want all tests to produce deterministic output (sorted JSON keys, relative paths, fixed clock injection), so that golden-file tests don't flake across machines or CI environments.

## Implementation Decisions

### Module Architecture

The following modules will be built or modified, organized by the pipeline phase they serve:

**1. Declarative Regex Parser Engine** (Phase 3)
- Implements the `core.Parser` interface using ordered regex rules from `config.yaml`.
- Each rule is a compiled `*regexp.Regexp` with an emit template specifying node type, content (with `$0`/`$1`/`$2` capture group substitution), and attributes.
- Rules are evaluated per line, top to bottom, first match wins.
- Regex patterns are compiled and validated at config load time — invalid patterns produce an INIT-phase error.
- When a declarative parser name collides with a compiled parser name, the compiled parser takes priority.
- Lives in `internal/parse/` as a new `DeclarativeParser` struct.

**2. Route Pattern Resolver** (Phase 4)
- Extends the existing route module with pattern-based URL resolution.
- Route patterns are defined per content type in `config.yaml` (e.g., `trivia: "/quiz/{dir}/{slug}"`).
- Three-tier priority: frontmatter `slug` > config route pattern > filesystem mirror.
- `{dir}` expands to the relative subdirectory path, `{slug}` expands to the filename (or frontmatter slug), `{filename}` expands to the raw filename without extension.
- Collision detection remains GlobalFatal with both conflicting file paths in the error message.

**3. Ignore Pattern Matcher** (Phase 5)
- Glob pattern matching applied during the DISCOVER phase.
- Requires a third-party library (`doublestar`) for `**` (globstar) support, since Go's `filepath.Match` does not support recursive glob patterns.
- Patterns are evaluated against relative paths from the content directory.

**4. Static File Copier** (Phase 5)
- Copies the `public/` directory contents verbatim to the output root.
- Copies content-colocated binary assets (images, fonts, etc.) to mirrored paths in the output directory.
- Binary assets are identified during DISCOVER by exclusion — any file in the content directory that is not matched by a registered parser extension is classified as an asset.

**5. Taxonomy & Collection Indexer** (Phase 6)
- Scans all page envelopes for fields declared as taxonomies in config.
- Builds an inverted index: `{term → []PageRef}` for each taxonomy field.
- Generates two tiers of virtual pages per taxonomy: term pages (e.g., `/tags/go/`) and index pages (e.g., `/tags/`).
- Collection builder glob-matches pages by source path, sorts by a configured envelope field, and generates virtual `CollectionPage` structs.
- Virtual pages flow through the same RENDER pipeline as regular content pages.
- Lives in a new `internal/index/` package.

**6. Site Manifest Generator** (Phase 9)
- Produces `site-manifest.json` in the output directory during the RENDER phase.
- Contains a flat array of page objects (URL, type, layout, flattened envelope fields, optional summary).
- Taxonomies and collections reference pages by integer index into the pages array, preserving sort order with zero data duplication.
- JSON output is deterministic: sorted keys, stable page ordering.

**7. CLI Command Suite** (Phase 7)
- `fuego validate`: Runs the pipeline through INDEX, collects errors, prints structured report, exits with appropriate code.
- `fuego list`: Runs through ROUTE, prints a `TYPE | SOURCE | URL` table to stdout.
- `fuego init <name>`: Scaffolds a complete project from `go:embed` templates. Checks for Go toolchain availability. Runs `go mod tidy` automatically. Injects engine version dynamically via `-ldflags`.
- Engine gains a `RunUntil(phase)` method for partial pipeline execution, supporting `validate` and `list` without rendering.

**8. Dev Server** (Phase 8)
- Go HTTP server on configurable port (default 8080).
- Reverse proxy via `net/http/httputil.ReverseProxy` forwarding asset requests (`/assets/*`, `/@vite/*`, WebSocket upgrades) to the Vite dev server port.
- `fsnotify` watcher on `content/` and `theme/` directories with 100ms debounce.
- On content change: single-file incremental re-parse, full re-render of all pages (because taxonomy/collection membership may change).
- Error overlay: when a LocalFatal error occurs, the affected page serves a simple HTML error page showing the error context instead of crashing the server.
- Subprocess management: spawns `dev.command` as a background process, forwards its stdout/stderr, sends SIGTERM on shutdown.
- Lenient error handling mode: LocalFatal errors don't stop the server, only GlobalFatal errors cause shutdown.

### Cross-Cutting Decisions

- **Error handling strategy pattern**: The pipeline emits `EngineError` structs into an accumulator. The CLI handler determines behavior: `build` mode stops on any Fatal, `serve` mode continues on LocalFatal and renders error overlays.
- **Concurrency model**: `errgroup` with `SetLimit(runtime.NumCPU())` for PARSE and RENDER phases. Lock-free pre-allocated slices indexed by file position. Sequential execution for ROUTE, INDEX/COLLECT, and STATIC phases.
- **Deferred lifecycle**: `engine.New()` creates a bare registry. Cobra parses `--config`, `--port` flags before the pipeline runs. Config is loaded inside CLI commands, not at engine construction time.
- **Template rendering contract**: Go renders pre-rendered HTML content + JSON blob. JavaScript performs full replacement hydration (atomic synchronous swap). The JSON schema (`Envelope` + `[]Node`) is the sole contract between Go and JS.

## Testing Decisions

### What Makes a Good Test

Tests should verify **external behavior through the public interface**, not implementation details. For Fuego, this means:

- **Unit tests** call exported functions with controlled inputs and assert on outputs. They do not test internal helper functions directly.
- **Integration tests** use the golden-file pattern: run the full pipeline on a fixture project, compare output files byte-for-byte against committed golden files. The `-update` flag regenerates golden files when behavior intentionally changes.
- **Error case tests** use `expected_error` fixture files to verify that the pipeline fails with the correct error message.
- Tests must be **deterministic**: sorted JSON keys, relative paths, injected fixed clock. No map iteration ordering, no absolute paths, no timestamps.
- All fixtures run with `t.Parallel()` for speed.

### Modules Under Comprehensive Test

**1. Declarative Regex Parser Engine**
- Unit tests: regex compilation, single-rule match, multi-rule ordered evaluation, capture group substitution (`$0`, `$1`), unmatched lines, empty payload, invalid regex detection at config time.
- Integration test: golden-file fixture with a declarative-only `.card` parser defined in config.yaml. Second fixture testing compiled-vs-declarative collision priority.
- Prior art: `internal/parse/parse_test.go` (ParseAll tests with mock parsers).

**2. Route Pattern Resolver**
- Unit tests: `{dir}` expansion, `{slug}` expansion, `{filename}` expansion, nested directories, root-level files, frontmatter slug override with pattern, collision between pattern-resolved URLs, missing pattern fallback to filesystem mirror.
- Integration test: golden-file fixture with config routes producing non-filesystem URLs.
- Prior art: `internal/route/route_test.go` (filesystem mirror, slug override, collision detection).

**3. Taxonomy & Collection Indexer**
- Unit tests: inverted index construction from `tags: [go, web]` across multiple pages, term normalization, two-tier page generation, collection glob matching, sort by string/numeric envelope fields, empty taxonomy field handling, virtual page URL assignment.
- Integration test: golden-file fixture with taxonomy config producing term + index pages, collection config producing sorted listing pages.
- Prior art: `internal/render/render_test.go` (RenderAll with multiple pages).

**4. Dev Server (Watcher + Proxy)**
- Unit tests: reverse proxy request routing (asset paths → Vite, content paths → Go), watcher debounce timing, subprocess start/stop lifecycle, error overlay HTML generation.
- Integration test: start server, programmatically modify a content file, verify output is updated (with short timeout). Start server with a broken content file, verify error overlay is served.
- Prior art: `internal/render/render_test.go` (template rendering with temp dirs).

### Additional Test Fixtures (Golden-File)

- `testdata/declarative-parser/` — Phase 3: config-defined parser rules
- `testdata/route-patterns/` — Phase 4: config route patterns + slug override
- `testdata/ignore-and-static/` — Phase 5: ignore globs + public/ passthrough + colocated assets
- `testdata/taxonomies/` — Phase 6: taxonomy term + index pages
- `testdata/collections/` — Phase 6: collection listing pages
- `testdata/url-collision/` — Phase 4: error case with `expected_error` file
- `testdata/comprehensive/` — Phase 10: all features exercised together

## Out of Scope

- **Markdown rendering**: Fuego does not convert Markdown to HTML. Raw passthrough wraps payload in a `Node{Type: "raw"}`. Markdown-to-HTML conversion is the responsibility of a compiled Go parser or client-side JavaScript.
- **CSS/JS minification or bundling**: The engine does not process theme assets. Vite/esbuild handles this via the `prebuild` hook or dev server proxy.
- **Asset cache busting**: Content hashing of static assets (e.g., `main.abcdef12.css`) is deferred to v2. Phase 9 copies assets as-is.
- **Incremental builds**: Every build wipes the output directory and rebuilds from scratch. Dependency-graph-based incremental builds are a v2 optimization.
- **Custom CLI subcommands**: Users cannot extend the CLI with their own commands in v1. `engine.Run()` owns the Cobra tree entirely.
- **Go plugin system (`plugin.Open`)**: Rejected during architecture design due to cross-platform fragility. All Go parsers are compiled-in via `engine.Register()`.
- **Server-side rendering of interactive components**: The Go engine renders semantic HTML for crawlers/FCP. Interactive rendering is entirely JavaScript's responsibility via full-replacement hydration.
- **Multi-language / i18n support**: Not addressed in v1.

## Further Notes

- **Phase ordering matters**: Phases are vertical slices with dependency ordering. Phase 3 (declarative parsers) depends on the config loader. Phase 4 (routing patterns) must exist before Phase 6 (taxonomies/collections) because virtual pages need URLs. Phase 7 (CLI suite) depends on all pipeline phases. Phase 8 (dev server) depends on the complete build pipeline.
- **The `core/` package exists to break import cycles**: Both `engine/` (public API) and `internal/*` packages import from `core/`. The `engine/` package re-exports core types as type aliases so that external users only import `engine`.
- **`PageData` is the spine of the pipeline**: It is progressively enriched by each phase. DISCOVER sets paths, PARSE sets envelope/nodes, ROUTE sets URL, INDEX may annotate with taxonomy membership. It is a mutable pointer struct.
- **Determinism is a non-negotiable invariant**: From Phase 1 onward, output must be deterministic for golden-file tests. This means: sorted JSON keys (Go's `encoding/json` sorts map keys by default), relative paths in all error messages, and an injectable clock for any timestamp-dependent features.
- **The `</script>` injection vulnerability** is already mitigated: `json.Marshal` escapes `<`, `>`, `&` to unicode sequences, and `html/template` provides contextual escaping. This invariant must be maintained in all future rendering code.
