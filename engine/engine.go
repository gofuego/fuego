package engine

import (
	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/cli"
)

// Engine is the core orchestrator for the Fuego static site generator.
type Engine struct {
	parsers map[string]core.Parser
	hooks   core.Hooks
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

// AfterParse registers a hook that runs after PARSE, before ROUTE.
// Multiple hooks run in FIFO order; each receives the previous hook's output.
func (e *Engine) AfterParse(fn core.AfterParseHook) {
	e.hooks.AfterParse = append(e.hooks.AfterParse, fn)
}

// Index registers a hook that runs during INDEX, after taxonomy and
// collection generation but before the collision re-check. Use it to add
// virtual pages: set their URL and they are collision-checked like
// engine-generated virtual pages. Multiple hooks run in FIFO order.
func (e *Engine) Index(fn core.IndexHook) {
	e.hooks.Index = append(e.hooks.Index, fn)
}

// BeforeRender registers a hook that runs after INDEX, before RENDER.
// Multiple hooks run in FIFO order; each receives the previous hook's output.
func (e *Engine) BeforeRender(fn core.BeforeRenderHook) {
	e.hooks.BeforeRender = append(e.hooks.BeforeRender, fn)
}

// Parsers returns the registered parser map.
func (e *Engine) Parsers() map[string]core.Parser {
	return e.parsers
}

// Run hands control to the Cobra CLI.
func (e *Engine) Run(args []string) error {
	return cli.Execute(args, e.parsers, &e.hooks)
}
