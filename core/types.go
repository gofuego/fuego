// Package core defines the fundamental types shared across the Fuego engine.
// Both the public engine package and internal packages import from here,
// avoiding import cycles.
package core

// Envelope encapsulates the metadata parsed from content file frontmatter.
type Envelope = map[string]any

// Node represents a single element in the universal AST tree.
type Node struct {
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Content    string         `json:"content,omitempty"`
	Children   []Node         `json:"children,omitempty"`
}

// Parser defines the interface for content parsers.
type Parser interface {
	Type() string
	Parse(rawPayload []byte, meta Envelope) ([]Node, error)
}
