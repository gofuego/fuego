---
title: "Incremental builds (opt-in): cache core + byte-equivalence CI suite"
type: AFK
milestone: 0.3.1
labels: [needs-triage]
blocked_by: [09]
---

## What to build

`fuego build --incremental` (serve mode uses it implicitly). On-disk cache keyed by: **binary build ID** (the only honest proxy for compiled parsers/hooks/pack code) + resolved post-merge config hash + theme tree hash + per-content-file hashes. Any binary/config/theme change ⇒ full rebuild; content-only edits re-parse changed files, then re-run ROUTE/INDEX fully (cheap O(pages) map work) and re-render **all** pages (conservative — narrowing is #14). Deleted content removes its output via manifest diff (orphan removal). Default `fuego build` stays clean and cache-free.

The correctness contract ships WITH the feature: a CI suite that builds every integration fixture clean, then incrementally under mutations (edit/add/delete content, touch theme, touch config), asserting **byte-identical** output trees each time.

## Acceptance criteria

- [ ] Cache key includes binary build ID (via `debug.ReadBuildInfo`/executable hash), resolved config hash, theme tree hash, content file hashes
- [ ] Content edit: only changed files re-parsed (verified via counters/test hooks); ROUTE/INDEX re-run fully; output byte-identical to clean build
- [ ] Content delete: orphaned output files and emptied dirs removed via manifest diff
- [ ] Theme/config/binary change: silent full rebuild
- [ ] Equivalence CI suite covers all fixtures × {edit, add, delete, theme-touch, config-touch}; failure diff names the differing file
- [ ] Cache corruption/version mismatch falls back to full rebuild with a Warning, never an error
- [ ] Docs: CLI reference for `--incremental` + a concepts note on the cache key and guarantees

## Blocked by

- 09-config-deep-merge-provenance (cache key hashes the *resolved* config)
