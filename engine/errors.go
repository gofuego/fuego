package engine

import "github.com/FabioSol/fuego/core"

// Re-export core error types as the public API surface.
type (
	EngineError      = core.EngineError
	ErrorAccumulator = core.ErrorAccumulator
	Severity         = core.Severity
)

const (
	Warning    = core.Warning
	LocalFatal = core.LocalFatal
	GlobalFatal = core.GlobalFatal
)
