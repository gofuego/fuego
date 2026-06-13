---
title: "Pack config + lifecycle: packs.{name} subtrees and Init(ctx, PackContext)"
type: AFK
milestone: 0.3.0
labels: [needs-triage]
blocked_by: [07]
---

## What to build

config.yaml gains a `packs:` section with one subtree per pack name. During INIT, each registered pack's `Init(ctx, PackContext) error` runs (single lifecycle point — no Shutdown/PostBuild). `PackContext` provides the pack's raw config subtree (`map[string]any`) for the pack to validate itself in Go — no schema language — plus the ability to register parsers/hooks programmatically based on config. An Init error is GlobalFatal with the pack name in the message.

## Acceptance criteria

- [ ] `packs.{name}:` subtrees parsed and routed to the matching registered pack; unknown subtree names produce a Warning naming known packs
- [ ] `Init` is optional (nil-safe); called during INIT in registration order, before parser merge completes
- [ ] `PackContext` exposes the config subtree and registration methods; documented with a runnable example pack
- [ ] Init returning an error halts the build (GlobalFatal) with `pack {name}: <cause>`
- [ ] Integration fixture: pack whose Init reads config to conditionally register a parser; golden output for both config states
- [ ] Docs: "Build a format pack" tutorial covers Init + config validation

## Blocked by

- 07-core-pack-registration
