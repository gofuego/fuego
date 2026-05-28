package parse

import (
	"bytes"

	"github.com/FabioSol/fuego/core"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// MarkdownParser is the built-in parser for .md files.
// It converts Markdown (with GFM extensions) to HTML and wraps
// the result in a single Node{Type: "html"}.
type MarkdownParser struct {
	md goldmark.Markdown
}

// NewMarkdownParser creates a Markdown parser with GFM extensions enabled.
func NewMarkdownParser() *MarkdownParser {
	return &MarkdownParser{
		md: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
		),
	}
}

func (p *MarkdownParser) Type() string { return "md" }

func (p *MarkdownParser) Parse(rawPayload []byte, meta core.Envelope) ([]core.Node, error) {
	var buf bytes.Buffer
	if err := p.md.Convert(rawPayload, &buf); err != nil {
		return nil, err
	}

	html := buf.String()
	if html == "" {
		return nil, nil
	}

	return []core.Node{
		{Type: "html", Content: html},
	}, nil
}

// BuiltinParsers returns the set of parsers that ship with the engine.
// These are lowest priority — overridden by both declarative and compiled parsers.
func BuiltinParsers() map[string]core.Parser {
	return map[string]core.Parser{
		"md": NewMarkdownParser(),
	}
}
