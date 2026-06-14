---
title: Golden-file integration testing with enforced determinism
status: accepted
date_proposed: 2026-05-20
date_accepted: 2026-05-20
author: fabio
approvers: [fabio]
tags: [testing, determinism, quality]
affects:
  - testdata/**
  - internal/**
---

## Context

An engine that turns arbitrary input into a tree of files needs regression
protection that's easy to read and hard to fool. Assertion-based tests over
rendered HTML are brittle and verbose; they also don't catch nondeterministic
output, which would silently break incremental builds and caching.

## Decision

Integration tests use a **golden-file** pattern: each fixture has `input/` and
`golden/` directories, and the build is compared **byte-for-byte** against
`golden/` (regenerate with `go test -run TestIntegrationFixtures -update`).
Determinism is a hard requirement, enforced by running tests with
`go test -count=3` and by routing every map iteration through `sortedKeys()`
helpers. Fixtures run in parallel with `t.Parallel()`.

## Consequences

- **+** Golden files show exactly what the pipeline produces and double as
  documentation; any drift fails the byte comparison.
- **+** Enforced determinism is the precondition for incremental builds
  ([012](012-opt-in-incremental-builds.adr.md)).
- **−** Intentional output changes require regenerating goldens, which must be
  reviewed; a careless `-update` can bless a regression.
