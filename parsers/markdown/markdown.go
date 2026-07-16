// Package markdown provides a Fuego parser for Markdown files (.md)
// with YAML frontmatter and GFM (GitHub Flavored Markdown) extensions.
//
// It is the engine's co-versioned default format: it lives in the engine repo
// so the most common case needs no second module, but it follows the
// fuego-formats conventions from the outside — a claims-options constructor,
// exported node-type constants, and a schema.md contract (see schema.md).
//
// Usage:
//
//	eng := engine.New()
//	eng.Register(markdown.Parser())
//
// Override the default bare-extension claim for a brownfield repo:
//
//	eng.Register(markdown.Parser(markdown.WithPatterns("*.markdown")))
//	eng.Register(markdown.Parser(markdown.WithPatterns("README.md")))
//
// Patterns replace the claim entirely: with WithPatterns("README.md") the
// parser claims only README.md, not every .md file.
package markdown

import (
	"bytes"

	"github.com/gofuego/fuego/core"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// Type is the parser's Type() — the page type and, absent a pattern override,
// the bare-extension claim.
const Type = "md"

// NodeHTML is the single node type emitted per file: a raw node carrying the
// rendered HTML. The unprefixed value predates the fuego-formats slug-prefix
// convention and is kept for compatibility with existing themes
// (theme/renderers/html.html).
const NodeHTML = "html"

// Option configures the parser's claims. Options apply in order; the last
// WithPatterns wins.
type Option func(*config)

type config struct {
	patterns []string
}

// WithPatterns overrides the claim patterns entirely — the escape hatch for a
// repo whose markdown files don't match the default bare-md claim (e.g.
// "*.markdown", or claiming only "README.md"). The patterns become the
// parser's complete claim set: the default md extension claim is dropped.
// Claims match base names only — no path scoping, no content sniffing.
func WithPatterns(patterns ...string) Option {
	return func(c *config) { c.patterns = append([]string(nil), patterns...) }
}

// Parser returns a Fuego parser for .md files.
// It splits YAML frontmatter, then converts the Markdown payload to HTML
// using goldmark with GFM extensions (tables, strikethrough, autolinks).
// Pass WithPatterns(...) to replace the default bare-extension claim.
func Parser(opts ...Option) core.Parser {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)

	p := core.WithYAMLFrontmatter(Type, func(payload []byte, meta core.Envelope) ([]core.Node, error) {
		var buf bytes.Buffer
		if err := md.Convert(payload, &buf); err != nil {
			return nil, err
		}

		html := buf.String()
		if html == "" {
			return nil, nil
		}

		return []core.Node{
			{Type: NodeHTML, Content: html, Raw: true},
		}, nil
	})

	if len(cfg.patterns) == 0 {
		return p
	}
	return &patternParser{Parser: p, patterns: cfg.patterns}
}

// patternParser wraps the markdown parser with an explicit filename-pattern
// claim set, making it a core.FilenameParser so the dispatch resolver claims
// exactly the given patterns instead of the bare md extension.
type patternParser struct {
	core.Parser
	patterns []string
}

// Filenames reports the overridden claim patterns. It returns a copy so a
// caller cannot mutate the parser's claims.
func (p *patternParser) Filenames() []string {
	return append([]string(nil), p.patterns...)
}
