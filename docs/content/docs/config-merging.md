---
title: Config Merging
layout: doc
nav_section: "Reference"
nav_weight: 6
tags:
  - reference
  - config
  - packs
---

When you register [format packs](/docs/concepts/format-packs/), each pack may contribute config defaults — routes, taxonomies, collections, declarative parsers. Fuego deep-merges those defaults beneath your `config.yaml` so installing a pack "just works", while your config always has the final say.

## Precedence

From lowest to highest:

1. Earlier-registered pack defaults
2. Later-registered pack defaults
3. **Your `config.yaml`** — always wins

## Merge rules

The merge operates key by key on the YAML structure:

| Value kind | Behavior |
|---|---|
| **Maps** | Merged recursively, key by key |
| **Scalars** | Replaced whole by the higher-precedence layer |
| **Lists** | Replaced whole — never appended or element-merged |

The list rule is deliberate and worth internalizing: if a pack ships a list (say, a parser's `rules:`) and you define the same list, **your list replaces it entirely** — the two are not concatenated. This keeps merges predictable and lets you remove pack entries, at the cost of having to restate a list you want to extend.

### Example

A pack contributes:

```yaml
routes:
  adr: /adr/{slug}
taxonomies:
  status:
    path: /status/{term}
```

Your `config.yaml`:

```yaml
routes:
  adr: /decisions/{slug}
```

Resolved result: `routes.adr` is `/decisions/{slug}` (yours), and `taxonomies.status` is inherited from the pack. `routes` is a map, so the two `routes` blocks merge; `adr` is a scalar, so yours replaces the pack's.

## Inspecting the result

Deep merges can make "why is this value what it is?" hard to answer. The `fuego config` command prints the fully resolved configuration with a provenance comment on every key:

```console
$ fuego config
routes:
  adr: /decisions/{slug}   # user
taxonomies:
  status:                  # pack: adr
    path: /status/{term}   # pack: adr
```

Output is deterministic (keys sorted), so it is safe to diff across changes.

## Validation

Config validation runs **after** the merge, on the resolved result. If a pack contributes an invalid entry — a malformed parser regex, a negative `page_size` — the error names the contributing pack:

```
parser "broken" rule 0: invalid regex "[" (from pack "adr"): ...
```
