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

**Wave 4:** ← NEXT
- 09 deep merge + `fuego config` (after 08) · 18 scaffold refresh (after 01–04)

**Wave 5:**
- 17 docs dogfood (after 01–04, 09)

→ **tag v0.3.0**

**Wave 6 (0.3.1):**
- 13 incremental core (after 09) · 15 port fuego-adr, 16 port fuego-devops (after 09, 10 — HITL, separate repos)

**Wave 7 (0.3.1):**
- 14 render narrowing (after 13) · 11 `init --pack` (after 09, 15)

→ **tag v0.3.1**

## Milestone summary

| Milestone | Issues |
|-----------|--------|
| 0.3.0 | 01, 02, 03, 04, 05, 06, 07, 08, 09, 10, 12, 17, 18 |
| 0.3.1 | 11, 13, 14, 15, 16 |

HITL issues: 15, 16 (product decisions in satellite repos). Everything else is AFK.

v1.0 (API freeze + compat promise) is intentionally NOT in this set — it follows after the
satellite ports' friction logs are absorbed and external feedback lands.
