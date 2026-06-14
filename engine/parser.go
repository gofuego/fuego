package engine

import "github.com/gofuego/fuego/core"

// Re-export core types as the public API surface.
type (
	Node     = core.Node
	Envelope = core.Envelope
	Parser   = core.Parser
)
