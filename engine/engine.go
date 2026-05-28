package engine

import (
	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/cli"
)

// Engine is the core orchestrator for the Fuego static site generator.
type Engine struct {
	parsers map[string]core.Parser
}

// New creates a new Engine instance with an empty parser registry.
func New() *Engine {
	return &Engine{
		parsers: make(map[string]core.Parser),
	}
}

// Register adds a compiled Go parser to the engine's registry.
func (e *Engine) Register(p core.Parser) {
	e.parsers[p.Type()] = p
}

// Parsers returns the registered parser map.
func (e *Engine) Parsers() map[string]core.Parser {
	return e.parsers
}

// Run hands control to the Cobra CLI.
func (e *Engine) Run(args []string) error {
	return cli.Execute(args, e.parsers)
}
