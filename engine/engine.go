package engine

import (
	"fmt"
	"os"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/cli"
)

// Engine is the core orchestrator for the Fuego static site generator.
type Engine struct {
	parsers         map[string]core.Parser
	userParsers     map[string]bool   // parser types registered directly via Register
	packParserOwner map[string]string // parser type -> pack name, for override warnings
	hooks           core.Hooks
	packs           []core.Pack
}

// New creates a new Engine instance with an empty parser registry.
func New() *Engine {
	return &Engine{
		parsers:         make(map[string]core.Parser),
		userParsers:     make(map[string]bool),
		packParserOwner: make(map[string]string),
	}
}

// Register adds a compiled Go parser to the engine's registry.
// User-registered parsers always win over pack parsers, regardless of
// registration order.
func (e *Engine) Register(p core.Parser) {
	name := p.Type()
	e.parsers[name] = p
	e.userParsers[name] = true
	delete(e.packParserOwner, name)
}

// Use registers a format pack: its parsers, hooks, and theme templates.
// Among packs, later registration wins on conflicts (with a warning);
// user-registered parsers and user theme files always win over packs.
func (e *Engine) Use(p core.Pack) {
	e.packs = append(e.packs, p)

	for _, parser := range p.Parsers {
		name := parser.Type()
		if e.userParsers[name] {
			continue // user compiled parser wins silently — that's the override gesture
		}
		if owner, ok := e.packParserOwner[name]; ok {
			fmt.Fprintf(os.Stderr, "fuego: warning: parser %q from pack %q overridden by pack %q\n",
				name, owner, p.Name)
		}
		e.parsers[name] = parser
		e.packParserOwner[name] = p.Name
	}

	e.hooks.AfterParse = append(e.hooks.AfterParse, p.Hooks.AfterParse...)
	e.hooks.Index = append(e.hooks.Index, p.Hooks.Index...)
	e.hooks.BeforeRender = append(e.hooks.BeforeRender, p.Hooks.BeforeRender...)
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
	return cli.Execute(args, e.parsers, &e.hooks, e.packs)
}
