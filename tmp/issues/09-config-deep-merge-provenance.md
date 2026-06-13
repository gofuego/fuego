---
title: "Pack config defaults: deep merge + `fuego config` provenance command"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: [08]
---

## What to build

Packs may contribute config defaults (routes, taxonomies, collections, declarative parser rules). Merge into effective config with precedence **user > later pack > earlier pack** and shape rules: maps merge recursively key-wise, scalars replace, **lists replace whole**. Merging happens at INIT after pack registration, before config validation (validation runs on the merged result). Ship a `fuego config` CLI command printing the fully resolved effective config as YAML with per-key provenance comments (`# from pack: adr`, `# user`), so "why is this value X" is one command away.

## Acceptance criteria

- [ ] Merge implements the three shape rules with table-driven unit tests, including nested map-in-list and list-in-map cases
- [ ] Precedence verified: user value beats any pack; later-registered pack beats earlier
- [ ] Config validation runs post-merge; invalid pack-contributed entries fail with the contributing pack named
- [ ] `fuego config` prints resolved YAML with provenance per top-level and nested key; deterministic output
- [ ] Integration fixture: pack contributes a taxonomy + route; user config overrides the route; golden output proves both directions
- [ ] Docs: reference page "Config merging" with the precedence table and list-replacement rule called out prominently

## Blocked by

- 08-pack-config-lifecycle
