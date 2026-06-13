package core

import (
	"context"
	"io/fs"
)

// Pack bundles a content format's parsers, hooks, and theme templates into
// one registerable unit — the ecosystem building block for domain-specific
// site generators. Register with engine.Use().
//
// Precedence rules:
//   - Parsers: user-registered compiled parsers always win over pack
//     parsers; among packs, later registration wins (with a warning).
//     Declarative config parsers stay lowest.
//   - Theme templates: the user's theme directory always wins over pack
//     themes; among packs, later registration wins (with a warning).
type Pack struct {
	// Name identifies the pack in warnings and config namespacing
	// (the `packs.{Name}:` subtree of config.yaml).
	Name string

	// Parsers are compiled parsers the pack provides.
	Parsers []Parser

	// Hooks are appended to the engine's hooks in registration order.
	Hooks Hooks

	// Theme holds the pack's templates: base.html (optional), layouts/,
	// renderers/, and partials/ at the FS root. Typically an embed.FS
	// rooted at the pack's theme directory.
	Theme fs.FS

	// ConfigDefaults is an optional YAML fragment the pack contributes to
	// the site config (routes, taxonomies, collections, declarative
	// parsers). It is deep-merged under the user's config.yaml: maps merge
	// key-wise, scalars and lists are replaced whole, and the user always
	// wins. Among packs, later registration wins.
	ConfigDefaults []byte

	// Init, if set, runs once during the INIT phase. It receives the pack's
	// config subtree and may register additional parsers and hooks based on
	// it. Returning an error halts the build. Init is the single pack
	// lifecycle point — there is no Shutdown.
	Init func(ctx context.Context, pc *PackContext) error
}

// PackContext is handed to a pack's Init function. It exposes the pack's
// config subtree and lets the pack register parsers and hooks conditionally.
// Parsers and hooks registered here merge with those declared on the Pack
// struct, under the same precedence rules.
type PackContext struct {
	name   string
	config map[string]any

	parsers      []Parser
	afterParse   []AfterParseHook
	index        []IndexHook
	beforeRender []BeforeRenderHook
}

// NewPackContext builds a PackContext for the named pack with the given
// config subtree (nil when the pack has no config). Intended for the
// pipeline; packs receive a ready-made context.
func NewPackContext(name string, config map[string]any) *PackContext {
	return &PackContext{name: name, config: config}
}

// Name returns the pack's name.
func (pc *PackContext) Name() string { return pc.name }

// Config returns the pack's raw config subtree (the `packs.{name}:` map),
// or nil if the pack has no config. Packs validate this themselves in Go.
func (pc *PackContext) Config() map[string]any { return pc.config }

// Register adds a parser discovered during Init.
func (pc *PackContext) Register(p Parser) { pc.parsers = append(pc.parsers, p) }

// AfterParse registers an after-parse hook during Init.
func (pc *PackContext) AfterParse(fn AfterParseHook) { pc.afterParse = append(pc.afterParse, fn) }

// Index registers an index hook during Init.
func (pc *PackContext) Index(fn IndexHook) { pc.index = append(pc.index, fn) }

// BeforeRender registers a before-render hook during Init.
func (pc *PackContext) BeforeRender(fn BeforeRenderHook) {
	pc.beforeRender = append(pc.beforeRender, fn)
}

// Registered returns the parsers and hooks accumulated during Init.
func (pc *PackContext) Registered() ([]Parser, Hooks) {
	return pc.parsers, Hooks{
		AfterParse:   pc.afterParse,
		Index:        pc.index,
		BeforeRender: pc.beforeRender,
	}
}
