package engine

import "github.com/FabioSol/fuego/core"

// Re-export core types as the public API surface.
type (
	Node     = core.Node
	Envelope = core.Envelope
	Parser   = core.Parser
)
