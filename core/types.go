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

// Page is the central data carrier flowing through the pipeline.
// It is progressively enriched by each phase: DISCOVER sets paths,
// PARSE sets envelope/nodes, ROUTE sets URL, INDEX may annotate further.
type Page struct {
	SourcePath string   // absolute path to the source file
	RelPath    string   // path relative to content dir
	Ext        string   // file extension without dot
	Envelope   Envelope // parsed frontmatter metadata
	Nodes      []Node   // parsed AST body
	URL        string   // resolved output URL (set by route phase)
	Layout     string   // layout name from envelope
	Type       string   // content type (from envelope or extension)
	IsRaw      bool     // true if no parser matched (raw passthrough)
}
