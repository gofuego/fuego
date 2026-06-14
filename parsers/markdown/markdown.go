// Package markdown provides a Fuego parser for Markdown files (.md)
// with YAML frontmatter and GFM (GitHub Flavored Markdown) extensions.
//
// Usage:
//
//	eng := engine.New()
//	eng.Register(markdown.Parser())
package markdown

import (
	"bytes"

	"github.com/gofuego/fuego/core"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// Parser returns a Fuego parser for .md files.
// It splits YAML frontmatter, then converts the Markdown payload to HTML
// using goldmark with GFM extensions (tables, strikethrough, autolinks).
func Parser() core.Parser {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)

	return core.WithYAMLFrontmatter("md", func(payload []byte, meta core.Envelope) ([]core.Node, error) {
		var buf bytes.Buffer
		if err := md.Convert(payload, &buf); err != nil {
			return nil, err
		}

		html := buf.String()
		if html == "" {
			return nil, nil
		}

		return []core.Node{
			{Type: "html", Content: html, Raw: true},
		}, nil
	})
}
