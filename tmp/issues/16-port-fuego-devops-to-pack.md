---
title: "Port fuego-devops to a pack + Index hook (closes collision bypass)"
type: HITL
milestone: 0.3.1
labels: [needs-triage]
blocked_by: [09, 10]
---

## What to build

Port fuego-devops (separate repo) onto the v0.3 surface: Dockerfile + Kubernetes parsers, dark theme, and graph builder become a `core.Pack`; the graph builder moves from `BeforeRender` to the new `Index` hook so the virtual diagram page goes through URL resolution and collision re-checking like every other virtual page (the bypass it currently exploits is the motivating case for #10).

HITL: product decisions in the fuego-devops repo (scanner stays a separate stage or joins Init, pack config surface).

## Acceptance criteria

- [ ] `devops.Pack()` bundles both parsers + theme FS + the graph Index hook; no fuego core changes needed
- [ ] Diagram virtual page is created in the Index hook and is collision-checked (regression test: a content page at the diagram URL fails the build with both claimants named)
- [ ] Output site equivalent to pre-port build (diagram data, per-resource pages) verified against a sample infra repo
- [ ] Scanner decision documented (stays standalone vs. Init-triggered) with rationale
- [ ] Friction log filed: pack API + Index hook pain points become issues before v0.3.1 tags

## Blocked by

- 09-config-deep-merge-provenance
- 10-index-hook-page-skip
