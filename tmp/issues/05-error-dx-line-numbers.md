---
title: "Error DX: line numbers on errors, template errors name layout + page"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: []
---

## What to build

Add `Line int` to `core.EngineError` and thread positions through the places that already know them: the declarative parser fills line numbers as it iterates rules over lines; `core.SplitFrontmatter` maps YAML errors to file line offsets (frontmatter starts at line 2); template execution errors are wrapped with the layout name and page path. `core.Node` stays position-free. Compiled parsers can optionally return a `core.ParseError{Line, Err}` that the dispatcher unwraps into `EngineError.Line` — nothing forces them to.

## Acceptance criteria

- [ ] `core.EngineError` has `Line`; error formatting prints `file:line` when Line > 0
- [ ] Declarative parser reports the line of the first unmatched/offending input line in LocalFatal errors
- [ ] `SplitFrontmatter` YAML errors carry the correct file-relative line (unit tests with multiline frontmatter)
- [ ] Template errors read like: `RENDER theme/layouts/post.html (content/foo.md): <cause>`
- [ ] `core.ParseError` exists, documented in the custom-parsers docs page; dispatcher unwraps it
- [ ] No change to `core.Node`; no new dependencies in `core/`

## Blocked by

None - can start immediately

## Implementation note (2026-06-13)

`EngineError.Line` already existed with `file:line` formatting. One criterion adjusted during
implementation: the declarative parser **silently skips unmatched lines by design** (prose lines
in a `.trivia` file are not errors), so it has no "first unmatched line" failure to position.
Its error positions come from frontmatter: `SplitFrontmatter` now returns `core.ParseError`
with file-relative lines (YAML errors offset past the opening `---`, unclosed-frontmatter
errors point at the delimiter), and the dispatcher unwraps `ParseError` from any parser.
Also fixed while in there: docs/custom-parsers.md still documented the pre-decoupling
`Parse(rawPayload, meta)` signature and "built-in Markdown" — rewritten to match AD-2/AD-3.
