---
title: "core.Pack + eng.Use(): pack-registered parsers, hooks, and theme FS"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: [01]
---

## What to build

Introduce the pack unit: `core.Pack` (stdlib-only — keeps the core/ zero-dependency rule) carrying `Name`, `Parsers []Parser`, `Hooks Hooks`, and `Theme fs.FS` (containing `renderers/`, `layouts/`, `partials/`). Register via `eng.Use(pack)`. Template resolution precedence: user theme dir always beats packs; among packs, later registration wins with a logged warning on conflict. Parser precedence slots packs between user-compiled parsers and declarative parsers (user compiled > pack > declarative).

End-to-end: a test pack bundling a parser + renderer + layout produces a working site with zero files in the user's `theme/`.

## Acceptance criteria

- [ ] `core.Pack` defined in core/ with no non-stdlib imports; `eng.Use()` on the engine
- [ ] `LoadTemplates` layers pack theme FSes under the user theme dir with the agreed precedence; conflicts between packs log a warning naming both packs
- [ ] Pack parsers merge into dispatch with precedence: user compiled > pack compiled > declarative
- [ ] Pack hooks (AfterParse/BeforeRender) run in registration order alongside user hooks
- [ ] Integration fixture: site rendered entirely from a fixture pack's theme; second fixture proving user `theme/renderers/X.html` overrides the pack's
- [ ] Docs: concepts page "Format packs" (registration + precedence rules)

## Blocked by

- 01-template-partials-and-funcs (template cache restructuring lands there first)
