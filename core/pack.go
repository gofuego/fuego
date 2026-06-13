package core

import "io/fs"

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
	// Name identifies the pack in warnings and (later) config namespacing.
	Name string

	// Parsers are compiled parsers the pack provides.
	Parsers []Parser

	// Hooks are appended to the engine's hooks in registration order.
	Hooks Hooks

	// Theme holds the pack's templates: base.html (optional), layouts/,
	// renderers/, and partials/ at the FS root. Typically an embed.FS
	// rooted at the pack's theme directory.
	Theme fs.FS
}
