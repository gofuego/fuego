# Fuego v0.3 release — issue set

18 tracer-bullet slices for the framework release agreed on 2026-06-12. Each file is one
issue (frontmatter: type, milestone, blocked_by by file number; all start as needs-triage).
Partial scope may ship as **0.3.0** with the remainder in **0.3.1** — milestones reflect that split.

## Dependency-ordered work plan

**Wave 1 — no blockers (parallelizable):** ✅ DONE 2026-06-13
- 01 template partials + funcs ✅ · 05 error DX ✅ · 06 .JSON detection ✅ · 10 Index hook + Skip ✅ · 12 benchmark baseline ✅
- Bonus fixes found during the wave: trailing-slash URL-collision bug in DetectCollisions;
  stale custom-parsers.md (pre-decoupling signature, "built-in Markdown"); docs/main.go never
  registered the Markdown parser after the zero-built-ins refactor (docs site built 0 pages).

**Wave 2:** ✅ DONE 2026-06-13
- 02 .Site.Pages ✅ · 07 core.Pack registration ✅
- Fixtures: site-pages (nav via where/sortBy, taxonomy terms in refs), pack-theme (site rendered
  entirely from pack FS), pack-theme-override (user renderer beats pack). Engine precedence
  unit-tested incl. Register-after-Use.

**Wave 3:** ✅ DONE 2026-06-13
- 03 pagination ✅ · 04 outputs ✅ · 08 pack config + Init ✅
- Fixtures: pagination (collection + taxonomy term split, prev/next), outputs (sitemap.xml +
  feed.xml via theme/outputs/ as text/template), outputs-collision (output vs page GlobalFatal),
  pack-init / pack-init-disabled (Init conditionally registers parser by config — two golden
  states), pack-init-error (Init failure halts build naming pack). core.PackContext unit-tested.
- Stale-doc fixes: build-pipeline.md (built-in Markdown claim, added OUTPUTS phase + Index hook +
  pagination + Skip).

**Wave 4:** ✅ DONE 2026-06-13
- 09 deep merge + `fuego config` ✅ · 18 scaffold refresh ✅
- 09: core.Pack.ConfigDefaults []byte (YAML), config.LoadLayered + mergeLayers (maps merge,
  scalars+lists replace, user>later pack>earlier), Provenance, `fuego config` with per-key
  comments. Fixture pack-config-defaults (user overrides pack route, inherits pack taxonomy).
  Provenance-aware validation names the pack. Table-driven merge tests incl. map-in-list/list-in-map.
- 18: scaffold refreshed — registers Markdown (was BROKEN post-refactor: index.md never rendered),
  nav partial off .Site.Pages, sitemap+rss outputs, paginated cards collection, front/back/page-ref
  renderers, rewritten CLAUDE.md (packs/Index hook/funcs/error DX), README quick-start updated.
  Refactored scaffold.Generate→WriteFiles+resolveDeps; scaffold_test builds the generated site offline.

**Wave 5:** ✅ DONE 2026-06-13
- 17 docs dogfood ✅
- Docs site theme converted: topbar+sidebar partials (sidebar driven entirely by .Site.Pages,
  sorted by URL, current-page highlight), sitemap.xml + rss.xml outputs, tutorials collection
  paginated (page_size:1). Docs site consumes docs/docspack (in-repo example pack contributing
  the tags taxonomy + tutorials collection as ConfigDefaults — config.yaml slimmed to site only;
  `fuego config` shows # pack: docs provenance live). New "Build a Format Pack" tutorial (order:3).
  CLI ref gains `fuego config`. Stale "built-in Markdown" purged from format-agnostic.md.
  Docs build: 38 pages, clean.

→ **0.3.0 COMPLETE — ready to tag v0.3.0** (all of 01-10, 12, 17, 18 done & committed)

**Wave 6 (0.3.1):**
- 13 incremental core ✅ DONE 2026-06-13 · 15 port fuego-adr, 16 port fuego-devops (after 09, 10 — HITL, separate repos)
  - 13: internal/buildcache (gob cache: header = binaryID+configHash+themeHash, per-file content hash;
    orphan removal; version-mismatch → miss). parse.ParseAllCached reuses unchanged. pipeline.Options
    {Incremental,CacheDir}; full rebuild cleans, incremental updates in place. CLI `build --incremental`;
    serve implicit. byte-equivalence test (controlled site × {noop,edit,add,delete,theme,config}) +
    fixture-parity sweep (23 fixtures, warm-reuse == clean) + parse-counter test. Smoke: docs 0/19→1/18.

**Wave 7 (0.3.1):**
- 14 render narrowing ✅ DONE 2026-06-13 · 11 `init --pack` (after 09, 15)
  - 14: incremental RENDER narrowed to affected set = changed pages ∪ virtual pages (SourcePath=="")
    ∪ pages whose template reads .Site.Pages (precise — .Site.Name is build-constant). render/sitedetect.go
    detects .Site.Pages incl. transitive partials (fixpoint). CacheStats.Changed threaded to RenderAll.
    Bench: 10k single-file edit = 324ms vs 3852ms full (~12x). Equivalence/parity suites pass with
    narrowing on (now exercise skipping). TestIncrementalNarrowsRendering proves skip via mtime.

→ **tag v0.3.1**

## Milestone summary

| Milestone | Issues |
|-----------|--------|
| 0.3.0 | 01, 02, 03, 04, 05, 06, 07, 08, 09, 10, 12, 17, 18 |
| 0.3.1 | 11, 13, 14, 15, 16 |

HITL issues: 15, 16 (product decisions in satellite repos). Everything else is AFK.

v1.0 (API freeze + compat promise) is intentionally NOT in this set — it follows after the
satellite ports' friction logs are absorbed and external feedback lands.
